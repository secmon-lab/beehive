package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// SourceView is the API-facing combination of a TOML-defined Source and
// its dynamic state from Firestore. The HTTP handler builds responses
// from this struct.
type SourceView struct {
	Source *model.Source
	State  *model.SourceState
}

// ListSources merges the in-memory catalog with per-source state read
// from Firestore. Missing state docs become nil pointers — the caller
// can interpret that as "never fetched".
func ListSources(ctx context.Context, cat *source_catalog.Catalog, repo interfaces.StateRepository) ([]*SourceView, error) {
	if cat == nil {
		return nil, goerr.New("catalog is nil")
	}
	views := make([]*SourceView, 0, len(cat.Sources))
	for _, src := range cat.Sources {
		state, err := repo.GetSourceState(ctx, src.ID)
		if err != nil && !errutil.IsNotFound(err) {
			return nil, err
		}
		views = append(views, &SourceView{Source: src, State: state})
	}
	return views, nil
}

// GetSource returns one merged view. Returns a tagged not-found when
// the SourceID is not in the catalog.
func GetSource(ctx context.Context, cat *source_catalog.Catalog, repo interfaces.StateRepository, id types.SourceID) (*SourceView, error) {
	src, ok := cat.ByID(id)
	if !ok {
		return nil, errutil.NotFound("source", goerr.V("id", id))
	}
	state, err := repo.GetSourceState(ctx, src.ID)
	if err != nil && !errutil.IsNotFound(err) {
		return nil, err
	}
	return &SourceView{Source: src, State: state}, nil
}

// SetEnabledOverride is a thin pass-through to the repository — kept
// in usecase so handlers depend on usecase rather than the repo
// interface directly.
func SetEnabledOverride(ctx context.Context, repo interfaces.StateRepository, id types.SourceID, override types.EnabledOverride) error {
	if !override.Valid() {
		return goerr.New("invalid override value",
			goerr.V("value", override),
			goerr.T(errutil.TagInvalidInput))
	}
	return repo.SetEnabledOverride(ctx, id, override)
}
