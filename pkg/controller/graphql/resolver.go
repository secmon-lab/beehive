package graphql

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/usecase"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	repo         interfaces.Repository
	uc           *usecase.UseCases
	fetchUseCase *usecase.FetchUseCase
	sourcesMap   map[string]model.Source
}

func NewResolver(repo interfaces.Repository, uc *usecase.UseCases, fetchUseCase *usecase.FetchUseCase, sourcesConfigPath string) (*Resolver, error) {
	// Initialize empty sources map
	sourcesMap := make(map[string]model.Source)

	// Load and cache sources configuration if path is provided
	if sourcesConfigPath != "" {
		cfg, err := config.LoadConfig(sourcesConfigPath)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to load sources config")
		}

		// Add RSS sources
		for id, src := range cfg.RSS {
			sourcesMap[id] = model.Source{
				Type:    model.SourceTypeRSS,
				URL:     src.URL,
				Tags:    ensureStringSlice(src.Tags.Strings()),
				Enabled: !src.Disabled,
				RSSConfig: &model.RSSConfig{
					MaxArticles: src.MaxArticles,
				},
			}
		}

		// Add Feed sources
		for id, src := range cfg.Feed {
			sourcesMap[id] = model.Source{
				Type:    model.SourceTypeFeed,
				URL:     src.GetURL(),
				Tags:    ensureStringSlice(src.Tags.Strings()),
				Enabled: !src.Disabled,
				FeedConfig: &model.FeedConfig{
					Schema:   string(src.Schema),
					MaxItems: src.MaxItems,
				},
			}
		}
	}

	return &Resolver{
		repo:         repo,
		uc:           uc,
		fetchUseCase: fetchUseCase,
		sourcesMap:   sourcesMap,
	}, nil
}

// Repository returns the repository instance
func (r *Resolver) Repository() interfaces.Repository {
	return r.repo
}
