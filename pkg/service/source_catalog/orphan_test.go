package source_catalog_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

func TestDetectOrphanStates(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	known := types.SourceID("known-" + id.NewULID())
	orphan := types.SourceID("orphan-" + id.NewULID())
	// Seed two state docs.
	gt.NoError(t, repo.SetEnabledOverride(ctx, known, types.OverrideNone))
	gt.NoError(t, repo.SetEnabledOverride(ctx, orphan, types.OverrideNone))

	cat := &source_catalog.Catalog{
		Sources: []*model.Source{{ID: known}},
	}
	got, err := source_catalog.DetectOrphanStates(ctx, repo, cat)
	gt.NoError(t, err)
	gt.A(t, got).Length(1)
	gt.Equal(t, got[0], orphan)
}

func TestDetectOrphanStates_EmptyState(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()
	got, err := source_catalog.DetectOrphanStates(ctx, repo, &source_catalog.Catalog{})
	gt.NoError(t, err)
	gt.A(t, got).Length(0)
}
