package flags

import "github.com/urfave/cli/v2"

var ExportFlags = []cli.Flag{
	OutputDirFlag,
	S3BucketFlag,
}
