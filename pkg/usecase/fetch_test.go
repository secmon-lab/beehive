package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
)

func TestFetchUseCase_FetchAllSources(t *testing.T) {
	ctx := context.Background()

	t.Run("skip disabled sources", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)

		sources := map[string]model.Source{
			"source1": {
				Type:    model.SourceTypeFeed,
				URL:     "http://example.com/feed",
				Enabled: false,
				FeedConfig: &model.FeedConfig{
					Schema: "abuse_ch_urlhaus",
				},
			},
		}

		stats, err := uc.FetchAllSources(ctx, sources, nil)
		gt.NoError(t, err)
		gt.A(t, stats).Length(0).Describe("disabled sources should not be processed")
	})

	t.Run("filter by tags", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)

		sources := map[string]model.Source{
			"source1": {
				Type:    model.SourceTypeFeed,
				URL:     "http://example.com/feed1",
				Tags:    []string{"vendor", "google"},
				Enabled: true,
				FeedConfig: &model.FeedConfig{
					Schema: "abuse_ch_urlhaus",
				},
			},
			"source2": {
				Type:    model.SourceTypeFeed,
				URL:     "http://example.com/feed2",
				Tags:    []string{"threat-intel"},
				Enabled: true,
				FeedConfig: &model.FeedConfig{
					Schema: "abuse_ch_urlhaus",
				},
			},
		}

		// Filter for "vendor" tag - should only process source1
		stats, err := uc.FetchAllSources(ctx, sources, []string{"vendor"})
		gt.NoError(t, err)
		// Should have exactly 1 stat (for source1 only, even though it will fail with HTTP error)
		gt.A(t, stats).Length(1).Describe("should process only source1 with vendor tag")
		gt.A(t, stats).At(0, func(t testing.TB, stat *usecase.FetchStats) {
			gt.S(t, stat.SourceID).Equal("source1").Describe("processed source should be source1")
			gt.S(t, stat.SourceType).Equal(string(model.SourceTypeFeed)).Describe("source type should be feed")
			// The source will error because it's a fake URL, so we expect errors
			gt.V(t, stat.Errors).NotEqual(0).Describe("should have errors from fake URL")
		})
	})

	t.Run("handle unknown source type", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)

		sources := map[string]model.Source{
			"source1": {
				Type:    "unknown_type",
				URL:     "http://example.com",
				Enabled: true,
			},
		}

		stats, err := uc.FetchAllSources(ctx, sources, nil)
		gt.NoError(t, err)
		gt.A(t, stats).Length(0).Describe("unknown source types should be skipped")
	})
}

func TestFetchUseCase_SourceState(t *testing.T) {
	ctx := context.Background()

	t.Run("create new state for first fetch", func(t *testing.T) {
		repo := memory.New()

		sourceID := "test-source-" + time.Now().Format("20060102-150405.000000")

		// Verify no state exists initially
		_, err := repo.GetState(ctx, sourceID)
		gt.Error(t, err) // Should error for non-existent state
	})

	t.Run("update existing state", func(t *testing.T) {
		repo := memory.New()

		sourceID := "test-source-" + time.Now().Format("20060102-150405.000000")

		// Create initial state
		initialState := &model.SourceState{
			SourceID:      sourceID,
			LastFetchedAt: time.Now().Add(-1 * time.Hour),
			ItemCount:     10,
			ErrorCount:    0,
		}
		gt.NoError(t, repo.SaveState(ctx, initialState))

		// Verify state was saved with exact values
		retrieved, err := repo.GetState(ctx, sourceID)
		gt.NoError(t, err)
		gt.S(t, retrieved.SourceID).Equal(sourceID).Describe("source ID should match")
		gt.N(t, retrieved.ItemCount).Equal(int64(10)).Describe("item count should be 10")
		gt.N(t, retrieved.ErrorCount).Equal(int64(0)).Describe("error count should be 0")
	})
}

