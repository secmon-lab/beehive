package types

// RunStatus is the outcome of a Run or a per-source RunSource entry.
type RunStatus string

const (
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	// RunStatusPartial means some of the per-source sub-results failed but
	// at least one succeeded. Only set on the parent Run.
	RunStatusPartial RunStatus = "partial"
	RunStatusFailed  RunStatus = "failed"
	// RunStatusSkipped is only used in `RunSource.Status` to indicate that
	// the source was not actually fetched in this Run.
	RunStatusSkipped RunStatus = "skipped"
)

func (s RunStatus) String() string { return string(s) }

// SkipReason explains why a `RunSource.Status == RunStatusSkipped`.
type SkipReason string

const (
	SkipReasonNotDue   SkipReason = "not_due"
	SkipReasonLocked   SkipReason = "locked"
	SkipReasonDisabled SkipReason = "disabled"
)

func (r SkipReason) String() string { return string(r) }
