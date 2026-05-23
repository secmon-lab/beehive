package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

func newSourceID(t *testing.T) types.SourceID {
	t.Helper()
	return types.SourceID("src-" + id.NewULID())
}

func TestSourceState(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		sid := newSourceID(t)

		_, err := repo.GetSourceState(ctx, sid)
		gt.True(t, errutil.IsNotFound(err))

		gt.NoError(t, repo.SetEnabledOverride(ctx, sid, types.OverrideForceOff)).Required()
		got, err := repo.GetSourceState(ctx, sid)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.EnabledOverride, types.OverrideForceOff)
		gt.Equal(t, got.ID, sid)

		state := &model.SourceState{
			ID:              sid,
			EnabledOverride: types.OverrideForceOff,
			LastFetchedAt:   time.Now().UTC(),
			LastStatus:      types.RunStatusSuccess,
			LastRunID:       types.RunID("run-" + id.NewULID()),
			TotalIoCCount:   5,
		}
		gt.NoError(t, repo.UpdateSourceState(ctx, state)).Required()
		got, err = repo.GetSourceState(ctx, sid)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.LastStatus, types.RunStatusSuccess)
		gt.Equal(t, got.TotalIoCCount, 5)
		gt.False(t, got.UpdatedAt.IsZero())
	})
}

func TestSourceStateList(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		ids := []types.SourceID{newSourceID(t), newSourceID(t)}
		for _, sid := range ids {
			gt.NoError(t, repo.SetEnabledOverride(ctx, sid, types.OverrideForceOn))
		}
		states, err := repo.ListSourceStates(ctx)
		gt.NoError(t, err)
		seen := map[types.SourceID]bool{}
		for _, s := range states {
			seen[s.ID] = true
		}
		for _, sid := range ids {
			gt.True(t, seen[sid])
		}
	})
}

func TestArticle(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		aid := types.ArticleID("art-" + id.NewULID())
		url := "https://example.com/" + id.NewULID()
		a := &model.Article{
			ID:          aid,
			SourceID:    newSourceID(t),
			URL:         url,
			Title:       "hello",
			ContentHash: "abc",
			BodyText:    "body",
			FetchedAt:   time.Now().UTC(),
		}

		_, err := repo.GetArticleByURL(ctx, a.URL)
		gt.True(t, errutil.IsNotFound(err))

		gt.NoError(t, repo.CreateArticle(ctx, a)).Required()
		got, err := repo.GetArticleByURL(ctx, a.URL)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Title, "hello")

		// Conflict on duplicate Create.
		gt.Error(t, repo.CreateArticle(ctx, a))

		a2 := *a
		a2.Title = "updated"
		gt.NoError(t, repo.UpdateArticle(ctx, &a2)).Required()
		got, err = repo.GetArticleByURL(ctx, a.URL)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Title, "updated")
	})
}

func TestIoCUpsert(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now().UTC()
		value := "1.2.3." + uniqueOctet()
		ioc := &model.IoC{
			ID:          model.ComputeIoCID(types.IoCTypeIPv4, value),
			Type:        types.IoCTypeIPv4,
			Value:       value,
			Raw:         "first-raw",
			FirstSeenAt: now,
			LastSeenAt:  now,
		}
		ref := &model.IoCRef{
			SourceID:   newSourceID(t),
			SourceKind: types.KindBlog,
			ArticleID:  types.ArticleID("art-" + id.NewULID()),
			Confidence: 0.9,
			SeenAt:     now,
		}

		gt.NoError(t, repo.UpsertIoCWithRef(ctx, ioc, ref)).Required()
		got, err := repo.GetIoC(ctx, ioc.ID)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Raw, "first-raw")
		gt.Equal(t, got.Value, value)

		// Second upsert with new LastSeenAt — Raw must stay immutable.
		later := now.Add(time.Hour)
		ioc2 := *ioc
		ioc2.Raw = "should-not-overwrite"
		ioc2.LastSeenAt = later
		gt.NoError(t, repo.UpsertIoCWithRef(ctx, &ioc2, ref)).Required()

		got, err = repo.GetIoC(ctx, ioc.ID)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Raw, "first-raw")
		// Allow small clock skew on Firestore round-trip.
		gt.True(t, got.LastSeenAt.After(ioc.LastSeenAt) || got.LastSeenAt.Equal(later))
	})
}

