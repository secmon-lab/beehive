package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := memory.New()
	cat := &source_catalog.Catalog{
		Sources: []*model.Source{
			{ID: types.SourceID("src-1"), Name: "S1", Kind: types.KindFeed, Type: "cinsscore_badguys", Interval: 30 * time.Minute},
		},
	}
	deps := usecase.Deps{
		Repo:    repo,
		Lock:    usecase.NewLockManager(repo, "test"),
		Catalog: cat,
	}
	srv := bhhttp.New(bhhttp.WithDeps(deps), bhhttp.WithCatalog(cat))
	return httptest.NewServer(srv.Router)
}

func TestListSources(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/sources")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusOK)

	var body map[string]any
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	arr, _ := body["sources"].([]any)
	gt.Equal(t, len(arr), 1)
}

func TestGetSource_NotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/sources/missing")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusNotFound)
}

func TestSetEnabledOverride(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut,
		ts.URL+"/api/v1/sources/src-1/enabled_override",
		strings.NewReader(`{"value":"force_off"}`),
	)
	gt.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusNoContent)
}

func TestLookupIoC_BadType(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/iocs/lookup?type=bogus&value=x")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func TestLookupIoC_NotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/iocs/lookup?type=ipv4&value=1.2.3.4")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusNotFound)
}

func TestTriggerFetchAll_NoDueSources(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/fetch", "application/json", nil)
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestTriggerFetchAll_AsyncReturnsRunID(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/fetch?mode=async", "application/json", nil)
	gt.NoError(t, err)
	defer resp.Body.Close()
	// Async path returns 202 Accepted immediately.
	gt.Equal(t, resp.StatusCode, http.StatusAccepted)

	var body struct {
		RunID string `json:"runId"`
		Mode  string `json:"mode"`
	}
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.True(t, body.RunID != "")
	gt.Equal(t, body.Mode, "async")
}

func TestTriggerFetchAll_DedupesConcurrent(t *testing.T) {
	// A pre-existing in-progress Run should make the next /fetch call
	// echo the same runId instead of starting a new one.
	repo := memory.New()
	cat := &source_catalog.Catalog{}
	deps := usecase.Deps{Repo: repo, Lock: usecase.NewLockManager(repo, "test"), Catalog: cat}
	srv := bhhttp.New(bhhttp.WithDeps(deps), bhhttp.WithCatalog(cat))
	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	existing := &model.Run{
		ID:        types.RunID("run-existing-" + id.NewULID()),
		Trigger:   "api",
		Status:    types.RunStatusRunning,
		StartedAt: time.Now().UTC(),
	}
	gt.NoError(t, repo.CreateRun(t.Context(), existing))

	resp, err := http.Post(ts.URL+"/api/v1/fetch?mode=async", "application/json", nil)
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusAccepted)

	var body struct {
		RunID      string `json:"runId"`
		InProgress bool   `json:"inProgress"`
	}
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body.RunID, string(existing.ID))
	gt.True(t, body.InProgress)
}

func TestListRuns_Empty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/runs")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusOK)

	var body map[string]any
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	arr, _ := body["runs"].([]any)
	gt.Equal(t, len(arr), 0)
}

func TestListRuns_OrdersByStartedAtDesc(t *testing.T) {
	repo := memory.New()
	cat := &source_catalog.Catalog{}
	deps := usecase.Deps{Repo: repo, Lock: usecase.NewLockManager(repo, "test"), Catalog: cat}
	srv := bhhttp.New(bhhttp.WithDeps(deps), bhhttp.WithCatalog(cat))
	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	// Three runs with monotonically increasing StartedAt so the API
	// must reorder them DESC.
	now := time.Now().UTC()
	for i, dt := range []time.Duration{-2 * time.Hour, -1 * time.Hour, -30 * time.Minute} {
		run := &model.Run{
			ID:        types.RunID("run-" + id.NewULID()),
			Trigger:   "scheduler",
			Status:    types.RunStatusSuccess,
			StartedAt: now.Add(dt),
			Total:     i + 1,
		}
		gt.NoError(t, repo.CreateRun(t.Context(), run))
	}

	resp, err := http.Get(ts.URL + "/api/v1/runs")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusOK)

	var body struct {
		Runs []struct {
			ID    string `json:"id"`
			Total int    `json:"total"`
		} `json:"runs"`
	}
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, len(body.Runs), 3)
	// Newest first: Total 3, 2, 1.
	gt.Equal(t, body.Runs[0].Total, 3)
	gt.Equal(t, body.Runs[1].Total, 2)
	gt.Equal(t, body.Runs[2].Total, 1)
}

func TestIoCStats_EmptyCounterDoc(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// Fresh repo: counter has never been persisted, the handler must
	// return the zero shape rather than a 5xx.
	resp, err := http.Get(ts.URL + "/api/v1/iocs/stats")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusOK)

	var body struct {
		Total  int64            `json:"total"`
		ByType map[string]int64 `json:"byType"`
	}
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	for _, typ := range []string{"ipv4", "ipv6", "domain", "url", "md5", "sha1", "sha256", "sha512", "email"} {
		_, ok := body.ByType[typ]
		gt.True(t, ok)
	}
	gt.Equal(t, body.Total, int64(0))
}

func TestListRuns_BadLimit(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/runs?limit=-1")
	gt.NoError(t, err)
	defer resp.Body.Close()
	gt.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

// silence unused id import in case tests are pruned later.
var _ = id.NewULID
