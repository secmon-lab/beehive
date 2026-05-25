package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// IoCListCursor identifies a boundary between two pages of the
// recent-IoCs list. The cursor pairs (LastSeenAt, ID) so that bulk-
// persisted batches — every IoC of which shares the same LastSeenAt
// (see usecase/fetch.go::bulkPersistSeeds) — cannot be silently
// truncated at a page boundary. A LastSeenAt-only cursor would skip
// every same-timestamp document past the page break.
//
// Pages are walked under (LastSeenAt DESC, ID DESC); the matching
// direction lets the Firestore backend stay on its auto-created
// single-field index without provisioning a composite index.
type IoCListCursor struct {
	LastSeenAt time.Time
	ID         types.IoCID
}
