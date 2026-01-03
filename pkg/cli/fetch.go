package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdFetch() *cli.Command {
	var (
		llmCfg       config.LLM
		firestoreCfg config.Firestore
		configPath   string
		tags         []string
		dryRun       bool
	)

	return &cli.Command{
		Name:  "fetch",
		Usage: "Fetch IoCs from configured sources",
		Flags: append(append(llmCfg.Flags(), firestoreCfg.Flags()...),
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Path to configuration file",
				Value:       "config/config.toml",
				Destination: &configPath,
				Sources:     cli.EnvVars("BEEHIVE_CONFIG"),
			},
			&cli.StringSliceFlag{
				Name:        "tag",
				Aliases:     []string{"t"},
				Usage:       "Filter sources by tag (can be specified multiple times)",
				Destination: &tags,
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "Dry run mode (fetch but don't save to database)",
				Destination: &dryRun,
			},
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.Default()

			// Find and load configuration file
			cfgPath, err := findConfigFile(configPath)
			if err != nil {
				return goerr.Wrap(err, "failed to find config file",
					goerr.V("config_path", configPath))
			}

			logger.Info("loading configuration", "path", cfgPath)

			cfg, err := config.LoadConfig(cfgPath)
			if err != nil {
				return goerr.Wrap(err, "failed to load config")
			}

			logger.Info("loaded configuration",
				"rss_sources", len(cfg.RSS),
				"feed_sources", len(cfg.Feed))

			// Initialize repository
			var repo *firestoreRepo.Firestore
			var memRepo *memory.Memory

			if dryRun {
				// Use in-memory storage for dry-run
				memRepo = memory.New()
				logger.Info("using in-memory storage (dry-run mode)")
			} else {
				// Use Firestore for production
				if firestoreCfg.ProjectID == "" {
					return goerr.New("firestore-project-id is required for production mode")
				}

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
				logger.Info("using Firestore storage",
					"project_id", firestoreCfg.ProjectID,
					"database_id", firestoreCfg.DatabaseID)
			}

			// Create LLM client
			llmClient, err := llmCfg.NewLLMClient(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to create LLM client")
			}

			logger.Info("initialized LLM client", "provider", llmCfg.Provider, "model", llmCfg.Model)

			// Convert config to sources map (model.Source)
			sourcesMap := convertConfigToSourcesMap(cfg)
			logger.Info("converted sources from config", "total", len(sourcesMap))

			// Initialize FetchUseCase
			var fetchUC *usecase.FetchUseCase
			if dryRun {
				fetchUC = usecase.NewFetchUseCase(memRepo, llmClient)
			} else {
				fetchUC = usecase.NewFetchUseCase(repo, llmClient)
			}

			// Execute fetch via usecase
			logger.Info("starting fetch operation", "sources", len(sourcesMap), "tags", tags, "dry_run", dryRun)

			stats, err := fetchUC.FetchAllSources(ctx, sourcesMap, tags)
			if err != nil {
				return goerr.Wrap(err, "fetch operation failed")
			}

			// Print results
			printFetchResults(stats)

			return nil
		},
	}
}

// findConfigFile searches for the configuration file in multiple locations
func findConfigFile(path string) (string, error) {
	// If absolute path is provided, use it directly
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", goerr.New("config file not found", goerr.V("path", path))
	}

	// Search in order of priority
	searchPaths := []string{
		path,                                    // Provided path (relative or default)
		filepath.Join("config", "config.toml"),  // ./config/config.toml
		filepath.Join("config", "sources.toml"), // ./config/sources.toml (legacy)
		filepath.Join(os.Getenv("HOME"), ".config", "beehive", "config.toml"), // ~/.config/beehive/config.toml
		"/etc/beehive/config.toml", // /etc/beehive/config.toml
	}

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); err == nil {
			return searchPath, nil
		}
	}

	return "", goerr.New("config file not found in any search path",
		goerr.V("searched_paths", searchPaths))
}

// convertConfigToSourcesMap converts config to model.Source map
func convertConfigToSourcesMap(cfg *config.Config) map[string]model.Source {
	sourcesMap := make(map[string]model.Source)

	// Add RSS sources
	for id, rssSrc := range cfg.RSS {
		if rssSrc.Disabled {
			continue
		}
		sourcesMap[id] = model.Source{
			Type:    model.SourceTypeRSS,
			URL:     rssSrc.URL,
			Tags:    rssSrc.Tags.Strings(),
			Enabled: !rssSrc.Disabled,
			RSSConfig: &model.RSSConfig{
				MaxArticles: rssSrc.MaxArticles,
			},
		}
	}

	// Add Feed sources
	for id, feedSrc := range cfg.Feed {
		if feedSrc.Disabled {
			continue
		}
		sourcesMap[id] = model.Source{
			Type:    model.SourceTypeFeed,
			URL:     feedSrc.GetURL(),
			Tags:    feedSrc.Tags.Strings(),
			Enabled: !feedSrc.Disabled,
			FeedConfig: &model.FeedConfig{
				Schema:   feedSrc.Schema.String(),
				MaxItems: feedSrc.MaxItems,
			},
		}
	}

	return sourcesMap
}

// printFetchResults prints the fetch results to stdout
func printFetchResults(histories []*model.History) {
	fmt.Println("\n=== Fetch Results ===")
	fmt.Println()

	totalItems := 0
	totalExtracted := 0
	totalCreated := 0
	totalUpdated := 0
	totalUnchanged := 0
	totalErrors := 0

	for _, h := range histories {
		fmt.Printf("Source: %s (%s)\n", h.SourceID, h.SourceType)
		fmt.Printf("  Items Fetched:   %d\n", h.ItemsFetched)
		fmt.Printf("  IoCs Extracted:  %d\n", h.IoCsExtracted)
		fmt.Printf("  IoCs Created:    %d\n", h.IoCsCreated)
		fmt.Printf("  IoCs Updated:    %d\n", h.IoCsUpdated)
		fmt.Printf("  IoCs Unchanged:  %d\n", h.IoCsUnchanged)
		fmt.Printf("  Errors:          %d\n", h.ErrorCount)
		fmt.Printf("  Processing Time: %v\n", h.ProcessingTime)
		fmt.Println()

		totalItems += h.ItemsFetched
		totalExtracted += h.IoCsExtracted
		totalCreated += h.IoCsCreated
		totalUpdated += h.IoCsUpdated
		totalUnchanged += h.IoCsUnchanged
		totalErrors += h.ErrorCount
	}

	fmt.Println("=== Summary ===")
	fmt.Printf("Total Sources Processed: %d\n", len(histories))
	fmt.Printf("Total Items Fetched:     %d\n", totalItems)
	fmt.Printf("Total IoCs Extracted:    %d\n", totalExtracted)
	fmt.Printf("Total IoCs Created:      %d\n", totalCreated)
	fmt.Printf("Total IoCs Updated:      %d\n", totalUpdated)
	fmt.Printf("Total IoCs Unchanged:    %d\n", totalUnchanged)
	fmt.Printf("Total Errors:            %d\n", totalErrors)
	fmt.Println()
}
