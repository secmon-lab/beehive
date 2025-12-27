package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

func runSourceStateRepositoryTest(t *testing.T, repo interfaces.SourceStateRepository) {
	ctx := context.Background()

	t.Run("save and get source state", func(t *testing.T) {
		// Use random ID based on current timestamp
		sourceID := time.Now().Format("source-20060102-150405.000000")

		// Create test state
		state := &model.SourceState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastItemID:    "item-123",
			LastItemDate:  time.Now().Add(-1 * time.Hour),
			ItemCount:     42,
			ErrorCount:    0,
			LastError:     "",
		}

		// Save state
		gt.NoError(t, repo.SaveState(ctx, state))

		// Retrieve state
		retrieved, err := repo.GetState(ctx, sourceID)
		gt.NoError(t, err)

		// Verify all fields
		gt.Equal(t, retrieved.SourceID, state.SourceID)
		gt.Equal(t, retrieved.LastItemID, state.LastItemID)
		gt.Equal(t, retrieved.ItemCount, state.ItemCount)
		gt.Equal(t, retrieved.ErrorCount, state.ErrorCount)
		gt.Equal(t, retrieved.LastError, state.LastError)

		// Verify timestamps (with tolerance for storage precision)
		gt.True(t, retrieved.LastFetchedAt.Sub(state.LastFetchedAt).Abs() <= time.Second)
		gt.True(t, retrieved.LastItemDate.Sub(state.LastItemDate).Abs() <= time.Second)
		gt.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("update existing state", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")

		// Create initial state
		initial := &model.SourceState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastItemID:    "item-100",
			ItemCount:     100,
			ErrorCount:    0,
		}

		gt.NoError(t, repo.SaveState(ctx, initial))

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Update state
		updated := &model.SourceState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastItemID:    "item-200",
			ItemCount:     200,
			ErrorCount:    5,
			LastError:     "some error occurred",
		}

		gt.NoError(t, repo.SaveState(ctx, updated))

		// Retrieve updated state
		retrieved, err := repo.GetState(ctx, sourceID)
		gt.NoError(t, err)

		// Verify fields were updated
		gt.Equal(t, retrieved.LastItemID, "item-200")
		gt.Equal(t, retrieved.ItemCount, int64(200))
		gt.Equal(t, retrieved.ErrorCount, int64(5))
		gt.Equal(t, retrieved.LastError, "some error occurred")
	})

	t.Run("get non-existent state returns error", func(t *testing.T) {
		sourceID := time.Now().Format("nonexistent-20060102-150405.000000")

		_, err := repo.GetState(ctx, sourceID)
		gt.Error(t, err)
	})

	t.Run("empty source ID returns error", func(t *testing.T) {
		state := &model.SourceState{
			SourceID: "",
		}

		err := repo.SaveState(ctx, state)
		gt.Error(t, err)
	})

	t.Run("incremental updates", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")

		// Initial state
		state := &model.SourceState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			ItemCount:     10,
			ErrorCount:    0,
		}

		gt.NoError(t, repo.SaveState(ctx, state))

		// Simulate multiple fetch operations
		for i := 1; i <= 3; i++ {
			state.ItemCount += 10
			state.LastFetchedAt = time.Now()
			gt.NoError(t, repo.SaveState(ctx, state))
		}

		// Verify final state
		final, err := repo.GetState(ctx, sourceID)
		gt.NoError(t, err)
		gt.Equal(t, final.ItemCount, int64(40))
	})

	t.Run("error tracking", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")

		state := &model.SourceState{
			SourceID:   sourceID,
			ErrorCount: 0,
			LastError:  "",
		}

		gt.NoError(t, repo.SaveState(ctx, state))

		// Simulate error
		state.ErrorCount++
		state.LastError = "connection timeout"

		gt.NoError(t, repo.SaveState(ctx, state))

		retrieved, err := repo.GetState(ctx, sourceID)
		gt.NoError(t, err)
		gt.Equal(t, retrieved.ErrorCount, int64(1))
		gt.Equal(t, retrieved.LastError, "connection timeout")
	})
}

func TestSourceStateRepository_Memory(t *testing.T) {
	repo := memory.New()
	runSourceStateRepositoryTest(t, repo)
}

func TestSourceStateRepository_Firestore(t *testing.T) {
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT_ID")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE_ID")

	if projectID == "" || databaseID == "" {
		t.Skip("TEST_FIRESTORE_PROJECT_ID and TEST_FIRESTORE_DATABASE_ID environment variables not set")
	}

	ctx := context.Background()
	repo, err := firestoreRepo.New(ctx, projectID, firestoreRepo.WithDatabaseID(databaseID))
	if err != nil {
		t.Fatalf("failed to create Firestore repository: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			t.Errorf("failed to close repository: %v", err)
		}
	}()

	runSourceStateRepositoryTest(t, repo)
}
