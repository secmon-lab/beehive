package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Run is the record of one POST /api/v1/fetch invocation (or one equivalent
// CLI / manual trigger). Per-source results are kept inline in `Sources`
// rather than as a sub-collection, so that the entire summary of one
// scheduler tick is readable as a single document.
//
// `SourceIDs` denormalises the source identifiers so that a Firestore
// array-contains query can pull "all Runs that touched source X".
type Run struct {
	ID           types.RunID
	Trigger      string // "api" | "manual" | "cli"
	Status       types.RunStatus
	StartedAt    time.Time
	FinishedAt   time.Time
	DurationMs   int64
	Total        int
	Triggered    int
	Skipped      int
	Failed       int
	ErrorMessage string // full text, repository truncates near 1 MiB
	Sources      []RunSource
	SourceIDs    []types.SourceID
}

// RunSource is the per-source outcome inside a Run. Embedded as an element
// of Run.Sources.
type RunSource struct {
	SourceID        types.SourceID
	SourceKind      types.SourceKind
	Status          types.RunStatus
	SkipReason      types.SkipReason // populated only when Status == Skipped
	StartedAt       time.Time
	FinishedAt      time.Time
	ArticleCount    int // blog only
	NewArticleCount int // blog only
	IoCCount        int
	ErrorMessage    string // full text, repository truncates near 1 MiB
}
