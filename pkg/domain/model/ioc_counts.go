package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// IoCCounts is the precomputed per-type IoC document count. The
// counter is refreshed at the tail of every fetch run (see
// usecase.RefreshIoCCounts) and the operator UI reads it via
// /api/v1/iocs/stats — so the stats page costs one document read per
// view instead of an aggregation count() per IoC type.
type IoCCounts struct {
	// ByType holds every known IoC type. Implementations MUST
	// populate every type returned by types.AllIoCTypes(), zero
	// included, so callers can render a stable list.
	ByType    map[types.IoCType]int64
	Total     int64
	UpdatedAt time.Time
}

// ZeroIoCCounts returns a zero-valued counter that contains every
// known IoC type set to 0 and an unset UpdatedAt. Used as the fallback
// shape when no counter document has been written yet.
func ZeroIoCCounts() *IoCCounts {
	out := &IoCCounts{
		ByType: make(map[types.IoCType]int64, len(types.AllIoCTypes())),
	}
	for _, t := range types.AllIoCTypes() {
		out.ByType[t] = 0
	}
	return out
}
