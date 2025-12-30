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

func runHistoryRepositoryTest(t *testing.T, repo interfaces.HistoryRepository) {
	ctx := context.Background()

	t.Run("save and get history", func(t *testing.T) {
		// Use timestamp-based unique IDs
		now := time.Now()
		sourceID := now.Format("source-20060102-150405.000000")
		historyID := model.GenerateHistoryID()

		// Create test history
		history := &model.History{
			ID:             historyID,
			SourceID:       sourceID,
			SourceType:     model.SourceTypeRSS,
			Status:         model.FetchStatusSuccess,
			StartedAt:      now,
			CompletedAt:    now.Add(5 * time.Second),
			ProcessingTime: 5 * time.Second,
			URLs:           []string{"https://example.com/rss"},
			ItemsFetched:   10,
			IoCsExtracted:  20,
			IoCsCreated:    15,
			IoCsUpdated:    3,
			IoCsUnchanged:  2,
			ErrorCount:     0,
			Errors:         []*model.FetchError{},
			CreatedAt:      now,
		}

		// Save history
		gt.NoError(t, repo.SaveHistory(ctx, history))

		// Retrieve history
		retrieved, err := repo.GetHistory(ctx, sourceID, historyID)
		gt.NoError(t, err)

		// Verify all fields
		gt.V(t, retrieved.ID).Equal(history.ID).Describe("history ID")
		gt.V(t, retrieved.SourceID).Equal(history.SourceID).Describe("source ID")
		gt.V(t, retrieved.SourceType).Equal(history.SourceType).Describe("source type")
		gt.V(t, retrieved.Status).Equal(history.Status).Describe("status")
		gt.N(t, retrieved.ItemsFetched).Equal(history.ItemsFetched).Describe("items fetched")
		gt.N(t, retrieved.IoCsExtracted).Equal(history.IoCsExtracted).Describe("iocs extracted")
		gt.N(t, retrieved.IoCsCreated).Equal(history.IoCsCreated).Describe("iocs created")
		gt.N(t, retrieved.IoCsUpdated).Equal(history.IoCsUpdated).Describe("iocs updated")
		gt.N(t, retrieved.IoCsUnchanged).Equal(history.IoCsUnchanged).Describe("iocs unchanged")
		gt.N(t, retrieved.ErrorCount).Equal(history.ErrorCount).Describe("error count")

		// Verify timestamps (with tolerance for storage precision)
		gt.True(t, retrieved.StartedAt.Sub(history.StartedAt).Abs() < time.Second)
		gt.True(t, retrieved.CompletedAt.Sub(history.CompletedAt).Abs() < time.Second)
		gt.True(t, retrieved.CreatedAt.Sub(history.CreatedAt).Abs() < time.Second)
		gt.True(t, retrieved.ProcessingTime >= 4*time.Second && retrieved.ProcessingTime <= 6*time.Second)
	})

	t.Run("save history with errors", func(t *testing.T) {
		now := time.Now()
		sourceID := now.Format("source-20060102-150405.000000")
		historyID := model.GenerateHistoryID()

		// Create history with errors
		history := &model.History{
			ID:             historyID,
			SourceID:       sourceID,
			SourceType:     model.SourceTypeFeed,
			Status:         model.FetchStatusPartialSuccess,
			StartedAt:      now,
			CompletedAt:    now.Add(10 * time.Second),
			ProcessingTime: 10 * time.Second,
			URLs:           []string{"https://example.com/feed"},
			ItemsFetched:   5,
			IoCsExtracted:  8,
			IoCsCreated:    5,
			IoCsUpdated:    2,
			IoCsUnchanged:  1,
			ErrorCount:     2,
			Errors: []*model.FetchError{
				{
					Message: "failed to fetch item 1",
					Values: map[string]string{
						"item_id": "123",
						"url":     "https://example.com/feed",
					},
				},
				{
					Message: "failed to parse item 2",
					Values: map[string]string{
						"item_id": "456",
						"error":   "invalid format",
					},
				},
			},
			CreatedAt: now,
		}

		// Save history
		gt.NoError(t, repo.SaveHistory(ctx, history))

		// Retrieve and verify
		retrieved, err := repo.GetHistory(ctx, sourceID, historyID)
		gt.NoError(t, err)

		gt.V(t, retrieved.Status).Equal(model.FetchStatusPartialSuccess).Describe("status should be partial success")
		gt.N(t, retrieved.ErrorCount).Equal(2).Describe("error count")
		gt.A(t, retrieved.Errors).Length(2).Describe("errors list length")

		gt.A(t, retrieved.Errors).At(0, func(t testing.TB, err *model.FetchError) {
			gt.S(t, err.Message).Equal("failed to fetch item 1").Describe("first error message")
			gt.V(t, err.Values["item_id"]).Equal("123").Describe("first error item_id")
			gt.V(t, err.Values["url"]).Equal("https://example.com/feed").Describe("first error url")
		})

		gt.A(t, retrieved.Errors).At(1, func(t testing.TB, err *model.FetchError) {
			gt.S(t, err.Message).Equal("failed to parse item 2").Describe("second error message")
			gt.V(t, err.Values["item_id"]).Equal("456").Describe("second error item_id")
			gt.V(t, err.Values["error"]).Equal("invalid format").Describe("second error context")
		})
	})

	t.Run("list histories by source", func(t *testing.T) {
		now := time.Now()
		sourceID := now.Format("source-20060102-150405.000000")

		// Create multiple histories
		histories := []*model.History{
			{
				ID:             model.GenerateHistoryID(),
				SourceID:       sourceID,
				SourceType:     model.SourceTypeRSS,
				Status:         model.FetchStatusSuccess,
				StartedAt:      now.Add(-3 * time.Hour),
				CompletedAt:    now.Add(-3 * time.Hour).Add(5 * time.Second),
				ProcessingTime: 5 * time.Second,
				URLs:           []string{"https://example.com/rss1"},
				ItemsFetched:   5,
				IoCsExtracted:  10,
				IoCsCreated:    10,
				IoCsUpdated:    0,
				IoCsUnchanged:  0,
				ErrorCount:     0,
				Errors:         []*model.FetchError{},
				CreatedAt:      now.Add(-3 * time.Hour),
			},
			{
				ID:             model.GenerateHistoryID(),
				SourceID:       sourceID,
				SourceType:     model.SourceTypeRSS,
				Status:         model.FetchStatusSuccess,
				StartedAt:      now.Add(-2 * time.Hour),
				CompletedAt:    now.Add(-2 * time.Hour).Add(5 * time.Second),
				ProcessingTime: 5 * time.Second,
				URLs:           []string{"https://example.com/rss2"},
				ItemsFetched:   3,
				IoCsExtracted:  6,
				IoCsCreated:    5,
				IoCsUpdated:    1,
				IoCsUnchanged:  0,
				ErrorCount:     0,
				Errors:         []*model.FetchError{},
				CreatedAt:      now.Add(-2 * time.Hour),
			},
			{
				ID:             model.GenerateHistoryID(),
				SourceID:       sourceID,
				SourceType:     model.SourceTypeRSS,
				Status:         model.FetchStatusSuccess,
				StartedAt:      now.Add(-1 * time.Hour),
				CompletedAt:    now.Add(-1 * time.Hour).Add(5 * time.Second),
				ProcessingTime: 5 * time.Second,
				URLs:           []string{"https://example.com/rss3"},
				ItemsFetched:   2,
				IoCsExtracted:  4,
				IoCsCreated:    2,
				IoCsUpdated:    2,
				IoCsUnchanged:  0,
				ErrorCount:     0,
				Errors:         []*model.FetchError{},
				CreatedAt:      now.Add(-1 * time.Hour),
			},
		}

		// Save all histories
		for _, h := range histories {
			gt.NoError(t, repo.SaveHistory(ctx, h))
		}

		// List all histories (no limit)
		retrieved, total, err := repo.ListHistoriesBySource(ctx, sourceID, 0, 0)
		gt.NoError(t, err)
		gt.A(t, retrieved).Length(3).Describe("should retrieve all 3 histories")
		gt.N(t, total).Equal(3).Describe("total should be 3")

		// Verify order (newest first)
		gt.True(t, retrieved[0].StartedAt.After(retrieved[1].StartedAt))
		gt.True(t, retrieved[1].StartedAt.After(retrieved[2].StartedAt))

		// Test pagination - get first 2
		page1, total1, err := repo.ListHistoriesBySource(ctx, sourceID, 2, 0)
		gt.NoError(t, err)
		gt.A(t, page1).Length(2).Describe("first page should have 2 items")
		gt.N(t, total1).Equal(3).Describe("total should still be 3")

		// Test pagination - get next 2 (only 1 remains)
		page2, total2, err := repo.ListHistoriesBySource(ctx, sourceID, 2, 2)
		gt.NoError(t, err)
		gt.A(t, page2).Length(1).Describe("second page should have 1 item")
		gt.N(t, total2).Equal(3).Describe("total should still be 3")

		// Test pagination - offset beyond available
		page3, total3, err := repo.ListHistoriesBySource(ctx, sourceID, 2, 10)
		gt.NoError(t, err)
		gt.A(t, page3).Length(0).Describe("page beyond available should be empty")
		gt.N(t, total3).Equal(3).Describe("total should still be 3")
	})

	t.Run("get non-existent history returns error", func(t *testing.T) {
		now := time.Now()
		sourceID := now.Format("nonexistent-20060102-150405.000000")
		historyID := model.GenerateHistoryID()

		_, err := repo.GetHistory(ctx, sourceID, historyID)
		gt.Error(t, err)
	})

	t.Run("list histories for non-existent source returns empty", func(t *testing.T) {
		now := time.Now()
		sourceID := now.Format("nonexistent-20060102-150405.000000")

		histories, total, err := repo.ListHistoriesBySource(ctx, sourceID, 10, 0)
		gt.NoError(t, err)
		gt.A(t, histories).Length(0).Describe("should return empty list for non-existent source")
		gt.N(t, total).Equal(0).Describe("total should be 0 for non-existent source")
	})

	t.Run("empty source ID returns error", func(t *testing.T) {
		now := time.Now()
		history := &model.History{
			ID:             model.GenerateHistoryID(),
			SourceID:       "",
			SourceType:     model.SourceTypeRSS,
			Status:         model.FetchStatusSuccess,
			StartedAt:      now,
			CompletedAt:    now.Add(5 * time.Second),
			ProcessingTime: 5 * time.Second,
			URLs:           []string{},
			CreatedAt:      now,
		}

		err := repo.SaveHistory(ctx, history)
		gt.Error(t, err)
	})

	t.Run("empty history ID returns error", func(t *testing.T) {
		now := time.Now()
		sourceID := now.Format("source-20060102-150405.000000")
		history := &model.History{
			ID:             "",
			SourceID:       sourceID,
			SourceType:     model.SourceTypeRSS,
			Status:         model.FetchStatusSuccess,
			StartedAt:      now,
			CompletedAt:    now.Add(5 * time.Second),
			ProcessingTime: 5 * time.Second,
			URLs:           []string{},
			CreatedAt:      now,
		}

		err := repo.SaveHistory(ctx, history)
		gt.Error(t, err)
	})
}

func TestHistoryRepository_Memory(t *testing.T) {
	repo := memory.New()
	runHistoryRepositoryTest(t, repo)
}

func TestHistoryRepository_Firestore(t *testing.T) {
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

	runHistoryRepositoryTest(t, repo)
}