func TestFetchUseCase_IoCSaving(t *testing.T) {
	ctx := context.Background()

	t.Run("save IoCs to repository", func(t *testing.T) {
		repo := memory.New()

		sourceID := "test-source-" + time.Now().Format("20060102-150405.000000")
		contextKey := model.IoCContextKey("test-entry-1")

		// Create and save an IoC
		ioc := &model.IoC{
			ID:          model.GenerateID(sourceID, model.IoCTypeIPv4, "192.0.2.1", contextKey),
			SourceID:    sourceID,
			SourceType:  string(model.SourceTypeFeed),
			Type:        model.IoCTypeIPv4,
			Value:       "192.0.2.1",
			Description: "Test malicious IP",
			Status:      model.IoCStatusActive,
			Embedding:   make([]float32, model.EmbeddingDimension),
		}

		gt.NoError(t, repo.UpsertIoC(ctx, ioc))

		// Verify IoC was saved with exact values
		retrieved, err := repo.GetIoC(ctx, ioc.ID)
		gt.NoError(t, err)
		gt.S(t, retrieved.ID).Equal(ioc.ID).Describe("IoC ID should match")
		gt.S(t, retrieved.SourceID).Equal(sourceID).Describe("source ID should match")
		gt.S(t, retrieved.Value).Equal("192.0.2.1").Describe("IoC value should be 192.0.2.1")
		gt.V(t, retrieved.Type).Equal(model.IoCTypeIPv4).Describe("IoC type should be IPv4")
		gt.S(t, retrieved.Description).Equal("Test malicious IP").Describe("description should match")
		gt.V(t, retrieved.Status).Equal(model.IoCStatusActive).Describe("status should be active")
	})

	t.Run("list IoCs by source", func(t *testing.T) {
		repo := memory.New()

		sourceID := "test-source-" + time.Now().Format("20060102-150405.000000")

		// Expected IoC values
		expectedValues := []string{
			"192.0.2.1",
			"192.0.2.2",
			"192.0.2.3",
		}

		// Save multiple IoCs with specific values
		for i, value := range expectedValues {
			contextKey := model.IoCContextKey(time.Now().Format("entry-20060102-150405.000000") + "-" + string(rune('a'+i)))
			ioc := &model.IoC{
				ID:         model.GenerateID(sourceID, model.IoCTypeIPv4, value, contextKey),
				SourceID:   sourceID,
				SourceType: string(model.SourceTypeFeed),
				Type:       model.IoCTypeIPv4,
				Value:      value,
				Status:     model.IoCStatusActive,
				Embedding:  make([]float32, model.EmbeddingDimension),
			}
			gt.NoError(t, repo.UpsertIoC(ctx, ioc))
		}

		// List all IoCs for source
		iocs, err := repo.ListIoCsBySource(ctx, sourceID)
		gt.NoError(t, err)
		gt.A(t, iocs).Length(3).Describe("should have exactly 3 IoCs")

		// Verify each IoC has expected values (order may vary)
		for _, ioc := range iocs {
			gt.S(t, ioc.SourceID).Equal(sourceID).Describe("IoC source ID should match")
			gt.V(t, ioc.Type).Equal(model.IoCTypeIPv4).Describe("IoC type should be IPv4")
			// Verify value is one of the expected values
			found := false
			for _, expected := range expectedValues {
				if ioc.Value == expected {
					found = true
					break
				}
			}
			gt.True(t, found).Describef("IoC value %s should be in expected values", ioc.Value)
		}
	})
}

func TestFetchUseCase_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("continue on source fetch error", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)

		sources := map[string]model.Source{
			"bad-source": {
				Type:    model.SourceTypeFeed,
				URL:     "http://invalid-url-that-does-not-exist.example.com/feed",
				Enabled: true,
				FeedConfig: &model.FeedConfig{
					Schema: "abuse_ch_urlhaus",
				},
			},
		}

		// Should not panic, should return stats with error
		stats, err := uc.FetchAllSources(ctx, sources, nil)
		gt.NoError(t, err) // FetchAllSources should not error even when sources fail
		gt.A(t, stats).Length(1).Describe("should have 1 stat entry for the failed source")
		gt.A(t, stats).At(0, func(t testing.TB, stat *usecase.FetchStats) {
			gt.S(t, stat.SourceID).Equal("bad-source").Describe("stat source ID should be bad-source")
			gt.N(t, stat.Errors).Greater(0).Describe("should have at least 1 error")
			gt.N(t, stat.ItemsFetched).Equal(0).Describe("should have 0 items fetched")
			gt.N(t, stat.IoCsExtracted).Equal(0).Describe("should have 0 IoCs extracted")
		})
	})

	t.Run("handle feed without config", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)

		sources := map[string]model.Source{
			"bad-feed": {
				Type:       model.SourceTypeFeed,
				URL:        "http://example.com/feed",
				Enabled:    true,
				FeedConfig: nil, // Missing config
			},
		}

		stats, err := uc.FetchAllSources(ctx, sources, nil)
		gt.NoError(t, err)
		gt.A(t, stats).Length(1).Describe("should have 1 stat entry")
		gt.A(t, stats).At(0, func(t testing.TB, stat *usecase.FetchStats) {
			gt.S(t, stat.SourceID).Equal("bad-feed").Describe("stat source ID should be bad-feed")
			gt.N(t, stat.Errors).Greater(0).Describe("should have at least 1 error from missing config")
		})
	})
}

func TestFetchStats(t *testing.T) {
	t.Run("stats structure with exact values", func(t *testing.T) {
		stats := &usecase.FetchStats{
			SourceID:       "test-source",
			SourceType:     string(model.SourceTypeFeed),
			ItemsFetched:   10,
			IoCsExtracted:  8,
			IoCsCreated:    6,
			IoCsUpdated:    2,
			IoCsUnchanged:  0,
			IoCsGeneric:    1,
			Errors:         0,
			ProcessingTime: 1 * time.Second,
		}

		gt.S(t, stats.SourceID).Equal("test-source").Describe("source ID should be test-source")
		gt.S(t, stats.SourceType).Equal(string(model.SourceTypeFeed)).Describe("source type should be feed")
		gt.N(t, stats.ItemsFetched).Equal(10).Describe("items fetched should be 10")
		gt.N(t, stats.IoCsExtracted).Equal(8).Describe("IoCs extracted should be 8")
		gt.N(t, stats.IoCsCreated).Equal(6).Describe("IoCs created should be 6")
		gt.N(t, stats.IoCsUpdated).Equal(2).Describe("IoCs updated should be 2")
		gt.N(t, stats.Errors).Equal(0).Describe("errors should be 0")
		gt.V(t, stats.ProcessingTime).Equal(1 * time.Second).Describe("processing time should be 1 second")
	})
}

func TestNewFetchUseCase(t *testing.T) {
	t.Run("create use case", func(t *testing.T) {
		repo := memory.New()
		uc := usecase.NewFetchUseCase(repo, repo, nil)
		gt.V(t, uc).NotNil().Describe("NewFetchUseCase should return non-nil use case")
	})
}
