package usecase

import (
	"context"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/normalize"
)

// LookupIoC normalises (type, value) and returns the matching IoC if
// any. Returns a tagged not-found error when nothing matches, so the
// HTTP layer can map it to 404.
func LookupIoC(ctx context.Context, repo interfaces.IoCRepository, t types.IoCType, value string) (*model.IoC, error) {
	v, err := normalize.IoC(t, value)
	if err != nil {
		return nil, err
	}
	id := model.ComputeIoCID(t, v)
	return repo.GetIoC(ctx, id)
}

// IoCListLimits caps how many IoCs the UI list endpoint can request.
const (
	IoCListDefaultLimit = 100
	IoCListMaxLimit     = 500
)

// ListRecentIoCs is a thin wrapper around the repository call so the
// HTTP handler depends on the usecase layer rather than reaching into
// the repository directly. The limit is clamped to [1, IoCListMaxLimit]
// — bulk analytics still belong in BigQuery (CLAUDE.md §0).
func ListRecentIoCs(ctx context.Context, repo interfaces.IoCRepository, limit int) ([]*model.IoC, error) {
	if limit <= 0 {
		limit = IoCListDefaultLimit
	}
	if limit > IoCListMaxLimit {
		limit = IoCListMaxLimit
	}
	return repo.ListRecentIoCs(ctx, limit)
}
