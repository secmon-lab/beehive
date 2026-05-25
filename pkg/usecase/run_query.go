package usecase

import (
	"context"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// RunListLimits caps how many runs the UI list endpoint can request.
// The Runs page only browses the in-process buffer; longer-term history
// lives in BigQuery (CLAUDE.md §0).
const (
	RunListDefaultLimit = 50
	RunListMaxLimit     = 200
)

// ListRecentRuns wraps the repository call with the same clamp pattern
// used by ListRecentIoCs. The HTTP handler delegates here so the
// limit policy stays in one place.
func ListRecentRuns(ctx context.Context, repo interfaces.RunRepository, limit int) ([]*model.Run, error) {
	if limit <= 0 {
		limit = RunListDefaultLimit
	}
	if limit > RunListMaxLimit {
		limit = RunListMaxLimit
	}
	return repo.ListRecentRuns(ctx, limit)
}
