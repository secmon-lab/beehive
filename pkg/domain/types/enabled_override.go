package types

// EnabledOverride is a per-source state field that lets operators short-
// circuit the TOML-derived `Disabled` flag without editing TOML.
//
//   - OverrideNone     : follow TOML (default)
//   - OverrideForceOff : disable the source even if TOML says enabled
//   - OverrideForceOn  : enable the source even if TOML says disabled
//
// See §2.9 effectiveEnabled() in the spec for the evaluation order.
type EnabledOverride string

const (
	OverrideNone     EnabledOverride = "none"
	OverrideForceOff EnabledOverride = "force_off"
	OverrideForceOn  EnabledOverride = "force_on"
)

func (o EnabledOverride) String() string { return string(o) }

func (o EnabledOverride) Valid() bool {
	switch o {
	case OverrideNone, OverrideForceOff, OverrideForceOn, "":
		return true
	default:
		return false
	}
}
