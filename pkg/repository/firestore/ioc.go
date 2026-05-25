package firestore

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// bulkChunkSize controls how many distinct IoCs are processed per
// chunk: one GetAll RPC to classify create-vs-update, then one
// BulkWriter session that drains both kinds. Kept as a package var so
// tests can shrink it without staging 500+ docs.
//
// Firestore's BulkWriter internally batches at 500 writes per RPC and
// BatchGet caps at 1000 reads per RPC, so 500 fits cleanly in both
// dimensions and yields one progress log line per ~1–2s of work.
var bulkChunkSize = 500

// ListRecentIoCs returns up to `limit` IoCs ordered by LastSeenAt
// descending. Backed by Firestore's auto-created single-field index on
// LastSeenAt (descending) — no manual index setup is required.
func (f *Firestore) ListRecentIoCs(ctx context.Context, limit int) ([]*model.IoC, error) {
	return f.ListRecentIoCsAfter(ctx, limit, nil)
}

// ListRecentIoCsAfter pages over the (LastSeenAt DESC, __name__ DESC)
// stream. The cursor uses both fields because feed-kind sources stamp
// every IoC of a batch with the same LastSeenAt (see
// usecase/fetch.go::bulkPersistSeeds); a LastSeenAt-only cursor would
// silently drop every same-timestamp document past the page break.
//
// Both order terms are descending so that the query is served by
// Firestore's auto-created single-field index on LastSeenAt — the
// implicit __name__ tiebreaker matches the primary direction, so no
// composite index needs to be provisioned. Mixing directions
// (LastSeenAt DESC + __name__ ASC) would force a composite index,
// which this project intentionally avoids.
func (f *Firestore) ListRecentIoCsAfter(ctx context.Context, limit int, after *model.IoCListCursor) ([]*model.IoC, error) {
	if limit <= 0 {
		return nil, nil
	}
	q := f.client.Collection(collectionIoCs).
		OrderBy("LastSeenAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Limit(limit)
	if after != nil {
		q = q.StartAfter(after.LastSeenAt, string(after.ID))
	}
	iter := q.Documents(ctx)
	defer iter.Stop()

	out := make([]*model.IoC, 0, limit)
	for {
		doc, err := iter.Next()
		if errors.Is(err, ErrIteratorDone()) {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "iterate iocs")
		}
		var i model.IoC
		if err := doc.DataTo(&i); err != nil {
			return nil, goerr.Wrap(err, "decode ioc", goerr.V("id", doc.Ref.ID))
		}
		out = append(out, &i)
	}
	return out, nil
}

// CountIoCsOfType runs one Firestore aggregation `count()` query for
// a single IoC type. The `WHERE Type == "<type>"` filter rides
// Firestore's single-field `Type` index, so the server side does not
// scan the whole collection — only the matching subset is aggregated.
//
// Billing-wise: Firestore charges `count()` in chunks of 1000 matched
// documents. With N total docs of this type, one call costs about
// ceil(N / 1000) read units. The fan-out across every IoC type lives
// in usecase.RefreshIoCCounts, which calls this concurrently and
// persists the rolled-up result via SaveIoCCounts.
func (f *Firestore) CountIoCsOfType(ctx context.Context, t types.IoCType) (int64, error) {
	q := f.client.Collection(collectionIoCs).Where("Type", "==", string(t))
	res, err := q.NewAggregationQuery().WithCount("c").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "aggregation count", goerr.V("type", t))
	}
	v, ok := res["c"].(*firestorepb.Value)
	if !ok {
		return 0, goerr.New("unexpected aggregation result shape",
			goerr.V("type", t),
			goerr.V("got", res["c"]))
	}
	return v.GetIntegerValue(), nil
}

// iocCountsDoc is the on-disk shape of metrics/ioc_counts. We map
// types.IoCType to string explicitly because Firestore decoders
// don't handle named string types as map keys gracefully.
type iocCountsDoc struct {
	ByType    map[string]int64
	Total     int64
	UpdatedAt time.Time
}

// GetIoCCounts reads metrics/ioc_counts. Returns the zero-valued shape
// (every type set to 0, UpdatedAt unset) when the document has not
// been written yet — that case typically means RefreshIoCCounts hasn't
// run since the data was inserted.
func (f *Firestore) GetIoCCounts(ctx context.Context) (*model.IoCCounts, error) {
	doc, err := f.client.Collection(collectionMetrics).Doc(docIoCCounts).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return model.ZeroIoCCounts(), nil
		}
		return nil, goerr.Wrap(err, "get ioc counts")
	}
	var raw iocCountsDoc
	if err := doc.DataTo(&raw); err != nil {
		return nil, goerr.Wrap(err, "decode ioc counts")
	}
	out := model.ZeroIoCCounts()
	out.Total = raw.Total
	out.UpdatedAt = raw.UpdatedAt
	for k, v := range raw.ByType {
		out.ByType[types.IoCType(k)] = v
	}
	return out, nil
}

