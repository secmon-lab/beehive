package repository_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/repository"
	"github.com/secmon-lab/beehive/pkg/repository/firestore"
)

// newMemoryRepo always returns a usable in-memory Repository.
func newMemoryRepo(_ *testing.T) interfaces.Repository {
	return repository.NewMemory()
}

// newFirestoreRepo returns a Firestore-backed Repository when the
// TEST_FIRESTORE_* environment variables are set, and calls t.Skip
// otherwise. Tests using this helper inherit the standard `t.Skip` /
// `Required()` semantics from gt.
func newFirestoreRepo(t *testing.T) interfaces.Repository {
	t.Helper()
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT_ID")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE_ID")
	if projectID == "" || databaseID == "" {
		t.Skip("set TEST_FIRESTORE_PROJECT_ID and TEST_FIRESTORE_DATABASE_ID to run Firestore-backed tests")
	}
	repo, err := firestore.New(t.Context(), projectID, databaseID)
	gt.NoError(t, err).Required()
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return repo
}

// runOnBoth invokes testFn once against the in-memory backend and once
// against Firestore. The Firestore arm self-skips when the env vars are
// missing, so the default `go test ./...` invocation only exercises the
// memory backend.
func runOnBoth(t *testing.T, testFn func(*testing.T, interfaces.Repository)) {
	t.Helper()
	t.Run("Memory", func(t *testing.T) {
		testFn(t, newMemoryRepo(t))
	})
	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreRepo(t))
	})
}
