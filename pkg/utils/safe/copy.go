package safe

import (
	"context"
	"io"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Copy wraps io.Copy. Errors are surfaced through errutil.Handle (per
// CLAUDE.md §11). Use it where the copy result does not need to bubble
// up — e.g. serving a static file from an embedded FS.
func Copy(ctx context.Context, dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "copy failed"))
	}
}