// SaveIoCCounts replaces metrics/ioc_counts atomically (single-doc
// Set). Called by usecase.RefreshIoCCounts at the tail of every fetch
// run, never on a hot read path.
func (f *Firestore) SaveIoCCounts(ctx context.Context, counts *model.IoCCounts) error {
	if counts == nil {
		return goerr.New("counts is nil", goerr.T(errutil.TagInvalidInput))
	}
	raw := iocCountsDoc{
		ByType:    make(map[string]int64, len(counts.ByType)),
		Total:     counts.Total,
		UpdatedAt: counts.UpdatedAt,
	}
	for k, v := range counts.ByType {
		raw.ByType[string(k)] = v
	}
	_, err := f.client.Collection(collectionMetrics).Doc(docIoCCounts).Set(ctx, raw)
	if err != nil {
		return goerr.Wrap(err, "save ioc counts")
	}
	return nil
}

func (f *Firestore) GetIoC(ctx context.Context, id types.IoCID) (*model.IoC, error) {
	doc, err := f.client.Collection(collectionIoCs).Doc(string(id)).Get(ctx)
	if err != nil {
		return nil, wrapNotFound(err, "ioc", id)
	}
	var i model.IoC
	if err := doc.DataTo(&i); err != nil {
		return nil, goerr.Wrap(err, "decode ioc", goerr.V("id", id))
	}
	return &i, nil
}

// BulkUpsertIoCs persists many (IoC, ref) pairs efficiently. Designed
// for feed-kind sources that return thousands of indicators per fetch —
// calling UpsertIoCWithRef in a loop there would fire one transactional
// RPC per pair (~50–100ms × 10k+ = tens of minutes).
//
// Strategy (per chunk):
//  1. Bulk-read existing IoC docs via `client.GetAll` (one RPC for the
//     whole chunk).
//  2. One BulkWriter session: stage `Create` for IoCs that do not
//     exist (Raw included), `Update {LastSeenAt}` for those that do
//     (Raw preserved). Each document path is touched at most once
//     per session, which sidesteps BulkWriter's "duplicate write for
//     path" rejection.
//  3. Refs live in a sub-collection so their paths never collide with
//     the IoC writes. `Create` is used; `AlreadyExists` is a silent
//     no-op (natural-key dedup).
//
// Trade-off vs UpsertIoCWithRef:
//   - We give up the read-conditional "skip identical-value write"
//     optimisation per IoC; bulk mode always issues 1 Update for
//     existing IoCs. The throughput gain dominates the small extra
//     write cost.
//   - Cross-source race (different sources persisting the same IoC at
//     the same time) can lose a LastSeenAt bump: the loser's Create
//     hits AlreadyExists. We log the count as `chunk_raced` and move
//     on — the next fetch corrects it (CLAUDE.md §6: re-fetches are
//     cheap and idempotent).
//
// Returns the count of distinct IoCs in the (deduped) input — what
// the caller uses for `IoCCount` reporting.
func (f *Firestore) BulkUpsertIoCs(ctx context.Context, pairs []model.IoCWithRef) (int, error) {
	if len(pairs) == 0 {
		return 0, nil
	}
	logger := logging.From(ctx)

	entries, err := dedupePairs(pairs)
	if err != nil {
		return 0, err
	}
	total := len(entries)
	started := time.Now()

	logger.LogAttrs(ctx, slog.LevelDebug, "ioc: bulk upsert start",
		slog.Int("input_pairs", len(pairs)),
		slog.Int("deduped_iocs", total),
		slog.Int("chunk_size", bulkChunkSize),
	)

	for offset := 0; offset < total; offset += bulkChunkSize {
		end := min(offset+bulkChunkSize, total)
		chunk := entries[offset:end]
		if cerr := f.bulkUpsertChunk(ctx, chunk, started, end, total); cerr != nil {
			// Return the intended deduped count even on error so the
			// caller's IoCCount metric stays consistent across success
			// and failure paths.
			return total, cerr
		}
	}

	logger.LogAttrs(ctx, slog.LevelDebug, "ioc: bulk upsert done",
		slog.Int("persisted", total),
		slog.Duration("total_elapsed", time.Since(started)),
	)
	return total, nil
}

