package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
)

func TestHealthEndpoint(t *testing.T) {
	bhhttp.Version = "test-1.2.3"
	srv := httptest.NewServer(bhhttp.New().Router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	gt.NoError(t, err)
	defer resp.Body.Close()

	gt.Equal(t, resp.StatusCode, http.StatusOK)
	var body map[string]string
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body["status"], "ok")
	gt.Equal(t, body["version"], "test-1.2.3")
}

func TestUnknownRouteReturnsProblemJSON(t *testing.T) {
	srv := httptest.NewServer(bhhttp.New().Router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/nope")
	gt.NoError(t, err)
	defer resp.Body.Close()

	gt.Equal(t, resp.StatusCode, http.StatusNotFound)
	var body map[string]any
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body["code"], "not_found")
}

func TestRecoverer_TurnsPanicInto500(t *testing.T) {
	srv := bhhttp.New()
	srv.Router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/panic")
	gt.NoError(t, err)
	defer resp.Body.Close()

	gt.Equal(t, resp.StatusCode, http.StatusInternalServerError)
	var body map[string]string
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body["code"], "internal_error")
}
