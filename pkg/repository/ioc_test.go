package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	firestoreRepo "github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

func runIoCRepositoryTest(t *testing.T, repo interfaces.IoCRepository) {
	ctx := context.Background()

	t.Run("upsert and get IoC", func(t *testing.T) {
		// Use timestamp-based unique values to avoid conflicts in Firestore tests
		sourceID := time.Now().Format("source-20060102-150405.000000")
		value := time.Now().Format("192.168.1.150405000")
		contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000"))
		iocID := model.GenerateID(sourceID, model.IoCTypeIPv4, value, contextKey)

		// Create test IoC
		ioc := &model.IoC{
			ID:          iocID,
			SourceID:    sourceID,
			SourceType:  "feed",
			Type:        model.IoCTypeIPv4,
			Value:       value,
			Description: "Test malicious IP",
			SourceURL:   "https://example.com/feed",
			Context:     "Test context",
			Embedding:   make(firestore.Vector32, model.EmbeddingDimension),
			Status:      model.IoCStatusActive,
		}

		// Fill embedding with dummy data
		for i := 0; i < model.EmbeddingDimension; i++ {
			ioc.Embedding[i] = float32(i) / 1000.0
		}

		// Insert IoC
		gt.NoError(t, repo.UpsertIoC(ctx, ioc))

		// Retrieve IoC
		retrieved, err := repo.GetIoC(ctx, iocID)
		gt.NoError(t, err)

		// Verify all fields
		gt.Equal(t, retrieved.ID, ioc.ID)
		gt.Equal(t, retrieved.SourceID, ioc.SourceID)
		gt.Equal(t, retrieved.Type, ioc.Type)
		gt.Equal(t, retrieved.Value, ioc.Value)
		gt.Equal(t, retrieved.Status, ioc.Status)
		gt.False(t, retrieved.FirstSeenAt.IsZero())
		gt.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("update existing IoC preserves FirstSeenAt", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")
		value := time.Now().Format("evil-20060102-150405.000000.com")
		contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000"))
		iocID := model.GenerateID(sourceID, model.IoCTypeDomain, value, contextKey)

		// Create and insert initial IoC
		ioc := &model.IoC{
			ID:          iocID,
			SourceID:    sourceID,
			SourceType:  "rss",
			Type:        model.IoCTypeDomain,
			Value:       value,
			Description: "Initial description",
			Status:      model.IoCStatusActive,
			Embedding:   make(firestore.Vector32, model.EmbeddingDimension),
		}

		gt.NoError(t, repo.UpsertIoC(ctx, ioc))

		// Get initial state
		initial, err := repo.GetIoC(ctx, iocID)
		gt.NoError(t, err)

		// Wait a bit to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Update IoC
		updated := &model.IoC{
			ID:          iocID,
			SourceID:    sourceID,
			SourceType:  "rss",
			Type:        model.IoCTypeDomain,
			Value:       value,
			Description: "Updated description",
			Status:      model.IoCStatusInactive,
			Embedding:   make(firestore.Vector32, model.EmbeddingDimension),
		}

		gt.NoError(t, repo.UpsertIoC(ctx, updated))

		// Get updated state
		final, err := repo.GetIoC(ctx, iocID)
		gt.NoError(t, err)

		// Verify FirstSeenAt is preserved
		gt.True(t, final.FirstSeenAt.Equal(initial.FirstSeenAt))

		// Verify UpdatedAt changed
		gt.True(t, final.UpdatedAt.After(initial.UpdatedAt))

		// Verify description was updated
		gt.Equal(t, final.Description, "Updated description")

		// Verify status was updated
		gt.Equal(t, final.Status, model.IoCStatusInactive)
	})

	t.Run("list IoCs by source", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")

		// Insert multiple IoCs for the same source
		for i := 0; i < 3; i++ {
			// Create unique values for each iteration
			ts := time.Now().UnixNano()
			value := time.Now().Format("192.168.1.150405") + fmt.Sprintf("%d", ts%1000)
			contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000") + fmt.Sprintf("%d", i))
			iocID := model.GenerateID(sourceID, model.IoCTypeIPv4, value, contextKey)

			ioc := &model.IoC{
				ID:         iocID,
				SourceID:   sourceID,
				SourceType: "feed",
				Type:       model.IoCTypeIPv4,
				Value:      value,
				Status:     model.IoCStatusActive,
				Embedding:  make(firestore.Vector32, model.EmbeddingDimension),
			}

			gt.NoError(t, repo.UpsertIoC(ctx, ioc))
			// Small delay to ensure unique timestamps
			time.Sleep(1 * time.Millisecond)
		}

		// List IoCs by source
		iocs, err := repo.ListIoCsBySource(ctx, sourceID)
		gt.NoError(t, err)
		gt.Equal(t, len(iocs), 3)

		// Verify all belong to the same source
		for _, ioc := range iocs {
			gt.Equal(t, ioc.SourceID, sourceID)
		}
	})

	t.Run("get non-existent IoC returns error", func(t *testing.T) {
		_, err := repo.GetIoC(ctx, "non-existent-id")
		gt.Error(t, err)
	})

	t.Run("list empty source", func(t *testing.T) {
		iocs, err := repo.ListIoCsBySource(ctx, "non-existent-source")
		gt.NoError(t, err)
		gt.Equal(t, len(iocs), 0)
	})

	t.Run("validation errors", func(t *testing.T) {
		tests := []struct {
			name string
			ioc  *model.IoC
		}{
			{
				name: "empty source ID",
				ioc: &model.IoC{
					ID:        "test-id",
					SourceID:  "",
					Type:      model.IoCTypeIPv4,
					Value:     "192.168.1.1",
					Status:    model.IoCStatusActive,
					Embedding: make(firestore.Vector32, model.EmbeddingDimension),
				},
			},
			{
				name: "empty type",
				ioc: &model.IoC{
					ID:        "test-id",
					SourceID:  "test-source",
					Type:      "",
					Value:     "192.168.1.1",
					Status:    model.IoCStatusActive,
					Embedding: make(firestore.Vector32, model.EmbeddingDimension),
				},
			},
			{
				name: "empty value",
				ioc: &model.IoC{
					ID:        "test-id",
					SourceID:  "test-source",
					Type:      model.IoCTypeIPv4,
					Value:     "",
					Status:    model.IoCStatusActive,
					Embedding: make(firestore.Vector32, model.EmbeddingDimension),
				},
			},
			{
				name: "invalid status",
				ioc: &model.IoC{
					ID:        "test-id",
					SourceID:  "test-source",
					Type:      model.IoCTypeIPv4,
					Value:     "192.168.1.1",
					Status:    "invalid",
					Embedding: make(firestore.Vector32, model.EmbeddingDimension),
				},
			},
			{
				name: "invalid embedding dimension",
				ioc: &model.IoC{
					ID:        "test-id",
					SourceID:  "test-source",
					Type:      model.IoCTypeIPv4,
					Value:     "192.168.1.1",
					Status:    model.IoCStatusActive,
					Embedding: make(firestore.Vector32, 10), // Wrong dimension
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := repo.UpsertIoC(ctx, tt.ioc)
				gt.Error(t, err)
			})
		}
	})

	t.Run("different IoC types", func(t *testing.T) {
		sourceID := time.Now().Format("source-20060102-150405.000000")

		testCases := []struct {
			iocType model.IoCType
		}{
			{model.IoCTypeIPv4},
			{model.IoCTypeIPv6},
			{model.IoCTypeDomain},
			{model.IoCTypeURL},
			{model.IoCTypeEmail},
			{model.IoCTypeMD5},
			{model.IoCTypeSHA1},
			{model.IoCTypeSHA256},
		}

		for i, tc := range testCases {
			t.Run(string(tc.iocType), func(t *testing.T) {
				// Create unique value based on timestamp and IoC type
				var value string
				switch tc.iocType {
				case model.IoCTypeIPv4:
					value = time.Now().Format("192.168.1.150405")
				case model.IoCTypeIPv6:
					value = time.Now().Format("2001:0db8:85a3:0000:0000:8a2e:0370:150405")
				case model.IoCTypeDomain:
					value = time.Now().Format("evil-20060102-150405.000000.com")
				case model.IoCTypeURL:
					value = time.Now().Format("https://malicious-20060102-150405.000000.com/path")
				case model.IoCTypeEmail:
					value = time.Now().Format("attacker-20060102-150405.000000@evil.com")
				case model.IoCTypeMD5:
					value = time.Now().Format("5d41402abc4b2a76b9719d911017150405")
				case model.IoCTypeSHA1:
					value = time.Now().Format("aaf4c61ddcc5e8a2dabede0f3b482cd9aea9150405")
				case model.IoCTypeSHA256:
					value = time.Now().Format("2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266150405")
				}

				contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000") + string(rune('a'+i)))
				iocID := model.GenerateID(sourceID, tc.iocType, value, contextKey)
				ioc := &model.IoC{
					ID:         iocID,
					SourceID:   sourceID,
					SourceType: "test",
					Type:       tc.iocType,
					Value:      value,
					Status:     model.IoCStatusActive,
					Embedding:  make(firestore.Vector32, model.EmbeddingDimension),
				}

				gt.NoError(t, repo.UpsertIoC(ctx, ioc))

				retrieved, err := repo.GetIoC(ctx, iocID)
				gt.NoError(t, err)
				gt.Equal(t, retrieved.Type, tc.iocType)
			})
		}
	})
}

func TestIoCRepository_Memory(t *testing.T) {
	repo := memory.New()
	runIoCRepositoryTest(t, repo)
}

func TestIoCRepository_Firestore(t *testing.T) {
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

	runIoCRepositoryTest(t, repo)
}