func (f *Firestore) bulkUpsertChunk(ctx context.Context, chunk []*iocEntry, started time.Time, done, total int) error {
	logger := logging.From(ctx)
	chunkStart := time.Now()

	// 1. Classify: which IoCs already exist?
	iocRefs := make([]*firestore.DocumentRef, len(chunk))
	for i, e := range chunk {
		iocRefs[i] = f.client.Collection(collectionIoCs).Doc(string(e.ioc.ID))
	}
	snaps, err := f.client.GetAll(ctx, iocRefs)
	if err != nil {
		return goerr.Wrap(err, "bulk get iocs for classification",
			goerr.V("chunk_size", len(chunk)))
	}

	// 2. Single BulkWriter session: Create-new XOR Update-existing per
	//    IoC path, plus Create per ref path. No path gets two writes.
	bw := f.client.BulkWriter(ctx)
	type job struct {
		e       *iocEntry
		iocJob  *firestore.BulkWriterJob
		refJobs []*firestore.BulkWriterJob
		wasNew  bool
	}
	jobs := make([]job, len(chunk))
	var newCount, existingCount int
	for i, snap := range snaps {
		e := chunk[i]
		wasNew := !snap.Exists()
		var iocJob *firestore.BulkWriterJob
		switch {
		case wasNew:
			var jerr error
			iocJob, jerr = bw.Create(iocRefs[i], e.ioc)
			if jerr != nil {
				bw.End()
				return goerr.Wrap(jerr, "stage ioc create",
					goerr.V("id", e.ioc.ID))
			}
			newCount++
		case !e.ioc.LastSeenAt.IsZero():
			// Mirror UpsertIoCWithRef's defensive "don't bump backwards
			// or with zero" rule: only Update when the caller actually
			// supplied a LastSeenAt. A zero-value seed (programming bug
			// or partially-filled IoC) should never clobber the stored
			// timestamp.
			var jerr error
			iocJob, jerr = bw.Update(iocRefs[i], []firestore.Update{
				{Path: "LastSeenAt", Value: e.ioc.LastSeenAt},
			})
			if jerr != nil {
				bw.End()
				return goerr.Wrap(jerr, "stage ioc lastseenat update",
					goerr.V("id", e.ioc.ID))
			}
			existingCount++
		default:
			// Existing IoC, no LastSeenAt update requested. Leave the
			// IoC doc alone; refs still get staged below.
			existingCount++
		}
		jobs[i] = job{e: e, iocJob: iocJob, wasNew: wasNew}
		for _, r := range e.refs {
			refDocRef := iocRefs[i].Collection(collectionRefs).Doc(string(r.refID))
			refJob, rerr := bw.Create(refDocRef, r.ref)
			if rerr != nil {
				bw.End()
				return goerr.Wrap(rerr, "stage ref create",
					goerr.V("id", e.ioc.ID))
			}
			jobs[i].refJobs = append(jobs[i].refJobs, refJob)
		}
	}
	bw.End()

	// 3. Drain results. AlreadyExists on a Create is a cross-source
	//    race — accept it (doc exists, which is the desired state) and
	//    move on. AlreadyExists on a ref Create is the natural-key
	//    dedup case (CLAUDE.md §3) and is also silent.
	var raced int
	for _, j := range jobs {
		// iocJob is nil when we deliberately skipped the IoC write
		// (existing doc + zero LastSeenAt). Refs still need draining.
		if j.iocJob != nil {
			if _, rerr := j.iocJob.Results(); rerr != nil {
				if j.wasNew && isAlreadyExists(rerr) {
					raced++
					continue
				}
				return goerr.Wrap(rerr, "write ioc",
					goerr.V("id", j.e.ioc.ID),
					goerr.V("new", j.wasNew))
			}
		}
		for _, rj := range j.refJobs {
			if _, rerr := rj.Results(); rerr != nil && !isAlreadyExists(rerr) {
				return goerr.Wrap(rerr, "create ref",
					goerr.V("id", j.e.ioc.ID))
			}
		}
	}

	logger.LogAttrs(ctx, slog.LevelDebug, "ioc: bulk upsert progress",
		slog.Int("done", done),
		slog.Int("total", total),
		slog.Int("chunk_new", newCount),
		slog.Int("chunk_existing", existingCount),
		slog.Int("chunk_raced", raced),
		slog.Duration("chunk_elapsed", time.Since(chunkStart)),
		slog.Duration("total_elapsed", time.Since(started)),
	)
	return nil
}

// iocEntry is the deduplicated form: one entry per distinct IoC.ID,
// carrying all distinct refs observed for it.
type iocEntry struct {
	ioc  *model.IoC
	refs []refEntry
}

type refEntry struct {
	ref   *model.IoCRef
	refID types.RefID
}

