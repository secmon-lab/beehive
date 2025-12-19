package cli

import (
	"context"
	"fmt"
	"net/http"

	httpctrl "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdServe() *cli.Command {
	var addr string
	var enableGraphiQL bool

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Start HTTP server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "addr",
				Usage:       "HTTP server address",
				Value:       ":8080",
				Sources:     cli.EnvVars("BEEHIVE_ADDR"),
				Destination: &addr,
			},
			&cli.BoolFlag{
				Name:        "graphiql",
				Usage:       "Enable GraphiQL playground",
				Value:       true,
				Sources:     cli.EnvVars("BEEHIVE_GRAPHIQL"),
				Destination: &enableGraphiQL,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Initialize repository (using memory for now)
			repo := memory.New()

			// Initialize use cases
			uc := usecase.New(repo)

			// Create HTTP server
			server := httpctrl.New(repo, uc, httpctrl.WithGraphiQL(enableGraphiQL))

			logging.Default().Info("Starting HTTP server", "addr", addr, "graphiql", enableGraphiQL)
			if err := http.ListenAndServe(addr, server); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			return nil
		},
	}
}
