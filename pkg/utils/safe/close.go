// Package safe collects defensive IO helpers. The goal is to make
// "_ = x.Close()" disappear from the codebase entirely.
package safe

import (
	"context"
	"io"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Close calls Close() on c if c is non-nil. Errors are forwarded
// through errutil.Handle (per CLAUDE.md §11) so they show up with the
// full goerr metadata; the function itself returns nothing so it stays
// convenient inside defer statements.
func Close(ctx context.Context, c io.Closer) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "close failed"))
	}
}