// dedupePairs collapses (IoC.ID, RefID) duplicates so the chunked
// writer only ever stages one operation per Firestore path. Even with
// upstream dedup at the usecase layer, the repository contract
// requires defensive dedup so a buggy caller cannot trigger the
// `BulkWriter received duplicate write for path` failure mode.
//
// Semantics on duplicate IoC.ID: keep the entry whose LastSeenAt is
// most recent. Raw is preserved from the first occurrence (matches
// UpsertIoCWithRef's immutability rule for Raw).
func dedupePairs(pairs []model.IoCWithRef) ([]*iocEntry, error) {
	byID := make(map[types.IoCID]*iocEntry, len(pairs))
	seenRef := make(map[string]bool, len(pairs))
	ordered := make([]*iocEntry, 0, len(pairs))
	for i := range pairs {
		p := &pairs[i]
		if p.IoC == nil || p.IoC.ID == "" {
			return nil, goerr.New("ioc id is empty",
				goerr.V("index", i),
				goerr.T(errutil.TagInvalidInput))
		}
		e, ok := byID[p.IoC.ID]
		if !ok {
			e = &iocEntry{ioc: p.IoC}
			byID[p.IoC.ID] = e
			ordered = append(ordered, e)
		} else if p.IoC.LastSeenAt.After(e.ioc.LastSeenAt) {
			// Build a shallow copy that keeps the first-seen Raw and
			// bumps LastSeenAt to the latest observed value.
			merged := *e.ioc
			merged.LastSeenAt = p.IoC.LastSeenAt
			e.ioc = &merged
		}
		if p.Ref != nil {
			refID := model.ComputeRefID(p.Ref.SourceID, p.Ref.ArticleID, p.Ref.RunID)
			key := string(p.IoC.ID) + ":" + string(refID)
			if !seenRef[key] {
				seenRef[key] = true
				e.refs = append(e.refs, refEntry{ref: p.Ref, refID: refID})
			}
		}
	}
	return ordered, nil
}

// UpsertIoCWithRef implements the cost-optimised upsert: new IoC -> full
// create (Raw fixed); existing IoC -> only bump LastSeenAt; new ref ->
// create; existing ref -> no-op.
//
// Firestore transactions require **all reads to happen before all
// writes** ("firestore: read after write in transaction" otherwise), so
// we gather both snapshots first and execute every mutation at the end.
func (f *Firestore) UpsertIoCWithRef(ctx context.Context, ioc *model.IoC, ref *model.IoCRef) error {
	if ioc == nil || ioc.ID == "" {
		return goerr.New("ioc id is empty", goerr.T(errutil.TagInvalidInput))
	}
	iocRef := f.client.Collection(collectionIoCs).Doc(string(ioc.ID))

	var refRef *firestore.DocumentRef
	if ref != nil {
		refID := model.ComputeRefID(ref.SourceID, ref.ArticleID, ref.RunID)
		refRef = iocRef.Collection(collectionRefs).Doc(string(refID))
	}

	return f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// ---- reads (must happen before any write) ----
		iocSnap, iocErr := tx.Get(iocRef)
		if iocErr != nil && !isNotFound(iocErr) {
			return goerr.Wrap(iocErr, "get ioc for upsert", goerr.V("id", ioc.ID))
		}

		var refExists bool
		if refRef != nil {
			_, rerr := tx.Get(refRef)
			switch {
			case rerr == nil:
				refExists = true
			case isNotFound(rerr):
				refExists = false
			default:
				return goerr.Wrap(rerr, "get existing ref")
			}
		}

		// ---- writes ----
		if iocErr == nil {
			// Existing IoC — bump LastSeenAt only if it is strictly newer.
			var prev model.IoC
			if err := iocSnap.DataTo(&prev); err != nil {
				return goerr.Wrap(err, "decode existing ioc", goerr.V("id", ioc.ID))
			}
			if !ioc.LastSeenAt.IsZero() && ioc.LastSeenAt.After(prev.LastSeenAt) {
				if err := tx.Update(iocRef, []firestore.Update{
					{Path: "LastSeenAt", Value: ioc.LastSeenAt},
				}); err != nil {
					return goerr.Wrap(err, "update LastSeenAt")
				}
			}
		} else {
			// New IoC — create with the full payload (Raw included).
			if err := tx.Create(iocRef, ioc); err != nil {
				return goerr.Wrap(err, "create ioc", goerr.V("id", ioc.ID))
			}
		}

		if refRef != nil && !refExists {
			if err := tx.Create(refRef, ref); err != nil {
				return goerr.Wrap(err, "create ref")
			}
		}
		return nil
	})
}
