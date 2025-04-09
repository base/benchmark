package portconfig

type NodePorts struct {
	HTTP    int
	AuthRPC int
	Metrics int
}

type Config struct {
	Geth NodePorts
	Reth NodePorts
}

const (
	DefaultGethHTTPPort    = 8545
	DefaultGethAuthRPCPort = 8551
	DefaultGethMetricsPort = 8080

	DefaultRethHTTPPort    = 9545
	DefaultRethAuthRPCPort = 9551
	DefaultRethMetricsPort = 9080
)

func DefaultConfig() *Config {
	return &Config{
		Geth: NodePorts{
			HTTP:    DefaultGethHTTPPort,
			AuthRPC: DefaultGethAuthRPCPort,
			Metrics: DefaultGethMetricsPort,
		},
		Reth: NodePorts{
			HTTP:    DefaultRethHTTPPort,
			AuthRPC: DefaultRethAuthRPCPort,
			Metrics: DefaultRethMetricsPort,
		},
	}
}
