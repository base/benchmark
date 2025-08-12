package config

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
)

// FlagsConfig holds all configuration for the application when using CLI flags
type FlagsConfig struct {
	Port           string
	S3Bucket       string
	S3Region       string
	CacheTTL       time.Duration
	EnableCache    bool
	AllowedOrigins []string
	LogLevel       string
}

// NewConfigFromFlags creates a FlagsConfig from CLI context
func NewConfigFromFlags(ctx *cli.Context) (*FlagsConfig, error) {
	cacheTTL, err := time.ParseDuration(ctx.String("cache-ttl"))
	if err != nil {
		return nil, err
	}

	return &FlagsConfig{
		Port:           ctx.String("port"),
		S3Bucket:       ctx.String("s3-bucket"),
		S3Region:       ctx.String("s3-region"),
		CacheTTL:       cacheTTL,
		EnableCache:    ctx.Bool("enable-cache"),
		AllowedOrigins: strings.Split(ctx.String("allowed-origins"), ","),
		LogLevel:       ctx.String("log-level"),
	}, nil
}

// Validate checks if the configuration is valid
func (c *FlagsConfig) Validate() error {
	if c.S3Bucket == "" {
		return errors.New("BASE_BENCH_API_S3_BUCKET environment variable is required")
	}
	return nil
}

// CORS configures CORS middleware based on allowed origins
func (c *FlagsConfig) CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()

	if len(c.AllowedOrigins) == 1 && c.AllowedOrigins[0] == "*" {
		config.AllowAllOrigins = true
	} else {
		config.AllowOrigins = c.AllowedOrigins
	}

	config.AllowMethods = []string{"GET", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}
