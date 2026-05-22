// Package cli wires the urfave/cli/v3 command tree. The root command
// configures logging and delegates to one of the subcommands (serve,
// validate, fetch). Each subcommand lives in its own file.
package cli

import (
	"context"
	"os"

	"github.com/secmon-lab/beehive/pkg/cli/config"
	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Configs is the aggregate of all subsystem configs. Each subcommand
// declares only the flags it needs by listing the relevant *.Flags()
// slices.
type Configs struct {
	Logger    config.Logger
	HTTP      config.HTTP
	Repo      config.Repo
	Firestore config.Firestore
	Sources   config.Sources
	LLM       config.LLM
}

// Run is the entrypoint called from main. version is injected through
// build flags.
func Run(ctx context.Context, args []string, version string) error {
	bhhttp.Version = version

	cfg := &Configs{}

	app := &cli.Command{
		Name:    "beehive",
		Usage:   "Crawl security blogs / IoC feeds, extract IoCs, expose via API.",
		Version: version,
		Flags:   cfg.Logger.Flags(),
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			logger := cfg.Logger.Build(os.Stderr)
			logging.SetDefault(logger)
			return logging.With(ctx, logger), nil
		},
		Commands: []*cli.Command{
			cmdServe(cfg),
			cmdValidate(cfg),
			cmdFetch(cfg),
		},
	}

	return app.Run(ctx, args)
}
