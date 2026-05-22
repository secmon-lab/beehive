package model

import "time"

// Lock is the document stored at locks/{Kind}/entries/{TargetID}. It is a
// heartbeat-based mutex: the holder bumps `ExpiresAt` on a short interval
// while it runs, and other instances may take over once `ExpiresAt` is in
// the past.
type Lock struct {
	Kind            string
	TargetID        string
	Holder          string // hostname + pid + goroutine id
	AcquiredAt      time.Time
	ExpiresAt       time.Time
	LastHeartbeatAt time.Time
}

// LockKindFetch is the lock kind used by per-source fetches.
const LockKindFetch = "fetch"
