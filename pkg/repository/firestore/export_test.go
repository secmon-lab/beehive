package firestore

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// DedupedSummary is the test-only view of one dedupePairs entry. The
// internal iocEntry type is intentionally private so callers cannot
// build it manually; this struct flattens what the test needs to
// assert about.
type DedupedSummary struct {
	IoCID    types.IoCID
	Raw      string
	LastSeen time.Time
	RefIDs   []types.RefID
}

// SetBulkChunkSizeForTest overrides bulkChunkSize for the duration of
// a test and returns a restore function. Tests use this to exercise
// the chunking + cross-chunk progress logging path without having to
// stage 500+ docs.
func SetBulkChunkSizeForTest(n int) func() {
	prev := bulkChunkSize
	bulkChunkSize = n
	return func() { bulkChunkSize = prev }
}

// DedupePairsForTest wraps dedupePairs so the test in the
// firestore_test external package can exercise the dedup logic
// without standing up a real Firestore client.
func DedupePairsForTest(pairs []model.IoCWithRef) ([]DedupedSummary, error) {
	entries, err := dedupePairs(pairs)
	if err != nil {
		return nil, err
	}
	out := make([]DedupedSummary, len(entries))
	for i, e := range entries {
		out[i].IoCID = e.ioc.ID
		out[i].Raw = e.ioc.Raw
		out[i].LastSeen = e.ioc.LastSeenAt
		for _, r := range e.refs {
			out[i].RefIDs = append(out[i].RefIDs, r.refID)
		}
	}
	return out, nil
}
