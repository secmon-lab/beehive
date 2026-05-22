package errutil

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// Handle is the project-wide non-fatal error sink. Anywhere we have an
// error we don't propagate back to the caller (e.g. background tails,
// best-effort cleanups, CLI exit paths), funnel it through Handle so it
// gets logged with the full goerr metadata.
//
// Modelled after warren's errutil.Handle: every goerr.V key/value
// from the entire error chain becomes a structured log attribute, plus
// the error itself is attached under `error` so clog can pretty-print
// the chain in text mode.
//
// Do NOT call logger.Error directly for non-fatal errors — always use
// Handle so the policy stays consistent (see CLAUDE.global.md).
func Handle(ctx context.Context, err error) {
	if err == nil {
		return
	}
	// Last-ditch crash guard: never let an error handler take the
	// process down with it.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr,
				"[CRITICAL] panic inside errutil.Handle: original=%s recover=%v\n",
				err.Error(), r)
		}
	}()

	attrs := []any{slog.Any("error", err)}

	// Flatten the full goerr value chain. `goerr.Values` already walks
	// wrapped errors and merges them — the outer call wins on key
	// collisions, which is the right behaviour for our usage.
	for k, v := range goerr.Values(err) {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Surface tags so authn / authz / typed handlers downstream can
	// switch on them without re-walking the chain.
	if tags := goerr.Tags(err); len(tags) > 0 {
		attrs = append(attrs, slog.Any("tags", tags))
	}

	logging.From(ctx).Error(err.Error(), attrs...)
}
