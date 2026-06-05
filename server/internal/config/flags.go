package config

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvVarPrefix = "BASE_BENCH_API"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

const (
	// Default values for API server
	DefaultPort           = "8080"
	DefaultS3Region       = "us-east-1"
	DefaultCacheTTL       = "5m"
	DefaultEnableCache    = true
	DefaultAllowedOrigins = "*"
	DefaultLogLevel       = "debug"
)

var (
	PortFlag = &cli.StringFlag{
		Name:    "port",
		Usage:   "API server port",
		Value:   DefaultPort,
		EnvVars: prefixEnvVars("PORT"),
	}
	S3BucketFlag = &cli.StringFlag{
		Name:    "s3-bucket",
		Usage:   "AWS S3 bucket name for benchmark data (required unless --local-dir is set)",
		EnvVars: prefixEnvVars("S3_BUCKET"),
	}
	S3RegionFlag = &cli.StringFlag{
		Name:    "s3-region",
		Usage:   "AWS S3 region",
		Value:   DefaultS3Region,
		EnvVars: prefixEnvVars("AWS_REGION"),
	}
	S3EndpointFlag = &cli.StringFlag{
		Name:    "s3-endpoint",
		Usage:   "Override S3 endpoint URL (use with MinIO or other S3-compatible stores; leave empty for AWS)",
		Value:   "",
		EnvVars: prefixEnvVars("S3_ENDPOINT"),
	}
	LocalDirFlag = &cli.StringFlag{
		Name:    "local-dir",
		Usage:   "Read benchmark output from this local directory instead of S3. Mutually exclusive with --s3-bucket. Use the directory written by 'base-bench run --output-dir <dir>'.",
		EnvVars: prefixEnvVars("LOCAL_DIR"),
	}
	CacheTTLFlag = &cli.StringFlag{
		Name:    "cache-ttl",
		Usage:   "Cache time-to-live duration (e.g., 5m, 1h)",
		Value:   DefaultCacheTTL,
		EnvVars: prefixEnvVars("CACHE_TTL"),
	}
	EnableCacheFlag = &cli.BoolFlag{
		Name:    "enable-cache",
		Usage:   "Enable in-memory caching",
		Value:   DefaultEnableCache,
		EnvVars: prefixEnvVars("ENABLE_CACHE"),
	}
	AllowedOriginsFlag = &cli.StringFlag{
		Name:    "allowed-origins",
		Usage:   "CORS allowed origins (comma-separated)",
		Value:   DefaultAllowedOrigins,
		EnvVars: prefixEnvVars("ALLOWED_ORIGINS"),
	}
	LogLevelFlag = &cli.StringFlag{
		Name:    "log-level",
		Usage:   "Log level (debug, info, warn, error)",
		Value:   DefaultLogLevel,
		EnvVars: prefixEnvVars("LOG_LEVEL"),
	}
)

func CLIFlags() []cli.Flag {
	Flags := []cli.Flag{
		PortFlag,
		S3BucketFlag,
		S3RegionFlag,
		S3EndpointFlag,
		LocalDirFlag,
		CacheTTLFlag,
		EnableCacheFlag,
		AllowedOriginsFlag,
		LogLevelFlag,
	}
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	return Flags
}
