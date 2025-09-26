package flags

import (
	"github.com/urfave/cli/v2"
)

const (
	SrcTagFlagName    = "src-tag"
	DestTagFlagName   = "dest-tag"
	NoConfirmFlagName = "no-confirm"
)

var (
	SrcTagFlag = &cli.StringFlag{
		Name:  SrcTagFlagName,
		Usage: "Tag to apply to existing metadata runs (format: key=value)",
	}

	DestTagFlag = &cli.StringFlag{
		Name:  DestTagFlagName,
		Usage: "Tag to apply to imported metadata runs (format: key=value)",
	}

	NoConfirmFlag = &cli.BoolFlag{
		Name:  NoConfirmFlagName,
		Usage: "Skip confirmation prompts",
		Value: false,
	}
)

// ImportRunsFlags contains the list of flags for the import-runs command
var ImportRunsFlags = []cli.Flag{
	OutputDirFlag,
	SrcTagFlag,
	DestTagFlag,
	NoConfirmFlag,
}
