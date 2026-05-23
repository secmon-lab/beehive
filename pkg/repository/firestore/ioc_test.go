package firestore_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/repository/firestore"
)

// TestDedupePairs covers the dedup contract that BulkUpsertIoCs
// depends on for correctness. The behaviour cannot be observed from
// the harness without a live Firestore (Memory's UpsertIoCWithRef
// quietly absorbs duplicates via its map), so this test exists to
// catch regressions even when TEST_FIRESTORE_* is not set.
func TestDedupePairs(t *testing.T) {
	t.Run("empty input returns empty slice", func(t *testing.T) {
		out, err := firestore.DedupePairsForTest(nil)
		gt.NoError(t, err)
		gt.A(t, out).Length(0)
	})

	t.Run("all unique IoCs survive", func(t *testing.T) {
		now := time.Now().UTC()
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: "a", Type: types.IoCTypeIPv4, Value: "1.1.1.1", LastSeenAt: now}},
			{IoC: &model.IoC{ID: "b", Type: types.IoCTypeIPv4, Value: "2.2.2.2", LastSeenAt: now}},
			{IoC: &model.IoC{ID: "c", Type: types.IoCTypeIPv4, Value: "3.3.3.3", LastSeenAt: now}},
		}
		out, err := firestore.DedupePairsForTest(pairs)
		gt.NoError(t, err)
		gt.A(t, out).Length(3)
	})

	t.Run("duplicate IoC.ID collapses, latest LastSeenAt wins, first Raw preserved", func(t *testing.T) {
		now := time.Now().UTC()
		later := now.Add(time.Hour)
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: "x", Raw: "first-raw", LastSeenAt: now}},
			{IoC: &model.IoC{ID: "x", Raw: "later-raw-should-not-win", LastSeenAt: later}},
			{IoC: &model.IoC{ID: "x", Raw: "even-later-raw", LastSeenAt: later.Add(time.Minute)}},
		}
		out, err := firestore.DedupePairsForTest(pairs)
		gt.NoError(t, err)
		gt.A(t, out).Length(1)
		gt.Equal(t, out[0].Raw, "first-raw")
		gt.Equal(t, out[0].LastSeen, later.Add(time.Minute))
	})

	t.Run("identical (IoC.ID, RefID) pair collapses to one ref", func(t *testing.T) {
		now := time.Now().UTC()
		sid := types.SourceID("s1")
		aid := types.ArticleID("a1")
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: "y", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: aid}},
			{IoC: &model.IoC{ID: "y", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: aid}},
			{IoC: &model.IoC{ID: "y", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: aid}},
		}
		out, err := firestore.DedupePairsForTest(pairs)
		gt.NoError(t, err)
		gt.A(t, out).Length(1)
		gt.A(t, out[0].RefIDs).Length(1)
	})

	t.Run("distinct refs for same IoC are all kept", func(t *testing.T) {
		now := time.Now().UTC()
		sid := types.SourceID("s1")
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: "z", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: "a1"}},
			{IoC: &model.IoC{ID: "z", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: "a2"}},
			{IoC: &model.IoC{ID: "z", LastSeenAt: now}, Ref: &model.IoCRef{SourceID: sid, ArticleID: "a3"}},
		}
		out, err := firestore.DedupePairsForTest(pairs)
		gt.NoError(t, err)
		gt.A(t, out).Length(1)
		gt.A(t, out[0].RefIDs).Length(3)
	})

	t.Run("nil ref is allowed (IoC-only entries)", func(t *testing.T) {
		now := time.Now().UTC()
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: "q", LastSeenAt: now}, Ref: nil},
		}
		out, err := firestore.DedupePairsForTest(pairs)
		gt.NoError(t, err)
		gt.A(t, out).Length(1)
		gt.A(t, out[0].RefIDs).Length(0)
	})

	t.Run("empty IoC.ID is rejected", func(t *testing.T) {
		pairs := []model.IoCWithRef{
			{IoC: &model.IoC{ID: ""}},
		}
		_, err := firestore.DedupePairsForTest(pairs)
		gt.Error(t, err)
	})

	t.Run("nil IoC is rejected", func(t *testing.T) {
		pairs := []model.IoCWithRef{
			{IoC: nil},
		}
		_, err := firestore.DedupePairsForTest(pairs)
		gt.Error(t, err)
	})
}
