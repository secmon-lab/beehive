// Package integration_test exercises the beehive HTTP server, catalog
// loader and memory repository together against an in-process httptest
// server. These are Go-level integration tests; the browser-side end-
// to-end suite (Playwright) lives under `frontend/e2e/` and is added
// alongside the React frontend in a later PR.
//
// These tests intentionally live OUTSIDE the package they exercise so
// they only depend on the public surface (the same surface a downstream
// integrator would touch).
package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/utils/id"
	"github.com/secmon-lab/beehive/pkg/utils/safe"

	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
)

// stubBlogProvider is registered into a per-test *fetcher.Registry so
// no init()-time global state is needed.
type stubBlogProvider struct{}

func (stubBlogProvider) Kind() types.SourceKind { return types.KindBlog }
func (stubBlogProvider) Fetch(_ context.Context, _ *model.Source) ([]*interfaces.FetchedArticle, error) {
	return nil, nil
}

const stubBlogType = "it_stub_blog"

func newIntegrationRegistry(t *testing.T) *fetcher.Registry {
	t.Helper()
	reg := fetcher.NewRegistry()
	gt.NoError(t, reg.Register(stubBlogType, stubBlogProvider{}))
	return reg
}

// TestHealth_Integration starts the HTTP server in-process and hits /health.
func TestHealth_Integration(t *testing.T) {
	bhhttp.Version = "e2e"
	srv := httptest.NewServer(bhhttp.New().Router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	gt.NoError(t, err)
	defer safe.Close(t.Context(), resp.Body)

	gt.Equal(t, resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	gt.NoError(t, err)
	var got map[string]string
	gt.NoError(t, json.Unmarshal(body, &got))
	gt.Equal(t, got["status"], "ok")
	gt.Equal(t, got["version"], "e2e")
}

// TestCatalog_Integration walks the bootstrap path:
//  1. Write a TOML config file.
//  2. Load + validate via the source_catalog service.
//  3. Verify the in-memory Catalog matches the file and Kind was filled.
//  4. Round-trip through the memory Repository (state lookup / lock).
func TestCatalog_Integration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	srcID := "src-" + id.NewULID()
	tomlBody := `
[[source]]
id = "` + srcID + `"
name = "E2E stub"
type = "` + stubBlogType + `"
url = "https://example.com/feed"
interval = "1h"
`
	path := filepath.Join(dir, "config.toml")
	gt.NoError(t, os.WriteFile(path, []byte(tomlBody), 0o644))

	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	gt.A(t, cat.Sources).Length(1)

	reg := newIntegrationRegistry(t)
	problems := source_catalog.Validate(cat, reg)
	gt.A(t, problems).Length(0)

	src, ok := cat.ByID(types.SourceID(srcID))
	gt.True(t, ok)
	gt.Equal(t, src.Kind, types.KindBlog)

	// Round-trip through memory repo: SetEnabledOverride + GetSourceState.
	repo := memory.New()
	gt.NoError(t, repo.SetEnabledOverride(ctx, src.ID, types.OverrideForceOff))
	state, err := repo.GetSourceState(ctx, src.ID)
	gt.NoError(t, err)
	gt.Equal(t, state.EnabledOverride, types.OverrideForceOff)

	// effectiveEnabled is computed correctly when state has an override.
	gt.False(t, src.EffectiveEnabled(state))
}

// TestLock_Integration exercises the lock lifecycle end-to-end on the memory
// backend: acquire, conflict, renew, release.
func TestLock_Integration(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	lock := &model.Lock{
		Kind:     model.LockKindFetch,
		TargetID: "e2e-" + id.NewULID(),
		Holder:   "instance-A",
	}

	acquired, err := repo.TryAcquireLock(ctx, lock)
	gt.NoError(t, err)
	gt.True(t, acquired)

	other := *lock
	other.Holder = "instance-B"
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
}