func TestIoCBulkUpsert(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now().UTC()
		sid := newSourceID(t)
		rid := types.RunID("run-" + id.NewULID())

		// Build a 5-pair batch of fresh IPv4 IoCs.
		pairs := make([]model.IoCWithRef, 0, 5)
		ids := make([]types.IoCID, 0, 5)
		for range 5 {
			value := "10.0." + uniqueOctet() + "." + uniqueOctet()
			ioc := &model.IoC{
				ID:          model.ComputeIoCID(types.IoCTypeIPv4, value),
				Type:        types.IoCTypeIPv4,
				Value:       value,
				Raw:         "raw-" + value,
				FirstSeenAt: now,
				LastSeenAt:  now,
			}
			ref := &model.IoCRef{
				SourceID:   sid,
				SourceKind: types.KindFeed,
				RunID:      rid,
				Raw:        "raw-" + value,
				Confidence: 0.85,
				SeenAt:     now,
			}
			pairs = append(pairs, model.IoCWithRef{IoC: ioc, Ref: ref})
			ids = append(ids, ioc.ID)
		}

		// First pass: all IoCs are new.
		n, err := repo.BulkUpsertIoCs(ctx, pairs)
		gt.NoError(t, err).Required()
		gt.Equal(t, n, 5)
		for i, iid := range ids {
			got, err := repo.GetIoC(ctx, iid)
			gt.NoError(t, err).Required()
			gt.Equal(t, got.Raw, pairs[i].IoC.Raw)
			gt.Equal(t, got.Value, pairs[i].IoC.Value)
		}

		// Second pass: same pairs but later LastSeenAt and a different
		// Raw. Raw must stay immutable; LastSeenAt must be bumped.
		// This is the path that exposed the BulkWriter "duplicate
		// write for path" bug — Create→AlreadyExists→Update on the
		// same path within one BulkWriter session — so it MUST pass
		// once the GetAll-classify approach is in place.
		later := now.Add(2 * time.Hour)
		repeat := make([]model.IoCWithRef, 0, len(pairs))
		for i := range pairs {
			ioc := *pairs[i].IoC
			ioc.Raw = "should-not-overwrite"
			ioc.LastSeenAt = later
			ref := *pairs[i].Ref
			ref.SeenAt = later
			repeat = append(repeat, model.IoCWithRef{IoC: &ioc, Ref: &ref})
		}
		n, err = repo.BulkUpsertIoCs(ctx, repeat)
		gt.NoError(t, err).Required()
		gt.Equal(t, n, 5)
		for i, iid := range ids {
			got, err := repo.GetIoC(ctx, iid)
			gt.NoError(t, err).Required()
			gt.Equal(t, got.Raw, pairs[i].IoC.Raw) // unchanged
			gt.True(t, !got.LastSeenAt.Before(later.Add(-time.Second)))
		}

		// Empty batch is a valid no-op.
		n, err = repo.BulkUpsertIoCs(ctx, nil)
		gt.NoError(t, err)
		gt.Equal(t, n, 0)
		n, err = repo.BulkUpsertIoCs(ctx, []model.IoCWithRef{})
		gt.NoError(t, err)
		gt.Equal(t, n, 0)
	})
}

// TestIoCBulkUpsertInBatchDuplicates exercises the in-batch dedup
// path. urlhaus and similar feeds emit the same URL multiple times in
// one csv dump; the bulk path must collapse them to one IoC doc and
// not crash with "BulkWriter received duplicate write for path".
func TestIoCBulkUpsertInBatchDuplicates(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now().UTC()
		sid := newSourceID(t)
		rid := types.RunID("run-" + id.NewULID())

		value := "10.3." + uniqueOctet() + "." + uniqueOctet()
		iocID := model.ComputeIoCID(types.IoCTypeIPv4, value)
		mkPair := func(seenAt time.Time) model.IoCWithRef {
			return model.IoCWithRef{
				IoC: &model.IoC{
					ID:          iocID,
					Type:        types.IoCTypeIPv4,
					Value:       value,
					Raw:         "first-raw",
					FirstSeenAt: seenAt,
					LastSeenAt:  seenAt,
				},
				Ref: &model.IoCRef{
					SourceID:   sid,
					SourceKind: types.KindFeed,
					RunID:      rid,
					Raw:        "first-raw",
					Confidence: 0.85,
					SeenAt:     seenAt,
				},
			}
		}

		// 5 copies of the same (Type, Value) — duplicated path-wise.
		// Varying LastSeenAt to exercise "keep latest" semantics.
		batch := []model.IoCWithRef{
			mkPair(now),
			mkPair(now.Add(time.Minute)),
			mkPair(now.Add(2 * time.Minute)),
			mkPair(now.Add(3 * time.Minute)),
			mkPair(now.Add(4 * time.Minute)),
		}

		n, err := repo.BulkUpsertIoCs(ctx, batch)
		gt.NoError(t, err).Required()
		gt.Equal(t, n, 1) // distinct count, not raw input count

		got, err := repo.GetIoC(ctx, iocID)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Raw, "first-raw")
		gt.True(t, !got.LastSeenAt.Before(now.Add(4*time.Minute).Add(-time.Second)))
	})
}

