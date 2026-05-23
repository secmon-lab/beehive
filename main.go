package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/secmon-lab/beehive/pkg/cli"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// version is overridden via -ldflags="-X main.version=...".
var version = "dev"

func main() {
	ctx := context.Background()

	// Pre-install a stderr console logger so any error returned from
	// cli.Run before the `Before` hook runs (e.g. flag parsing failures)
	// still surfaces with the full goerr metadata via errutil.Handle.
	logging.SetDefault(logging.New(os.Stderr, slog.LevelInfo, logging.FormatConsole, true))

	if err := cli.Run(ctx, os.Args, version); err != nil {
		errutil.Handle(ctx, err)
		os.Exit(1)
	}
}
