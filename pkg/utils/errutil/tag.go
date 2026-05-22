// Package errutil maps domain errors to HTTP statuses and centralises the
// non-fatal error reporting flow (logger + future Sentry).
package errutil

import (
	"errors"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
)

// Tag values are attached to errors with goerr.T(...) and inspected at the
// HTTP boundary to pick a status code. Use errors.Is via goerr.HasTag to
// match.
var (
	TagNotFound     = goerr.NewTag("not_found")
	TagInvalidInput = goerr.NewTag("invalid_input")
	TagConflict     = goerr.NewTag("conflict")
	TagBusy         = goerr.NewTag("busy") // e.g. lock contention
)

// HTTPStatus returns the HTTP status code that best matches err. Unknown
// errors map to 500. Authn/Authz errors are intentionally NOT mapped here
// — that responsibility lives in the future Rego-based authz layer.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case goerr.HasTag(err, TagNotFound):
		return http.StatusNotFound
	case goerr.HasTag(err, TagInvalidInput):
		return http.StatusBadRequest
	case goerr.HasTag(err, TagConflict):
		return http.StatusConflict
	case goerr.HasTag(err, TagBusy):
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

// AsCode returns a stable short identifier suitable for a problem+json
// `code` field.
func AsCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case goerr.HasTag(err, TagNotFound):
		return "not_found"
	case goerr.HasTag(err, TagInvalidInput):
		return "invalid_input"
	case goerr.HasTag(err, TagConflict):
		return "conflict"
	case goerr.HasTag(err, TagBusy):
		return "busy"
	}
	return "internal_error"
}

// Wrap is a thin convenience around goerr.Wrap that preserves nil pass-
// through. Use it at call sites that always want to wrap, since goerr.Wrap
// itself does the same — this helper just lets us evolve the policy in
// one place later (e.g. attach trace IDs).
func Wrap(err error, msg string, opts ...goerr.Option) error {
	if err == nil {
		return nil
	}
	return goerr.Wrap(err, msg, opts...)
}

// IsNotFound is a tiny helper used by repository callers that need to
// distinguish "absent" from "real failure".
func IsNotFound(err error) bool {
	return err != nil && (goerr.HasTag(err, TagNotFound) || errors.Is(err, errNotFoundSentinel))
}

// errNotFoundSentinel lets non-goerr callers (rare) signal not-found via
// errors.Is.
var errNotFoundSentinel = goerr.New("not found", goerr.T(TagNotFound))

// NotFound returns a new tagged not-found error.
func NotFound(msg string, opts ...goerr.Option) error {
	opts = append(opts, goerr.T(TagNotFound))
	return goerr.New(msg, opts...)
}
