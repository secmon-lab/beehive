package cli

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/controller/graphql"
	httpctrl "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdServe() *cli.Command {
	var (
		addr           string
		enableGraphiQL bool
		configPath     string
		firestoreCfg   config.Firestore
	)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Start HTTP server",
		Flags: append(firestoreCfg.Flags(),
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
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Path to sources configuration file",
				Value:       "config/config.toml",
				Destination: &configPath,
				Sources:     cli.EnvVars("BEEHIVE_CONFIG"),
			},
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.Default()

			// Initialize repository
			var repo interfaces.Repository
			if firestoreCfg.ProjectID != "" {
				// Use Firestore
				opts := []firestoreRepo.Option{}
				if firestoreCfg.DatabaseID != "" {
					opts = append(opts, firestoreRepo.WithDatabaseID(firestoreCfg.DatabaseID))
				}

				fsRepo, err := firestoreRepo.New(ctx, firestoreCfg.ProjectID, opts...)
				if err != nil {
					return goerr.Wrap(err, "failed to create Firestore repository",
						goerr.V("project_id", firestoreCfg.ProjectID),
						goerr.V("database_id", firestoreCfg.DatabaseID))
				}
				defer func() {
					if err := fsRepo.Close(); err != nil {
						logger.Error("failed to close Firestore client", "error", err)
					}
				}()
				repo = fsRepo
				logger.Info("using Firestore repository", "project_id", firestoreCfg.ProjectID, "database_id", firestoreCfg.DatabaseID)
			} else {
				// Use in-memory repository with sample data
				memRepo := memory.New()
				if err := addSampleData(ctx, memRepo); err != nil {
					logger.Warn("Failed to add sample data", "error", err)
				}
				repo = memRepo
				logger.Info("using in-memory repository with sample data")
			}

			// Initialize use cases
			uc := usecase.New(repo)

			// Initialize GraphQL resolver
			gqlResolver := graphql.NewResolver(repo, uc, configPath)

			// Create HTTP server
			handler := httpctrl.New(gqlResolver, httpctrl.WithGraphiQL(enableGraphiQL))
			server := &http.Server{
				Addr:              addr,
				Handler:           handler,
				ReadHeaderTimeout: 30 * time.Second,
			}

			// Setup signal handling for graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			// Start server in goroutine
			errCh := make(chan error, 1)
			go func() {
				logging.Default().Info("Starting HTTP server", "addr", addr, "graphiql", enableGraphiQL)
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errCh <- goerr.Wrap(err, "failed to start server")
				}
			}()

			// Wait for shutdown signal or server error
			select {
			case err := <-errCh:
				return err
			case sig := <-sigCh:
				logging.Default().Info("Received shutdown signal", "signal", sig)

				// Create shutdown context with timeout
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Attempt graceful shutdown
				if err := server.Shutdown(shutdownCtx); err != nil {
					return goerr.Wrap(err, "failed to shutdown server gracefully")
				}

				logging.Default().Info("Server shutdown completed")
				return nil
			}
		},
	}
}

func addSampleData(ctx context.Context, repo interfaces.Repository) error {
	now := time.Now()

	sampleIoCs := []*model.IoC{
		{
			ID:          model.GenerateID("sample-source", model.IoCTypeIPv4, "192.168.1.100", "sample-1"),
			SourceID:    "sample-source",
			SourceType:  "rss",
			Type:        model.IoCTypeIPv4,
			Value:       "192.168.1.100",
			Description: "Suspicious IP address from malware campaign",
			SourceURL:   "https://example.com/blog/malware-analysis",
			Context:     "Observed in command and control traffic",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-24 * time.Hour),
			UpdatedAt:   now,
		},
		{
			ID:          model.GenerateID("sample-source", model.IoCTypeDomain, "malicious.example.com", "sample-2"),
			SourceID:    "sample-source",
			SourceType:  "rss",
			Type:        model.IoCTypeDomain,
			Value:       "malicious.example.com",
			Description: "Phishing domain targeting financial institutions",
			SourceURL:   "https://example.com/blog/phishing-alert",
			Context:     "Used in credential harvesting campaign",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-12 * time.Hour),
			UpdatedAt:   now,
		},
		{
			ID:          model.GenerateID("sample-source", model.IoCTypeSHA256, "d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2", "sample-3"),
			SourceID:    "sample-source",
			SourceType:  "feed",
			Type:        model.IoCTypeSHA256,
			Value:       "d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2",
			Description: "Ransomware executable",
			SourceURL:   "https://example.com/threat-feed",
			Context:     "Part of ransomware family XYZ",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-48 * time.Hour),
			UpdatedAt:   now,
		},
		{
			ID:          model.GenerateID("abuse-ch", model.IoCTypeURL, "http://evil.example.org/payload.exe", "urlhaus-123"),
			SourceID:    "abuse-ch",
			SourceType:  "feed",
			Type:        model.IoCTypeURL,
			Value:       "http://evil.example.org/payload.exe",
			Description: "Malware download URL",
			SourceURL:   "https://urlhaus.abuse.ch/",
			Context:     "URLhaus feed entry",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-6 * time.Hour),
			UpdatedAt:   now,
		},
		{
			ID:          model.GenerateID("sample-source", model.IoCTypeIPv4, "10.0.0.50", "sample-4"),
			SourceID:    "sample-source",
			SourceType:  "rss",
			Type:        model.IoCTypeIPv4,
			Value:       "10.0.0.50",
			Description: "Bot network controller IP",
			SourceURL:   "https://example.com/blog/botnet-analysis",
			Context:     "Identified as part of botnet infrastructure",
			Status:      model.IoCStatusInactive,
			FirstSeenAt: now.Add(-168 * time.Hour),
			UpdatedAt:   now.Add(-72 * time.Hour),
		},
	}

	for _, ioc := range sampleIoCs {
		if err := repo.UpsertIoC(ctx, ioc); err != nil {
			return goerr.Wrap(err, "failed to add sample IoC", goerr.V("id", ioc.ID))
		}
	}

	logging.Default().Info("Added sample IoC data", "count", len(sampleIoCs))
	return nil
}
