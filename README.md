# `base-bench`

This is a tool that allows you to run benchmarks on Ethereum execution clients and analyze
performance impact as it relates to Optimism L2 networks.

## Build

```
make build
```

## Usage

Use Go to build base-bench

```
NAME:
   base-bench run - run benchmark

USAGE:
   base-bench run [command options]

DESCRIPTION:
   Runs benchmarks according to the specified config.

OPTIONS:
   
          --config value                                                         ($BASE_BENCH_CONFIG)
                Config Path
   
          --root-dir value                                                       ($BASE_BENCH_ROOT_DIR)
                Root Directory
   
          --reth-bin value                    (default: "reth")                  ($BASE_BENCH_RETH_BIN)
                Reth binary path
   
          --geth-bin value                    (default: "geth")                  ($BASE_BENCH_GETH_BIN)
                Geth binary path
   
          --help, -h                          (default: false)                  
                show help
```