package model

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// IoC is a deduplicated indicator. Stored at Firestore path iocs/{ID}.
//
// `Value` is the NORMALIZED form (lower-cased, defang-stripped, idna for
// domains, etc.) and is the lookup key. `Raw` keeps the original surface
// form from the first time we encountered the indicator — it is set ONLY
// on the very first insert (upsert path treats it as immutable) so that
// later defang differences in other articles do not overwrite the original
// observation.
type IoC struct {
	ID          types.IoCID
	Type        types.IoCType
	Value       string // normalized
	Raw         string // first-seen raw value (immutable after creation)
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

// ComputeIoCID returns the canonical document ID for a (type, normalized
// value) pair. Using the full sha256 hex (64 chars) avoids any collision
// concerns and keeps the lookup endpoint deterministic.
func ComputeIoCID(t types.IoCType, value string) types.IoCID {
	sum := sha256.Sum256([]byte(string(t) + "\t" + value))
	return types.IoCID(hex.EncodeToString(sum[:]))
}

// IoCRef records a single observation of an IoC. Stored at
// iocs/{IoCID}/refs/{RefID}.
//
// For blog sources, ArticleID is set and Context holds the surrounding
// sentence(s) the LLM identified. For feed sources, RunID is set instead
// and Context is empty.
type IoCRef struct {
	SourceID   types.SourceID
	SourceKind types.SourceKind
	ArticleID  types.ArticleID // blog only
	RunID      types.RunID     // feed only
	Context    string          // blog only
	Raw        string          // optional original surface form
	Confidence float64
	SeenAt     time.Time
}

// ComputeRefID returns the deterministic document ID for an occurrence
// record. It is built from the natural composite key of the occurrence so
// that re-running the same fetch is idempotent (no duplicate ref docs).
//
// For blog: SourceID + ":" + ArticleID.
// For feed: SourceID + ":" + RunID.
func ComputeRefID(sourceID types.SourceID, articleID types.ArticleID, runID types.RunID) types.RefID {
	target := string(articleID)
	if target == "" {
		target = string(runID)
	}
	sum := sha256.Sum256([]byte(string(sourceID) + ":" + target))
	return types.RefID(hex.EncodeToString(sum[:]))
}

// IoCWithRef pairs an IoC with the IoCRef that documents this particular
// occurrence. The bulk write path (Repository.BulkUpsertIoCs) takes a
// slice of these so feed-kind sources can persist tens of thousands of
// indicators in a single repository-level batch.
type IoCWithRef struct {
	IoC *IoC
	Ref *IoCRef
}
