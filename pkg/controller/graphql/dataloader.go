package graphql

import (
	"context"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

type contextKey string

const (
	loadersKey contextKey = "dataloaders"
)

// Loaders contains all data loaders
type Loaders struct {
	SourceStateLoader *dataloader.Loader[string, *model.SourceState]
}

// NewLoaders creates a new set of data loaders
func NewLoaders(repo interfaces.Repository) *Loaders {
	return &Loaders{
		SourceStateLoader: dataloader.NewBatchedLoader(
			func(ctx context.Context, sourceIDs []string) []*dataloader.Result[*model.SourceState] {
				return batchGetSourceStates(ctx, repo, sourceIDs)
			},
		),
	}
}

// batchGetSourceStates fetches multiple source states in a single batch
func batchGetSourceStates(ctx context.Context, repo interfaces.Repository, sourceIDs []string) []*dataloader.Result[*model.SourceState] {
	// Create result slice with same length as input
	results := make([]*dataloader.Result[*model.SourceState], len(sourceIDs))

	// Batch fetch all states in a single repository call
	statesMap, err := repo.BatchGetStates(ctx, sourceIDs)
	if err != nil {
		// If batch operation fails, return error for all items
		for i := range results {
			results[i] = &dataloader.Result[*model.SourceState]{Error: err}
		}
		return results
	}

	// Map results back to the original order
	for i, sourceID := range sourceIDs {
		if state, ok := statesMap[sourceID]; ok {
			results[i] = &dataloader.Result[*model.SourceState]{Data: state}
		} else {
			// State not found - return nil (not an error)
			results[i] = &dataloader.Result[*model.SourceState]{Data: nil}
		}
	}

	return results
}

// ContextWithLoaders adds loaders to context
func ContextWithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

// LoadersFromContext retrieves loaders from context
func LoadersFromContext(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}
