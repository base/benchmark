package flags

import (
	"github.com/urfave/cli/v2"
)

const (
	SrcTagFlagName      = "src-tag"
	DestTagFlagName     = "dest-tag"
	NoConfirmFlagName   = "no-confirm"
	S3DirectoryFlagName = "s3-directory"
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

	S3DirectoryFlag = &cli.StringFlag{
		Name:  S3DirectoryFlagName,
		Usage: "S3 directory/prefix to download from (use '.' for root directory)",
	}
)

// ImportRunsFlags contains the list of flags for the import-runs command
var ImportRunsFlags = []cli.Flag{
	OutputDirFlag,
	SrcTagFlag,
	DestTagFlag,
	NoConfirmFlag,
	S3BucketFlag,
	S3DirectoryFlag,
}
