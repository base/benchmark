# Benchmark Report API

A Go-based REST API for serving benchmark data from AWS S3 storage.

## Project Structure

This project follows the standard Go project layout:

```
benchmark/report/backend/
├── cmd/
│   └── api/                  # Application entrypoints
│       └── main.go          # Main application entry point
├── internal/                # Private application code
│   ├── config/              # Configuration, middleware, and errors
│   │   └── config.go        # Config, CORS, logging, and error handling
│   ├── handlers/            # HTTP handlers
│   │   ├── health.go        # Health check endpoint
│   │   ├── metadata.go      # Metadata endpoint handler
│   │   └── metrics.go       # Metrics endpoint handler
│   └── services/            # Business logic and data models
│       ├── cache.go         # In-memory caching with CacheItem model
│       └── s3.go            # AWS S3 service with Benchmark models
├── bin/                     # Compiled binaries (gitignored)
├── flags/                   # CLI flags for the API server
│   └── flags.go            # Command-line flag definitions
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
├── Makefile                 # Build automation
├── Dockerfile               # Docker container definition
├── docker-compose.yml       # Docker Compose configuration
└── README.md               # This file
```

## Features

- **REST API** for benchmark data retrieval
- **AWS S3 integration** with intelligent caching
- **Structured logging** with configurable levels
- **Health checks** for monitoring
- **CORS support** for frontend integration
- **Graceful shutdown** handling
- **Docker support** for containerized deployment

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `S3_BUCKET` | AWS S3 bucket name | **Required** |
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | AWS access key | From AWS credentials |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | From AWS credentials |
| `CACHE_TTL` | Cache time-to-live | `5m` |
| `ENABLE_CACHE` | Enable/disable caching | `true` |
| `ALLOWED_ORIGINS` | CORS allowed origins (comma-separated) | `*` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## API Endpoints

### Health Check
```
GET /api/v1/health
```
Returns the service health status.

### Metadata
```
GET /api/v1/metadata
```
Returns benchmark run metadata from S3.

### Metrics
```
GET /api/v1/metrics/:runId/:outputDir/:nodeType
```
Returns metrics data for a specific benchmark run and node type.

## Development

### Prerequisites

- Go 1.23 or later
- Docker (optional, for containerized development)
- Make (optional, for build automation)

### Quick Start

1. **Set up environment variables:**
   ```bash
   export S3_BUCKET=your-bucket-name
   export AWS_REGION=us-east-1
   # ... other environment variables
   ```

2. **Run in development mode:**
   ```bash
   make run-backend
   ```

## Architecture

### Layers

1. **Handlers Layer** (`internal/handlers/`): HTTP request handling and response formatting
2. **Services Layer** (`internal/services/`): Business logic, external service integration, and data models
3. **Configuration Layer** (`internal/config/`): Application configuration, middleware, and error handling

### Key Components

- **S3Service**: Handles AWS S3 interactions with intelligent caching (includes BenchmarkRuns/BenchmarkRun models)
- **MemoryCache**: Provides in-memory caching with TTL support (includes CacheItem model)
- **Configuration**: Environment-based configuration with validation, CORS, and logging setup
- **Handlers**: Clean HTTP handlers following single responsibility principle
- **Flags**: CLI flag definitions for server configuration
