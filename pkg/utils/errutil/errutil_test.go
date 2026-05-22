package errutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

func TestHTTPStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, http.StatusOK},
		{"not_found", goerr.New("x", goerr.T(errutil.TagNotFound)), http.StatusNotFound},
		{"invalid_input", goerr.New("x", goerr.T(errutil.TagInvalidInput)), http.StatusBadRequest},
		{"conflict", goerr.New("x", goerr.T(errutil.TagConflict)), http.StatusConflict},
		{"busy", goerr.New("x", goerr.T(errutil.TagBusy)), http.StatusServiceUnavailable},
		{"unknown", goerr.New("x"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gt.Equal(t, errutil.HTTPStatus(c.err), c.want)
		})
	}
}

func TestAsCode(t *testing.T) {
	gt.Equal(t, errutil.AsCode(nil), "")
	gt.Equal(t, errutil.AsCode(goerr.New("x", goerr.T(errutil.TagNotFound))), "not_found")
	gt.Equal(t, errutil.AsCode(goerr.New("x")), "internal_error")
}

func TestNotFoundAndIsNotFound(t *testing.T) {
	err := errutil.NotFound("missing", goerr.V("id", "abc"))
	gt.True(t, errutil.IsNotFound(err))
	gt.False(t, errutil.IsNotFound(goerr.New("other")))
	gt.False(t, errutil.IsNotFound(nil))
}

func TestHandle_LogsWithGoerrValues(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	ctx := logging.With(context.Background(), logger)

	errutil.Handle(ctx, goerr.New("boom",
		goerr.V("source_id", "src-1"),
		goerr.V("count", 3),
	))

	var entry map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	gt.Value(t, entry["msg"]).Equal("boom")
	gt.Value(t, entry["source_id"]).Equal("src-1")
	// JSON numbers come back as float64.
	gt.Value(t, entry["count"]).Equal(float64(3))
}

func TestHandle_NilIsNoop(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	ctx := logging.With(context.Background(), logger)
	errutil.Handle(ctx, nil)
	gt.Equal(t, buf.Len(), 0)
}
