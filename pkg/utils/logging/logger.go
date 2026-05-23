// Package logging provides a context-scoped slog logger so that every
// goroutine / request can carry its own attributes (request_id, source_id,
// etc.) without resorting to package-global state.
//
// Always obtain a logger with From(ctx). Never call slog.Info / slog.Error
// directly (see CLAUDE.global.md rules/go.md).
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/clog/hooks"
	"github.com/m-mizutani/masq"
	"github.com/mattn/go-isatty"
)

// Format selects the output encoding.
type Format int

const (
	// FormatAuto picks FormatConsole when the destination writer is a
	// TTY and FormatJSON otherwise — the right default for a CLI that
	// also runs under Cloud Run.
	FormatAuto Format = iota
	// FormatConsole emits human-friendly clog output (optionally coloured).
	FormatConsole
	// FormatJSON emits the stdlib slog JSON handler — used in Cloud Run.
	FormatJSON
)

type ctxKey struct{}

var (
	defaultLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	loggerMutex   sync.RWMutex
)

// SetDefault replaces the package-global fallback used by From(ctx) when
// the context has no logger attached. Call this once at process startup.
func SetDefault(l *slog.Logger) {
	loggerMutex.Lock()
	defaultLogger = l
	loggerMutex.Unlock()
}

// Default returns the current fallback logger.
func Default() *slog.Logger {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	return defaultLogger
}

// Quiet swaps the default logger for an info-level JSON sink that
// discards every record.
func Quiet() {
	SetDefault(slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}

// With returns a copy of ctx that carries logger as the scoped logger.
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// From returns the logger attached to ctx, or the package default.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return Default()
}

// ErrAttr is shorthand for slog.Any("error", err) — keeps call sites
// consistent so the GoErr hook can recognise the attribute.
func ErrAttr(err error) slog.Attr { return slog.Any("error", err) }

// isTerminal reports whether w is a TTY so clog can emit ANSI colours
// only when output won't be captured by a pipe or file.
func isTerminal(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// New constructs a *slog.Logger that writes to w at the given level.
// FormatConsole renders via clog (with goerr-aware attribute expansion
// and a masq filter that hides Authorization / "secret_*" / `secret:`-
// tagged fields). FormatJSON falls back to the stdlib JSON handler.
// FormatAuto picks Console for TTY destinations and JSON otherwise.
func New(w io.Writer, level slog.Level, format Format, stacktrace bool) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}

	if format == FormatAuto {
		if isTerminal(w) {
			format = FormatConsole
		} else {
			format = FormatJSON
		}
	}

	filter := masq.New(
		masq.WithTag("secret"),
		masq.WithFieldPrefix("secret_"),
		masq.WithFieldName("Authorization"),
	)

	switch format {
	case FormatConsole:
		return slog.New(clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithReplaceAttr(filter),
			clog.WithAttrHook(hooks.GoErr(hooks.WithStackTrace(stacktrace))),
			clog.WithColor(isTerminal(w)),
			clog.WithColorMap(&clog.ColorMap{
				Level: map[slog.Level]*color.Color{
					slog.LevelDebug: color.New(color.FgGreen, color.Bold),
					slog.LevelInfo:  color.New(color.FgCyan, color.Bold),
					slog.LevelWarn:  color.New(color.FgYellow, color.Bold),
					slog.LevelError: color.New(color.FgRed, color.Bold),
				},
				LevelDefault: color.New(color.FgBlue, color.Bold),
				Time:         color.New(color.FgWhite),
				Message:      color.New(color.FgHiWhite),
				AttrKey:      color.New(color.FgHiCyan),
				AttrValue:    color.New(color.FgHiWhite),
			}),
		))

	case FormatJSON:
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       level,
			ReplaceAttr: filter,
		}))

	default:
		panic(fmt.Sprintf("logging: unsupported format %d", format))
	}
}
