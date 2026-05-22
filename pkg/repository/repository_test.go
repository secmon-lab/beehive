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
