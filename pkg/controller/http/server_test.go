package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/m-mizutani/gt"
	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/utils/safe"
)

func TestHealthEndpoint(t *testing.T) {
	bhhttp.Version = "test-1.2.3"
	srv := httptest.NewServer(bhhttp.New().Router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	gt.NoError(t, err)
	defer safe.Close(t.Context(), resp.Body)

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
	defer safe.Close(t.Context(), resp.Body)

	gt.Equal(t, resp.StatusCode, http.StatusNotFound)
	var body map[string]any
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body["code"], "not_found")
}

func TestStaticFS_ServesIndexAndAssetsWithSPAFallback(t *testing.T) {
	static := fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><title>beehive</title>")},
		"assets/app.js":  {Data: []byte("console.log('hi');")},
		"assets/app.css": {Data: []byte("body{margin:0}")},
	}
	srv := httptest.NewServer(bhhttp.New(bhhttp.WithStaticFS(static)).Router)
	defer srv.Close()

	cases := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"root serves index.html", "/", "<!doctype html><title>beehive</title>"},
		{"asset path serves the asset", "/assets/app.js", "console.log('hi');"},
		{"SPA client route falls back to index.html", "/sources", "<!doctype html><title>beehive</title>"},
		{"deep SPA client route falls back to index.html", "/iocs/lookup/result", "<!doctype html><title>beehive</title>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			gt.NoError(t, err)
			defer safe.Close(t.Context(), resp.Body)

			gt.Equal(t, resp.StatusCode, http.StatusOK)
			body, err := io.ReadAll(resp.Body)
			gt.NoError(t, err)
			gt.Equal(t, string(body), tc.wantBody)
		})
	}
}

func TestStaticFS_DoesNotShadowAPI404(t *testing.T) {
	static := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html>")},
	}
	srv := httptest.NewServer(bhhttp.New(bhhttp.WithStaticFS(static)).Router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/nope")
	gt.NoError(t, err)
	defer safe.Close(t.Context(), resp.Body)

	gt.Equal(t, resp.StatusCode, http.StatusNotFound)
	gt.Equal(t, resp.Header.Get("Content-Type"), "application/json")
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
	defer safe.Close(t.Context(), resp.Body)

	gt.Equal(t, resp.StatusCode, http.StatusInternalServerError)
	var body map[string]string
	gt.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gt.Equal(t, body["code"], "internal_error")
}
