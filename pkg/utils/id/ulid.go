// Package id centralises identifier generation. ULIDs are used for
// time-ordered, sortable IDs (Article, Run); deterministic content-derived
// IDs are computed in pkg/domain/model.
package id

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	mu      sync.Mutex
	entropy = ulid.Monotonic(rand.Reader, 0)
)

// NewULID returns a fresh, time-sortable ULID encoded as 26 base32 chars.
// The monotonic entropy source ensures uniqueness even when many IDs are
// generated within the same millisecond.
func NewULID() string {
	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), entropy).String()
}