func TestIoCBulkUpsertMixed(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now().UTC()
		sid := newSourceID(t)
		rid := types.RunID("run-" + id.NewULID())

		// Seed one existing IoC via the per-pair path so the bulk batch
		// sees a "mixed" mix of new + existing.
		existingValue := "10.1." + uniqueOctet() + "." + uniqueOctet()
		existing := &model.IoC{
			ID:          model.ComputeIoCID(types.IoCTypeIPv4, existingValue),
			Type:        types.IoCTypeIPv4,
			Value:       existingValue,
			Raw:         "original-raw",
			FirstSeenAt: now,
			LastSeenAt:  now,
		}
		existingRef := &model.IoCRef{
			SourceID:   sid,
			SourceKind: types.KindFeed,
			RunID:      rid,
			Raw:        "original-raw",
			Confidence: 0.7,
			SeenAt:     now,
		}
		gt.NoError(t, repo.UpsertIoCWithRef(ctx, existing, existingRef)).Required()

		// New batch: 1 repeated + 3 new, with later LastSeenAt.
		later := now.Add(time.Hour)
		pairs := []model.IoCWithRef{}
		// 1) repeat with changed Raw
		repeat := *existing
		repeat.Raw = "bulk-raw-should-be-ignored"
		repeat.LastSeenAt = later
		repeatRef := *existingRef
		repeatRef.SeenAt = later
		pairs = append(pairs, model.IoCWithRef{IoC: &repeat, Ref: &repeatRef})
		// 2-4) three fresh values
		freshIDs := make([]types.IoCID, 0, 3)
		for range 3 {
			value := "10.2." + uniqueOctet() + "." + uniqueOctet()
			ioc := &model.IoC{
				ID:          model.ComputeIoCID(types.IoCTypeIPv4, value),
				Type:        types.IoCTypeIPv4,
				Value:       value,
				Raw:         "fresh-raw-" + value,
				FirstSeenAt: later,
				LastSeenAt:  later,
			}
			ref := &model.IoCRef{
				SourceID:   sid,
				SourceKind: types.KindFeed,
				RunID:      rid,
				Raw:        "fresh-raw-" + value,
				Confidence: 0.9,
				SeenAt:     later,
			}
			pairs = append(pairs, model.IoCWithRef{IoC: ioc, Ref: ref})
			freshIDs = append(freshIDs, ioc.ID)
		}

		n, err := repo.BulkUpsertIoCs(ctx, pairs)
		gt.NoError(t, err).Required()
		gt.Equal(t, n, 4) // 1 existing + 3 fresh, all distinct

		// The pre-existing one keeps its Raw, has its LastSeenAt bumped.
		got, err := repo.GetIoC(ctx, existing.ID)
		gt.NoError(t, err).Required()
		gt.Equal(t, got.Raw, "original-raw")
		gt.True(t, !got.LastSeenAt.Before(later.Add(-time.Second)))

		// Fresh ones are stored with their Raw intact.
		for _, fid := range freshIDs {
			g, err := repo.GetIoC(ctx, fid)
			gt.NoError(t, err).Required()
			gt.True(t, len(g.Raw) > 0)
		}

		// Re-running the same bulk again is idempotent (ref dedup, no
		// new docs, no error).
		n, err = repo.BulkUpsertIoCs(ctx, pairs)
		gt.NoError(t, err).Required()
		gt.Equal(t, n, 4)
	})
}

func TestRun(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		run := &model.Run{
			ID:        types.RunID("run-" + id.NewULID()),
			Trigger:   "api",
			Status:    types.RunStatusRunning,
			StartedAt: time.Now().UTC(),
		}
		gt.NoError(t, repo.CreateRun(ctx, run)).Required()

		sid := newSourceID(t)
		gt.NoError(t, repo.AppendRunSource(ctx, run.ID, &model.RunSource{
			SourceID:   sid,
			SourceKind: types.KindFeed,
			Status:     types.RunStatusSuccess,
			IoCCount:   3,
		})).Required()

		got, err := repo.GetRun(ctx, run.ID)
		gt.NoError(t, err).Required()
		gt.A(t, got.Sources).Length(1)
		gt.A(t, got.SourceIDs).Length(1)
		gt.Equal(t, got.SourceIDs[0], sid)
	})
}

func TestLock(t *testing.T) {
	runOnBoth(t, func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now().UTC()
		target := "src-" + id.NewULID()
		lock := &model.Lock{
			Kind:       model.LockKindFetch,
			TargetID:   target,
			Holder:     "holder-A-" + id.NewULID(),
			AcquiredAt: now,
			ExpiresAt:  now.Add(20 * time.Second),
		}

		acquired, err := repo.TryAcquireLock(ctx, lock)
		gt.NoError(t, err)
		gt.True(t, acquired)

		other := *lock
		other.Holder = "holder-B-" + id.NewULID()
		acquired, err = repo.TryAcquireLock(ctx, &other)
		gt.NoError(t, err)
		gt.False(t, acquired)

		gt.NoError(t, repo.RenewLock(ctx, lock))

		err = repo.RenewLock(ctx, &other)
		gt.True(t, err == interfaces.ErrLockLost)

		gt.NoError(t, repo.ReleaseLock(ctx, lock.Kind, lock.TargetID))
		acquired, err = repo.TryAcquireLock(ctx, &other)
		gt.NoError(t, err)
		gt.True(t, acquired)
		gt.NoError(t, repo.ReleaseLock(ctx, other.Kind, other.TargetID))
	})
}

// uniqueOctet builds a per-test IP octet derived from the ULID so the
// IoC document IDs are unique across parallel runs.
func uniqueOctet() string {
	s := id.NewULID()
	// Use the last 2 chars to derive a value 0..99.
	n := int(s[len(s)-1]) % 100
	out := []byte("000")
	for i := len(out) - 1; i >= 0 && n > 0; i-- {
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out)
}
