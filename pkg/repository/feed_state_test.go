package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/source/feed"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

func runFeedStateRepositoryTest(t *testing.T, repo feed.FeedStateRepository) {
	ctx := context.Background()

	t.Run("save and get Feed state", func(t *testing.T) {
		sourceID := time.Now().Format("feed-20060102-150405.000000")

		state := &feed.FeedState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			ItemCount:     42,
			ErrorCount:    0,
			LastError:     "",
		}

		gt.NoError(t, repo.SaveFeedState(ctx, state))

		retrieved, err := repo.GetFeedState(ctx, sourceID)
		gt.NoError(t, err)

		// CRITICAL: SourceID must be populated when retrieved from Firestore
		gt.V(t, retrieved.SourceID).Equal(sourceID).Describe("SourceID must be populated")
		gt.N(t, retrieved.ItemCount).Equal(state.ItemCount).Describe("ItemCount")
		gt.N(t, retrieved.ErrorCount).Equal(state.ErrorCount).Describe("ErrorCount")
		gt.S(t, retrieved.LastError).Equal(state.LastError).Describe("LastError")

		gt.True(t, retrieved.LastFetchedAt.Sub(state.LastFetchedAt).Abs() <= time.Second)
		gt.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("update existing Feed state", func(t *testing.T) {
		sourceID := time.Now().Format("feed-20060102-150405.000000")

		initial := &feed.FeedState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			ItemCount:     100,
			ErrorCount:    0,
		}

		gt.NoError(t, repo.SaveFeedState(ctx, initial))
		time.Sleep(10 * time.Millisecond)

		updated := &feed.FeedState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			ItemCount:     200,
			ErrorCount:    5,
			LastError:     "some error occurred",
		}

		gt.NoError(t, repo.SaveFeedState(ctx, updated))

		retrieved, err := repo.GetFeedState(ctx, sourceID)
		gt.NoError(t, err)

		gt.V(t, retrieved.SourceID).Equal(sourceID).Describe("SourceID must be populated after update")
		gt.N(t, retrieved.ItemCount).Equal(int64(200)).Describe("ItemCount updated")
		gt.N(t, retrieved.ErrorCount).Equal(int64(5)).Describe("ErrorCount updated")
		gt.S(t, retrieved.LastError).Equal("some error occurred").Describe("LastError updated")
	})

	t.Run("get non-existent Feed state returns error", func(t *testing.T) {
		sourceID := time.Now().Format("nonexistent-feed-20060102-150405.000000")

		_, err := repo.GetFeedState(ctx, sourceID)
		gt.Error(t, err)
	})

	t.Run("empty source ID returns error on save", func(t *testing.T) {
		state := &feed.FeedState{
			SourceID: "",
		}

		err := repo.SaveFeedState(ctx, state)
		gt.Error(t, err)
	})
}

func TestFeedStateRepository_Memory(t *testing.T) {
	repo := memory.New()
	runFeedStateRepositoryTest(t, repo)
}

func TestFeedStateRepository_Firestore(t *testing.T) {
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

	runFeedStateRepositoryTest(t, repo)
}
