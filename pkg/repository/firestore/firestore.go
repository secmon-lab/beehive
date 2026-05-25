// Package firestore is the production Repository implementation backed by
// Cloud Firestore. The shared test harness in pkg/repository runs the
// same assertions against this implementation and against the in-memory
// one to keep them in lock-step (see CLAUDE.md §10).
//
// All field <-> document mapping is done via Go's default reflection
// rules — no `firestore:"..."` struct tags are used, in line with the
// repository policy. Where the natural reflection mapping is awkward
// (e.g. nested time slicing), explicit *FromDoc / *ToDoc helpers live in
// this package.
package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Collection names mirror the spec §2.4 table. Defined as constants so
// that callers and tests share a single source of truth.
const (
	collectionStates   = "states"
	collectionArticles = "articles"
	collectionIoCs     = "iocs"
	collectionRefs     = "refs" // sub-collection under iocs/{IoCID}
	collectionRuns     = "runs"
	collectionLocks    = "locks"
	subLockEntries     = "entries" // sub-collection under locks/{Kind}
	collectionMetrics  = "metrics"
	docIoCCounts       = "ioc_counts" // doc id under collectionMetrics
)

// Firestore is the Repository backed by Cloud Firestore.
type Firestore struct {
	client *firestore.Client

	// holderID identifies this process in lock documents. Derived from
	// hostname + PID so concurrent acquires from the same machine are
	// still distinguishable across instances.
	holderID string
}

// New constructs a Firestore Repository connected to projectID / databaseID.
// Use `firestore.DetectProjectID` upstream if the caller wants ADC.
//
// We explicitly pin the quota project via option.WithQuotaProject so that
// gRPC routes both the data and the quota charges to projectID. Without
// this, ADC may carry a different `quota_project_id` (set via
// `gcloud auth application-default set-quota-project ...`) and the
// server returns the misleading
//
//	FailedPrecondition: Firestore API data access is disabled.
//	  Please enable it if you want to access the database through Firestore API.
//
// — the API IS enabled on projectID, but the error is reported against
// the *quota* project, which isn't.
func New(ctx context.Context, projectID, databaseID string) (*Firestore, error) {
	if projectID == "" {
		return nil, goerr.New("firestore project_id is required", goerr.T(errutil.TagInvalidInput))
	}
	if databaseID == "" {
		return nil, goerr.New("firestore database is required", goerr.T(errutil.TagInvalidInput))
	}
	client, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID,
		option.WithQuotaProject(projectID),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "new firestore client",
			goerr.V("project_id", projectID),
			goerr.V("database", databaseID))
	}
	host, _ := os.Hostname()
	return &Firestore{
		client:   client,
		holderID: fmt.Sprintf("%s/%d", host, os.Getpid()),
	}, nil
}

// Close releases the underlying gRPC connection. Safe to call multiple
// times.
func (f *Firestore) Close() error {
	if f == nil || f.client == nil {
		return nil
	}
	return f.client.Close()
}

// Compile-time check.
var _ interfaces.Repository = (*Firestore)(nil)

// ---- helpers ----

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return true
	}
	if errors.Is(err, iterator.Done) {
		return true
	}
	return false
}

// isAlreadyExists tells whether err is the Firestore "doc already exists"
// failure. BulkWriter surfaces it from a Create() job whose target doc
// pre-exists; the bulk upsert path uses this to fall back to an Update.
func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
		return true
	}
	return false
}

func wrapNotFound(err error, what string, key any) error {
	if isNotFound(err) {
		return goerr.New(what+" not found",
			goerr.V("key", key),
			goerr.T(errutil.TagNotFound),
		)
	}
	return goerr.Wrap(err, what, goerr.V("key", key))
}
