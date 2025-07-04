package options

// GethOptions are options that are specified at runtime for the geth client.
type GethOptions struct {
	// GethBin is the path to the geth binary to use in tests.
	GethBin         string
	GethHttpPort    int
	GethAuthRpcPort int
	GethMetricsPort int
	SkipInit        bool
}
