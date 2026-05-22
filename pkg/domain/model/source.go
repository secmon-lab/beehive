// Package model contains domain entities. These types are in-memory shapes
// (and, where applicable, Firestore-mirroring shapes). The actual Firestore
// document mapping is handled in pkg/repository/firestore via explicit
// convert functions, NOT via struct tags.
package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Source is a TOML-defined data source. Its single source of truth is the
// TOML files under config/sources/*.toml; it is loaded into memory at
// container start and is NEVER persisted to Firestore (only the dynamic
// state lives in Firestore, see SourceState).
//
// Fields are PascalCase intentionally — see CLAUDE.md "Repository
// implementation policy". TOML keys map to these fields via the struct tags
// added in pkg/service/source_catalog when loading.
type Source struct {
	ID       types.SourceID
	Name     string
	Kind     types.SourceKind
	Type     string // provider identifier (e.g. "rss", "abuseipdb_blacklist")
	URL      string // blog kind only; empty for feed kind
	Interval time.Duration
	// Disabled is true ONLY when the operator explicitly paused the source
	// in TOML. Default value (false) means enabled, so most TOML entries
	// omit this field.
	Disabled bool
}

// EffectiveEnabled returns whether the source should actually be fetched,
// taking the per-instance override into account. See spec §2.9.
func (s *Source) EffectiveEnabled(state *SourceState) bool {
	if state != nil {
		switch state.EnabledOverride {
		case types.OverrideForceOn:
			return true
		case types.OverrideForceOff:
			return false
		}
	}
	return !s.Disabled
}
