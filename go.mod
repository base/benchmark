module github.com/base/base-bench

go 1.22.8

require (
	github.com/ethereum-optimism/optimism v1.12.0
	github.com/ethereum/go-ethereum v1.15.3
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli/v2 v2.27.5
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/ethereum/go-ethereum => github.com/ethereum-optimism/op-geth v1.101411.2
