package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"benchmark-report-api/internal/config"
	"benchmark-report-api/internal/handlers"
	"benchmark-report-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

// autopopulated by the Makefile
var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

func Main() cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		logConfig := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logConfig)
		oplog.SetGlobalLogHandler(l.Handler())

		cfg, err := config.NewConfigFromFlags(cliCtx)
		if err != nil {
			l.Error("Error creating configuration from flags", "error", err)
			return nil, err
		}

		if err := cfg.Validate(); err != nil {
			l.Error("Invalid configuration", "error", err)
			return nil, err
		}

		opservice.ValidateEnvVars(config.EnvVarPrefix, config.CLIFlags(), l)

		// Setup logging using the config method
		l.Info("Starting benchmark report API server",
			"port", cfg.Port,
			"bucket", cfg.S3Bucket,
			"region", cfg.S3Region,
			"cache", cfg.EnableCache,
			"cacheTTL", cfg.CacheTTL)

		// Initialize services
		cache := services.NewMemoryCache(cfg.CacheTTL, l)
		if !cfg.EnableCache {
			cache = services.NewMemoryCache(0, l) // Disable caching
		}

		s3Service, err := services.NewS3Service(cfg.S3Bucket, cfg.S3Region, cache, l)
		if err != nil {
			l.Error("Failed to initialize S3 service", "error", err)
			return nil, err
		}

		// Setup Gin
		if cfg.LogLevel != "debug" {
			gin.SetMode(gin.ReleaseMode)
		}

		router := gin.New()
		router.Use(gin.Logger())
		router.Use(gin.Recovery())
		router.Use(cfg.CORS())

		// Setup routes
		setupRoutes(router, s3Service, l)

		// Configure server
		server := &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		return &ServerService{server: server, logger: l}, nil
	}
}

type ServerService struct {
	server  *http.Server
	logger  log.Logger
	stopped atomic.Bool
}

func (s *ServerService) Start(ctx context.Context) error {
	s.logger.Info("Server starting", "addr", s.server.Addr)

	// Create a channel to listen for OS signals (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan) // Clean up signal notification

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for either a signal or server error
	select {
	case err := <-serverErr:
		s.logger.Error("Server failed to start", "error", err)
		return err
	case sig := <-sigChan:
		s.logger.Info("Received shutdown signal", "signal", sig)

		// Create a context with timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.logger.Info("Shutting down server gracefully...")
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server forced to shutdown", "error", err)
			return err
		}

		s.logger.Info("Server shutdown complete")
		s.stopped.Store(true)
		return nil
	case <-ctx.Done():
		s.logger.Info("Context cancelled, shutting down server")

		// Create a context with timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server forced to shutdown", "error", err)
			return err
		}

		s.logger.Info("Server shutdown complete")
		s.stopped.Store(true)
		return ctx.Err()
	}
}

func (s *ServerService) Stop(ctx context.Context) error {
	s.logger.Info("Server stopping")
	s.stopped.Store(true)
	return s.server.Shutdown(ctx)
}

func (s *ServerService) Stopped() bool {
	return s.stopped.Load()
}

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(config.CLIFlags())
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "benchmark-report-api"
	app.Usage = "Benchmark Report API Server"
	app.Description = "REST API server for serving benchmark data from AWS S3 storage"

	app.Action = cliapp.LifecycleCmd(Main())
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// setupRoutes configures all API routes
func setupRoutes(router *gin.Engine, s3Service *services.S3Service, l log.Logger) {
	api := router.Group("/api/v1")
	{
		api.GET("/health", handlers.Health)
		api.GET("/metadata", handlers.MetadataHandler(s3Service, l))
		api.GET("/metrics/:runId/:outputDir/:nodeType", handlers.MetricsHandler(s3Service, l))
	}
}
