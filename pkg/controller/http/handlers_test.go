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

// silence unused id import in case tests are pruned later.
var _ = id.NewULID
