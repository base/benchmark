package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Configuration for the server
type Config struct {
	Port           string
	S3Bucket       string
	S3Region       string
	CacheTTL       time.Duration
	EnableCache    bool
	AllowedOrigins []string
	LogLevel       string
}

// S3Service handles interactions with AWS S3
type S3Service struct {
	client     *s3.S3
	bucketName string
	cache      *MemoryCache
}

// Simple in-memory cache for production optimization
type MemoryCache struct {
	data map[string]CacheItem
	ttl  time.Duration
}

type CacheItem struct {
	Data      []byte
	ExpiresAt time.Time
}

// BenchmarkRuns represents the metadata structure
type BenchmarkRuns struct {
	Runs      []BenchmarkRun `json:"runs"`
	CreatedAt *time.Time     `json:"createdAt"`
}

type BenchmarkRun struct {
	ID              string                 `json:"id"`
	SourceFile      string                 `json:"sourceFile"`
	OutputDir       string                 `json:"outputDir"`
	BucketPath      string                 `json:"bucketPath,omitempty"`
	TestName        string                 `json:"testName"`
	TestDescription string                 `json:"testDescription"`
	TestConfig      map[string]interface{} `json:"testConfig"`
	Result          interface{}            `json:"result"`
	Thresholds      interface{}            `json:"thresholds"`
	CreatedAt       *time.Time             `json:"createdAt"`
}

// NewConfig creates configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		S3Bucket:       getEnv("S3_BUCKET", ""),
		S3Region:       getEnv("AWS_REGION", "us-east-1"),
		CacheTTL:       getDurationEnv("CACHE_TTL", 5*time.Minute),
		EnableCache:    getBoolEnv("ENABLE_CACHE", true),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
}

// NewS3Service creates a new S3 service instance
func NewS3Service(bucketName, region string, cache *MemoryCache) (*S3Service, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Service{
		client:     s3.New(sess),
		bucketName: bucketName,
		cache:      cache,
	}, nil
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	cache := &MemoryCache{
		data: make(map[string]CacheItem),
		ttl:  ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves data from cache
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	item, exists := c.data[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		delete(c.data, key)
		return nil, false
	}
	return item.Data, true
}

// Set stores data in cache
func (c *MemoryCache) Set(key string, data []byte) {
	c.data[key] = CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// cleanup removes expired items
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.ExpiresAt) {
				delete(c.data, key)
			}
		}
	}
}

// GetObject retrieves an object from S3 with caching
func (s *S3Service) GetObject(key string) ([]byte, error) {
	// Check cache first
	if cached, hit := s.cache.Get(key); hit {
		log.Debug().Str("key", key).Msg("Cache hit")
		return cached, nil
	}

	log.Debug().Str("key", key).Str("bucket", s.bucketName).Msg("Fetching from S3")

	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer result.Body.Close()

	// Read the entire body
	data := make([]byte, *result.ContentLength)
	_, err = result.Body.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	// Cache the result
	s.cache.Set(key, data)

	return data, nil
}

// GetMetadata retrieves and parses the metadata.json from S3
func (s *S3Service) GetMetadata() (*BenchmarkRuns, error) {
	data, err := s.GetObject("metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata BenchmarkRuns
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// GetMetrics retrieves metrics data for a specific run and node type
func (s *S3Service) GetMetrics(runID, outputDir, nodeType string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/metrics-%s.json", runID, outputDir, nodeType)
	return s.GetObject(key)
}

// API Handlers

// healthHandler provides a health check endpoint
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "benchmark-report-api",
	})
}

// metadataHandler serves the benchmark metadata
func metadataHandler(s3Service *S3Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata, err := s3Service.GetMetadata()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get metadata")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve metadata",
			})
			return
		}

		c.Header("Cache-Control", "public, max-age=300") // 5 minutes
		c.JSON(http.StatusOK, metadata)
	}
}

// metricsHandler serves metrics data for a specific run and node type
func metricsHandler(s3Service *S3Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		runID := c.Param("runId")
		outputDir := c.Param("outputDir")
		nodeType := c.Param("nodeType")

		if runID == "" || outputDir == "" || nodeType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "runId, outputDir, and nodeType are required",
			})
			return
		}

		data, err := s3Service.GetMetrics(runID, outputDir, nodeType)
		if err != nil {
			log.Error().Err(err).
				Str("runId", runID).
				Str("outputDir", outputDir).
				Str("nodeType", nodeType).
				Msg("Failed to get metrics")

			c.JSON(http.StatusNotFound, gin.H{
				"error": "Metrics not found",
			})
			return
		}

		c.Header("Cache-Control", "public, max-age=3600") // 1 hour
		c.Header("Content-Type", "application/json")
		c.Data(http.StatusOK, "application/json", data)
	}
}

// setupLogging configures structured logging
func setupLogging(level string) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

// setupCORS configures CORS middleware
func setupCORS(allowedOrigins []string) gin.HandlerFunc {
	config := cors.DefaultConfig()

	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		config.AllowAllOrigins = true
	} else {
		config.AllowOrigins = allowedOrigins
	}

	config.AllowMethods = []string{"GET", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}

// gracefulShutdown handles graceful server shutdown
func gracefulShutdown(server *http.Server) {
	c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info().Msg("Received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		} else {
			log.Info().Msg("Server shutdown gracefully")
		}
	}()
}

func main() {
	config := NewConfig()

	// Validate required configuration
	if config.S3Bucket == "" {
		log.Fatal().Msg("S3_BUCKET environment variable is required")
	}

	setupLogging(config.LogLevel)

	log.Info().
		Str("port", config.Port).
		Str("bucket", config.S3Bucket).
		Str("region", config.S3Region).
		Bool("cache", config.EnableCache).
		Dur("cacheTTL", config.CacheTTL).
		Msg("Starting benchmark report API server")

	// Initialize cache
	var cache *MemoryCache
	if config.EnableCache {
		cache = NewMemoryCache(config.CacheTTL)
	} else {
		cache = NewMemoryCache(0) // No caching
	}

	// Initialize S3 service
	s3Service, err := NewS3Service(config.S3Bucket, config.S3Region, cache)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize S3 service")
	}

	// Setup Gin
	if config.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(setupCORS(config.AllowedOrigins))

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/health", healthHandler)
		api.GET("/metadata", metadataHandler(s3Service))
		api.GET("/metrics/:runId/:outputDir/:nodeType", metricsHandler(s3Service))
	}

	// Start server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	gracefulShutdown(server)

	log.Info().Str("addr", server.Addr).Msg("Server starting")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
