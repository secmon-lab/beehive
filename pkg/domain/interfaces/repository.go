// Package interfaces collects the boundary contracts the use cases depend
// on. Concrete implementations live under pkg/repository (Firestore +
// in-memory) and pkg/service.
package interfaces

import (
	"context"

	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Repository is the union of all persistence operations beehive needs.
// Kept as one interface rather than per-aggregate ones to avoid a kitchen
// sink of small interfaces in usecase signatures; we have only two impls
// anyway. Implementations MUST be safe for concurrent use.
type Repository interface {
	StateRepository
	ArticleRepository
	IoCRepository
	RunRepository
	LockRepository
}

// ---- states/{SourceID} ----

type StateRepository interface {
	GetSourceState(ctx context.Context, id types.SourceID) (*model.SourceState, error)
	// ListSourceStates returns all known states (used by bootstrap to detect
	// orphan states that no longer have a TOML entry).
	ListSourceStates(ctx context.Context) ([]*model.SourceState, error)
	// UpdateSourceState upserts only the dynamic fields that the caller
	// has set on `state` (zero-valued fields are not written, so callers
	// must set every field they want to change). The repository skips the
	// write if nothing actually changed.
	UpdateSourceState(ctx context.Context, state *model.SourceState) error
	// SetEnabledOverride writes only the EnabledOverride sub-field so that
	// the rest of the state document is left untouched.
	SetEnabledOverride(ctx context.Context, id types.SourceID, override types.EnabledOverride) error
}

// ---- articles/{ArticleID} ----

type ArticleRepository interface {
	GetArticleByURL(ctx context.Context, url string) (*model.Article, error)
	CreateArticle(ctx context.Context, a *model.Article) error
	UpdateArticle(ctx context.Context, a *model.Article) error
}

// ---- iocs/{ID} and iocs/{ID}/refs/{RefID} ----

type IoCRepository interface {
	GetIoC(ctx context.Context, id types.IoCID) (*model.IoC, error)
	// UpsertIoCWithRef writes an IoC and its ref idempotently:
	//   * If the IoC doc does not exist, it is created (including Raw).
	//   * If it exists, only LastSeenAt is updated; Raw is preserved.
	//   * If the ref doc already exists for the same RefID, nothing is
	//     written for the ref.
	UpsertIoCWithRef(ctx context.Context, ioc *model.IoC, ref *model.IoCRef) error
	// BulkUpsertIoCs persists many (IoC, IoCRef) pairs in one
	// repository-level batch. Each pair has the same upsert semantics as
	// UpsertIoCWithRef:
	//   * If the IoC does not exist, it is created (Raw included).
	//   * If it exists, only LastSeenAt is bumped; Raw stays immutable.
	//   * If the ref already exists for the same RefID, the ref write is
	//     skipped.
	// Implementations are free to fan out / chunk / pipeline writes,
	// and MUST deduplicate the input by (IoC.ID, RefID) — the same
	// IoC.ID may appear in multiple pairs, but the implementation must
	// not stage more than one write to the same document path.
	//
	// On success returns the count of distinct IoCs that the
	// implementation intended to persist (input deduplicated by
	// IoC.ID). The caller uses this for accurate `IoCCount` reporting
	// even when the input contained duplicates (e.g. urlhaus emitting
	// the same URL twice in one feed dump).
	//
	// The returned error is the first failure encountered; subsequent
	// writes within the same batch MAY or MAY NOT have been applied —
	// callers MUST treat the batch as having unspecified partial state
	// and rely on re-fetch idempotency to make progress. On error the
	// returned count still reflects the *intended* deduped IoC count
	// (matches the success case shape), not how many actually landed.
	BulkUpsertIoCs(ctx context.Context, pairs []model.IoCWithRef) (persisted int, err error)
	// ListRecentIoCs returns up to `limit` IoCs ordered by LastSeenAt
	// descending. Intended for the operator-facing UI; bulk analytics
	// still go through BigQuery so this method has no pagination cursor
	// — callers should pick a sane cap (the HTTP handler enforces an
	// upper bound).
	ListRecentIoCs(ctx context.Context, limit int) ([]*model.IoC, error)
}

// ---- runs/{ID} ----

type RunRepository interface {
	CreateRun(ctx context.Context, r *model.Run) error
	UpdateRun(ctx context.Context, r *model.Run) error
	// AppendRunSource transactionally appends one RunSource to Run.Sources
	// and adds its SourceID to Run.SourceIDs. Implementations MAY also
	// bump per-Run counters in the same transaction.
	AppendRunSource(ctx context.Context, runID types.RunID, src *model.RunSource) error
	GetRun(ctx context.Context, id types.RunID) (*model.Run, error)
}

// ---- locks/{Kind}/entries/{TargetID} ----

type LockRepository interface {
	// TryAcquireLock attempts to create or take over a lock document. It
	// returns (acquired=true, nil) on success, (false, nil) if another
	// holder currently owns it, or (false, err) on a real error.
	TryAcquireLock(ctx context.Context, lock *model.Lock) (bool, error)
	// RenewLock bumps ExpiresAt / LastHeartbeatAt. Returns ErrLockLost if
	// the lock has been taken over by another holder.
	RenewLock(ctx context.Context, lock *model.Lock) error
	// ReleaseLock deletes the lock document. Safe to call even if the
	// lock has already expired.
	ReleaseLock(ctx context.Context, kind, targetID string) error
}
