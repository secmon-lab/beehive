// Package async wraps `go` with panic recovery and logger propagation.
// All goroutines in beehive go through Dispatch — there are no raw
// `go func() { ... }()` invocations in the codebase.
//
// IMPORTANT: every goroutine launched here MUST finish within the lifetime
// of the originating HTTP request. beehive runs on Cloud Run with zero
// scale, so any goroutine that outlives its request risks being killed
// when the container shuts down. See CLAUDE.md §0.
package async

import (
	"context"
	"runtime/debug"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Dispatch runs f in a new goroutine with panic recovery and the same
// context-scoped logger. If f returns an error, it is reported through
// errutil.Handle. Panics are recovered, wrapped as a goerr, and likewise
// reported.
//
// The goroutine MUST finish within the lifetime of the originating
// HTTP request. For deliberately request-outliving work (e.g. async
// fetch mode) use DispatchDetached instead.
func Dispatch(ctx context.Context, f func(ctx context.Context) error) {
	dispatch(ctx, f)
}

// DispatchDetached runs f in a new goroutine that is ALLOWED to outlive
// the originating HTTP request. Reserved for endpoints that explicitly
// hand the client a runId / job id to poll (today: POST /api/v1/fetch
// ?mode=async).
//
// Cloud Run caveat: with zero-scale and no active requests, the
// container MAY be killed soon after the originating request returns.
// Clients SHOULD poll the persisted progress (e.g. runs/{RunID})
// rather than assume completion. Callers MUST pass a detached
// context.Background()-based ctx (NOT the http request ctx, which is
// cancelled on response). Pull the logger across with
// `logging.With(context.Background(), logging.From(r.Context()))`.
func DispatchDetached(ctx context.Context, f func(ctx context.Context) error) {
	dispatch(ctx, f)
}

func dispatch(ctx context.Context, f func(ctx context.Context) error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := goerr.New("panic in async.Dispatch",
					goerr.V("recover", r),
					goerr.V("stack", string(debug.Stack())),
				)
				errutil.Handle(ctx, err)
			}
		}()
		if err := f(ctx); err != nil {
			errutil.Handle(ctx, err)
		}
	}()
}
