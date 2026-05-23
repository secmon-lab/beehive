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

func TestNew_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New(buf, slog.LevelInfo, logging.FormatJSON, false)
	logger.Info("hello", slog.String("k", "v"))

	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	gt.Value(t, got["msg"]).Equal("hello")
	gt.Value(t, got["k"]).Equal("v")
}

func TestNew_ConsoleFormatUsesClog(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New(buf, slog.LevelInfo, logging.FormatConsole, false)
	logger.Info("hello-clog")
	gt.True(t, bytes.Contains(buf.Bytes(), []byte("hello-clog")))
}

func TestNew_RespectsLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New(buf, slog.LevelError, logging.FormatJSON, false)
	logger.Info("filtered-out")
	gt.Equal(t, buf.Len(), 0)
	logger.Error("kept")
	gt.True(t, bytes.Contains(buf.Bytes(), []byte("kept")))
}

func TestNew_AutoFallsBackToJSONForNonTTY(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New(buf, slog.LevelInfo, logging.FormatAuto, false)
	logger.Info("auto-json", slog.String("k", "v"))

	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	gt.Value(t, got["msg"]).Equal("auto-json")
	gt.Value(t, got["k"]).Equal("v")
}

func TestNew_JSONRedactsSecrets(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New(buf, slog.LevelInfo, logging.FormatJSON, false)
	logger.Info("auth", slog.String("Authorization", "Bearer s3cret"))

	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	gt.Value(t, got["Authorization"]).NotEqual("Bearer s3cret")
}

func TestQuiet_Discards(t *testing.T) {
	prev := logging.Default()
	t.Cleanup(func() { logging.SetDefault(prev) })

	logging.Quiet()
	logging.Default().Error("nope")
	// no assertion on output — sink is io.Discard. Just ensure no panic.
}

func TestErrAttr_AnyError(t *testing.T) {
	attr := logging.ErrAttr(context.Canceled)
	gt.Equal(t, attr.Key, "error")
}
