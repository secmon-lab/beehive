package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

func TestFrom_ReturnsDefaultWhenContextEmpty(t *testing.T) {
	def := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	logging.SetDefault(def)
	t.Cleanup(func() {
		logging.SetDefault(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	})

	got := logging.From(context.Background())
	gt.Equal(t, got, def)
}

func TestWithAndFrom_RoundTrip(t *testing.T) {
	custom := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	ctx := logging.With(context.Background(), custom)
	gt.Equal(t, logging.From(ctx), custom)
}

func TestBuild_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.Build(buf, logging.Config{Level: slog.LevelInfo, Format: "json"})
	logger.Info("hello", slog.String("k", "v"))

	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	gt.Value(t, got["msg"]).Equal("hello")
	gt.Value(t, got["k"]).Equal("v")
}

func TestBuild_TextFormatUsesClog(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.Build(buf, logging.Config{Level: slog.LevelInfo, Format: "text"})
	logger.Info("hello-clog")
	// We don't assert the clog-specific layout, only that something
	// recognisable was emitted.
	gt.True(t, bytes.Contains(buf.Bytes(), []byte("hello-clog")))
}

func TestBuild_RespectsLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.Build(buf, logging.Config{Level: slog.LevelError, Format: "json"})
	logger.Info("filtered-out")
	gt.Equal(t, buf.Len(), 0)
	logger.Error("kept")
	gt.True(t, bytes.Contains(buf.Bytes(), []byte("kept")))
}
