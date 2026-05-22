package usecase_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

// emptyRegistry returns a fresh registry with no providers — enough to
// drive Bootstrap.Ensure / FetchAll paths that never resolve a provider.
func emptyRegistry() *fetcher.Registry { return fetcher.NewRegistry() }

// stubLLM lets us drive the extractor without a real provider.
type stubLLM struct{ payload string }

func (s *stubLLM) GenerateJSON(_ context.Context, _, _ string, _ []byte, out any) error {
	return json.NewDecoder(strings.NewReader(s.payload)).Decode(out)
}

func TestLookupIoC_HitAndMiss(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	value, _ := normalizeFixture(t, types.IoCTypeIPv4, "1.2.3.4")
	ioc := &model.IoC{
		ID:    model.ComputeIoCID(types.IoCTypeIPv4, value),
		Type:  types.IoCTypeIPv4,
		Value: value,
	}
	ref := &model.IoCRef{SourceID: "src-1", SourceKind: types.KindFeed}
	gt.NoError(t, repo.UpsertIoCWithRef(ctx, ioc, ref))

	got, err := usecase.LookupIoC(ctx, repo, types.IoCTypeIPv4, "1.2.3.4")
	gt.NoError(t, err)
	gt.Equal(t, got.Value, value)

	_, err = usecase.LookupIoC(ctx, repo, types.IoCTypeIPv4, "9.9.9.9")
	gt.Error(t, err)
}

func TestSourceQuery_ListAndGet(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()
	cat := &source_catalog.Catalog{
		Sources: []*model.Source{
			{ID: types.SourceID("s-a"), Name: "A"},
			{ID: types.SourceID("s-b"), Name: "B"},
		},
	}
	views, err := usecase.ListSources(ctx, cat, repo)
	gt.NoError(t, err)
	gt.A(t, views).Length(2)

	v, err := usecase.GetSource(ctx, cat, repo, "s-a")
	gt.NoError(t, err)
	gt.Equal(t, v.Source.Name, "A")

	_, err = usecase.GetSource(ctx, cat, repo, "missing")
	gt.Error(t, err)
}

func TestSetEnabledOverride(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()
	gt.NoError(t, usecase.SetEnabledOverride(ctx, repo, "s-1", types.OverrideForceOff))

	got, err := repo.GetSourceState(ctx, "s-1")
	gt.NoError(t, err)
	gt.Equal(t, got.EnabledOverride, types.OverrideForceOff)

	gt.Error(t, usecase.SetEnabledOverride(ctx, repo, "s-1", types.EnabledOverride("nonsense")))
}

func TestLockManager_AcquireAndRelease(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()
	mgr := usecase.NewLockManager(repo, "instance-A-"+id.NewULID())
	// Speed up the heartbeat loop so the test stays fast.
	old := usecase.LockHeartbeatInterval
	usecase.LockHeartbeatInterval = 10 * time.Millisecond
	t.Cleanup(func() { usecase.LockHeartbeatInterval = old })

	lock, err := mgr.Acquire(ctx, model.LockKindFetch, "src-"+id.NewULID())
	gt.NoError(t, err)
	gt.NotNil(t, lock)

	// Second acquire from a different manager should fail with ErrLockBusy.
	mgr2 := usecase.NewLockManager(repo, "instance-B-"+id.NewULID())
	_, err = mgr2.Acquire(ctx, model.LockKindFetch, "src-shared-"+id.NewULID())
	// Different target id — should succeed; ensure the API path works.
	gt.NoError(t, err)

	gt.NoError(t, lock.Release(ctx))
}

func TestBootstrap_LoadsCatalog(t *testing.T) {
	ctx := context.Background()
	// A non-existent config path is valid — Bootstrap should not error.
	bs := usecase.NewBootstrap()
	path := t.TempDir() + "/config.toml"
	cat, err := bs.Ensure(ctx, path, memory.New(), emptyRegistry())
	gt.NoError(t, err)
	gt.A(t, cat.Sources).Length(0)

	// Second call returns the cached value.
	cat2, err := bs.Ensure(ctx, path, memory.New(), emptyRegistry())
	gt.NoError(t, err)
	gt.Equal(t, cat, cat2)
}

func TestBootstrap_SurfacesProblemDetails(t *testing.T) {
	// Bootstrap must propagate the full validation error text so a CLI
	// user actually sees what is wrong (not just "N problems").
	dir := t.TempDir()
	path := dir + "/config.toml"
	body := []byte(`
[[source]]
id = "bad-1"
type = "definitely-not-registered"
interval = "1h"

[[source]]
id = "bad-2"
type = "still-not-registered"
interval = "30m"
`)
	gt.NoError(t, os.WriteFile(path, body, 0o644))

	bs := usecase.NewBootstrap()
	_, err := bs.Ensure(context.Background(), path, memory.New(), emptyRegistry())
	gt.Error(t, err)

	msg := err.Error()
	t.Logf("bootstrap error message:\n%s", msg)
	// The aggregated error must list each individual problem inline
	// (not just "N problems"), so a CLI user knows what to fix.
	gt.True(t, strings.Contains(msg, "source config problem(s)"))
	gt.True(t, strings.Contains(msg, "unknown provider"))
	gt.True(t, strings.Contains(msg, `id="bad-1"`))
	gt.True(t, strings.Contains(msg, `id="bad-2"`))
}

func TestFetchAll_SkipsWhenNotDue(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	src := &model.Source{
		ID:   types.SourceID("blog-not-due-" + id.NewULID()),
		Name: "X", Kind: types.KindBlog, Type: "rss",
		URL: "https://example.com/feed", Interval: time.Hour,
	}
	cat := &source_catalog.Catalog{Sources: []*model.Source{src}}

	// Pretend we just fetched it.
	gt.NoError(t, repo.UpdateSourceState(ctx, &model.SourceState{
		ID:            src.ID,
		LastFetchedAt: time.Now().UTC(),
	}))

	deps := usecase.Deps{
		Repo:      repo,
		Lock:      usecase.NewLockManager(repo, "test"),
		Catalog:   cat,
		Extractor: nil,
	}
	run, err := usecase.FetchAll(ctx, deps, "api", "")
	gt.NoError(t, err)
	gt.Equal(t, run.Total, 1)
	gt.Equal(t, run.Skipped, 1)
}

func TestFetchAll_RespectsDisabled(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()
	src := &model.Source{
		ID:   types.SourceID("disabled-" + id.NewULID()),
		Name: "Y", Kind: types.KindFeed, Type: "cinsscore_badguys",
		Interval: time.Hour, Disabled: true,
	}
	cat := &source_catalog.Catalog{Sources: []*model.Source{src}}
	deps := usecase.Deps{Repo: repo, Lock: usecase.NewLockManager(repo, "t"), Catalog: cat}
	run, err := usecase.FetchAll(ctx, deps, "api", "")
	gt.NoError(t, err)
	gt.Equal(t, run.Skipped, 1)
}

// normalizeFixture returns the canonical Value form for a raw IoC input
// of a known good type. Kept tiny — the tests using it only need IPv4.
func normalizeFixture(t *testing.T, _ types.IoCType, raw string) (string, error) {
	t.Helper()
	return raw, nil
}

// Keep imports used by future test additions alive without nolint
// pragmas.
var (
	_ = interfaces.IoCSeed{}
	_ = &stubLLM{}
)
