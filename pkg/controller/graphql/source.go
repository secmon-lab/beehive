package graphql

import (
	"context"
	"errors"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	gqlmodel "github.com/secmon-lab/beehive/pkg/domain/model/graphql"
)

func (r *queryResolver) ListSources(ctx context.Context) ([]*gqlmodel.Source, error) {
	// Load sources configuration
	sources, err := config.LoadSources(r.sourcesConfigPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load sources config")
	}

	var result []*gqlmodel.Source
	for id, src := range sources.Sources {
		gqlSrc := &gqlmodel.Source{
			ID:      id,
			Type:    src.Type,
			URL:     src.URL,
			Tags:    src.Tags,
			Enabled: src.Enabled,
		}

		// Try to fetch source state
		state, err := r.repo.GetState(ctx, id)
		if err != nil && !errors.Is(err, interfaces.ErrSourceStateNotFound) {
			return nil, goerr.Wrap(err, "failed to get source state", goerr.V("source_id", id))
		}

		if state != nil {
			gqlSrc.State = toGraphQLSourceState(state)
		}

		result = append(result, gqlSrc)
	}

	return result, nil
}

func toGraphQLSourceState(state *model.SourceState) *gqlmodel.SourceState {
	var lastFetchedAt *time.Time
	var lastItemID *string
	var lastItemDate *time.Time
	var lastError *string

	if !state.LastFetchedAt.IsZero() {
		lastFetchedAt = &state.LastFetchedAt
	}
	if state.LastItemID != "" {
		lastItemID = &state.LastItemID
	}
	if !state.LastItemDate.IsZero() {
		lastItemDate = &state.LastItemDate
	}
	if state.LastError != "" {
		lastError = &state.LastError
	}

	return &gqlmodel.SourceState{
		SourceID:      state.SourceID,
		LastFetchedAt: lastFetchedAt,
		LastItemID:    lastItemID,
		LastItemDate:  lastItemDate,
		ItemCount:     int(state.ItemCount),
		ErrorCount:    int(state.ErrorCount),
		LastError:     lastError,
		UpdatedAt:     state.UpdatedAt,
	}
}
