package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// SourceState is the dynamic, Firestore-backed part of a Source. It is kept
// separate from Source because the SSoT for the static metadata is the TOML
// catalog, and only the things that change at runtime (last fetched time,
// override flag, etc.) belong in the database.
//
// Stored at Firestore path: states/{SourceID}.
type SourceState struct {
	ID              types.SourceID
	EnabledOverride types.EnabledOverride
	LastFetchedAt   time.Time
	LastStatus      types.RunStatus
	// LastError stores the FULL error text including goerr.V key/values.
	// The repository layer truncates only when the document size approaches
	// the 1 MiB Firestore limit.
	LastError     string
	LastRunID     types.RunID
	TotalIoCCount int
	UpdatedAt     time.Time
}
