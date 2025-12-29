package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/source/rss"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

func runRSSStateRepositoryTest(t *testing.T, repo rss.RSSStateRepository) {
	ctx := context.Background()

	t.Run("save and get RSS state", func(t *testing.T) {
		sourceID := time.Now().Format("rss-20060102-150405.000000")

		state := &rss.RSSState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastArticleID: "article-123",
			LastItemDate:  time.Now().Add(-1 * time.Hour),
			ItemCount:     42,
			ErrorCount:    0,
			LastError:     "",
		}

		gt.NoError(t, repo.SaveRSSState(ctx, state))

		retrieved, err := repo.GetRSSState(ctx, sourceID)
		gt.NoError(t, err)

		// CRITICAL: SourceID must be populated when retrieved from Firestore
		gt.V(t, retrieved.SourceID).Equal(sourceID).Describe("SourceID must be populated")
		gt.V(t, retrieved.LastArticleID).Equal(state.LastArticleID).Describe("LastArticleID")
		gt.N(t, retrieved.ItemCount).Equal(state.ItemCount).Describe("ItemCount")
		gt.N(t, retrieved.ErrorCount).Equal(state.ErrorCount).Describe("ErrorCount")
		gt.S(t, retrieved.LastError).Equal(state.LastError).Describe("LastError")

		gt.True(t, retrieved.LastFetchedAt.Sub(state.LastFetchedAt).Abs() <= time.Second)
		gt.True(t, retrieved.LastItemDate.Sub(state.LastItemDate).Abs() <= time.Second)
		gt.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("update existing RSS state", func(t *testing.T) {
		sourceID := time.Now().Format("rss-20060102-150405.000000")

		initial := &rss.RSSState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastArticleID: "article-100",
			ItemCount:     100,
			ErrorCount:    0,
		}

		gt.NoError(t, repo.SaveRSSState(ctx, initial))
		time.Sleep(10 * time.Millisecond)

		updated := &rss.RSSState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now(),
			LastArticleID: "article-200",
			ItemCount:     200,
			ErrorCount:    5,
			LastError:     "some error occurred",
		}

		gt.NoError(t, repo.SaveRSSState(ctx, updated))

		retrieved, err := repo.GetRSSState(ctx, sourceID)
		gt.NoError(t, err)

		gt.V(t, retrieved.SourceID).Equal(sourceID).Describe("SourceID must be populated after update")
		gt.V(t, retrieved.LastArticleID).Equal("article-200").Describe("LastArticleID updated")
		gt.N(t, retrieved.ItemCount).Equal(int64(200)).Describe("ItemCount updated")
		gt.N(t, retrieved.ErrorCount).Equal(int64(5)).Describe("ErrorCount updated")
		gt.S(t, retrieved.LastError).Equal("some error occurred").Describe("LastError updated")
	})

	t.Run("get non-existent RSS state returns error", func(t *testing.T) {
		sourceID := time.Now().Format("nonexistent-rss-20060102-150405.000000")

		_, err := repo.GetRSSState(ctx, sourceID)
		gt.Error(t, err)
	})

	t.Run("empty source ID returns error on save", func(t *testing.T) {
		state := &rss.RSSState{
			SourceID: "",
		}

		err := repo.SaveRSSState(ctx, state)
		gt.Error(t, err)
	})
}

func TestRSSStateRepository_Memory(t *testing.T) {
	repo := memory.New()
	runRSSStateRepositoryTest(t, repo)
}

func TestRSSStateRepository_Firestore(t *testing.T) {
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

	runRSSStateRepositoryTest(t, repo)
}
