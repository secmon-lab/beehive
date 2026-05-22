// Package logging provides a context-scoped slog logger so that every
// goroutine / request can carry its own attributes (request_id, source_id,
// etc.) without resorting to package-global state.
//
// Always obtain a logger with From(ctx). Never call slog.Info / slog.Error
// directly (see CLAUDE.global.md rules/go.md).
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/m-mizutani/clog"
)

type ctxKey struct{}

var defaultLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

// SetDefault replaces the package-global fallback used by From(ctx) when
// the context has no logger attached. Call this once at process startup
// (e.g. from cli.go).
func SetDefault(l *slog.Logger) { defaultLogger = l }

// Default returns the current fallback logger.
func Default() *slog.Logger { return defaultLogger }

// With returns a copy of ctx that carries logger as the scoped logger.
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// From returns the logger attached to ctx, or the package default.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return defaultLogger
}

// Config describes how to construct a process-wide default logger from
// env-driven settings.
type Config struct {
	Level  slog.Level
	Format string // "json" | "text"
}

// Build returns a logger writing to w according to cfg. "text" uses clog
// for human-friendly output; "json" uses the stdlib JSON handler so logs
// land in Cloud Logging in a structured form.
func Build(w io.Writer, cfg Config) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	switch cfg.Format {
	case "text":
		return slog.New(clog.New(
			clog.WithWriter(w),
			clog.WithLevel(cfg.Level),
			clog.WithSource(false),
		))
	default:
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: cfg.Level}))
	}
}
