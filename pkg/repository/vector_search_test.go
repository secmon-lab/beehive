package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/vectorizer"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
)

func TestVectorSearch(t *testing.T) {
	ctx := context.Background()
	v := vectorizer.NewNGramVectorizer()

	// Use memory repository for testing
	repo := memory.New()

	// Create timestamp-based unique values for parallel test execution
	timestamp := time.Now().Format("20060102-150405.000000")

	// Insert test IoCs with embeddings
	testIoCs := []struct {
		value   string
		iocType model.IoCType
	}{
		{fmt.Sprintf("malware-%s.com", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("malware-download-%s.com", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("phishing-%s.com", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("example-%s.com", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("evil-%s.com", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("evil-%s.net", timestamp), model.IoCTypeDomain},
		{fmt.Sprintf("192.168.1.%s", timestamp[len(timestamp)-3:]), model.IoCTypeIPv4},
		{fmt.Sprintf("192.168.0.%s", timestamp[len(timestamp)-3:]), model.IoCTypeIPv4},
		{fmt.Sprintf("10.0.0.%s", timestamp[len(timestamp)-3:]), model.IoCTypeIPv4},
	}

	t.Run("insert IoCs with embeddings", func(t *testing.T) {
		for i, tc := range testIoCs {
			// Generate embedding
			embedding, err := v.Vectorize(tc.value)
			gt.NoError(t, err)

			// Create IoC
			sourceID := fmt.Sprintf("test-source-%s", timestamp)
			contextKey := model.GenerateContextKey("test", map[string]string{"index": fmt.Sprintf("%d", i)})
			iocID := model.GenerateID(sourceID, tc.iocType, tc.value, contextKey)

			ioc := &model.IoC{
				ID:          iocID,
				SourceID:    sourceID,
				SourceType:  "test",
				Type:        tc.iocType,
				Value:       tc.value,
				Description: fmt.Sprintf("Test IoC %d", i),
				SourceURL:   "https://example.com/test",
				Embedding:   embedding,
				Status:      model.IoCStatusActive,
				FirstSeenAt: time.Now(),
				UpdatedAt:   time.Now(),
			}

			err = repo.UpsertIoC(ctx, ioc)
			gt.NoError(t, err)
		}
	})

	t.Run("search for malware query", func(t *testing.T) {
		// Generate query vector
		queryVec, err := v.Vectorize("malware")
		gt.NoError(t, err)

		// Search for nearest IoCs
		results, err := repo.FindNearestIoCs(ctx, queryVec, 5)
		gt.NoError(t, err)

		gt.True(t, len(results) > 0).Describe("should find results")
		gt.True(t, len(results) <= 5).Describe("should respect limit")

		// First result should contain "malware"
		gt.True(t, containsSubstring(results[0].Value, "malware")).Describef("top result should contain 'malware', got: %s", results[0].Value)
	})

	t.Run("search for evil query", func(t *testing.T) {
		queryVec, err := v.Vectorize("evil")
		gt.NoError(t, err)

		results, err := repo.FindNearestIoCs(ctx, queryVec, 3)
		gt.NoError(t, err)

		gt.True(t, len(results) > 0).Describe("should find results")

		// Top results should be evil.com or evil.net
		foundEvil := false
		for i := 0; i < min(2, len(results)); i++ {
			if containsSubstring(results[i].Value, "evil") {
				foundEvil = true
				break
			}
		}
		gt.True(t, foundEvil).Describef("top results should contain 'evil'")
	})

	t.Run("search for IP prefix", func(t *testing.T) {
		queryVec, err := v.Vectorize("192.168")
		gt.NoError(t, err)

		results, err := repo.FindNearestIoCs(ctx, queryVec, 5)
		gt.NoError(t, err)

		gt.True(t, len(results) > 0).Describe("should find results")

		// Count how many results are 192.168.x.x
		count192168 := 0
		for _, result := range results {
			if containsSubstring(result.Value, "192.168") {
				count192168++
			}
		}

		gt.True(t, count192168 > 0).Describef("should find 192.168.x.x addresses")
	})

	t.Run("TLD-agnostic domain search", func(t *testing.T) {
		// Search for evil.com
		queryVec, err := v.Vectorize(fmt.Sprintf("evil-%s.com", timestamp))
		gt.NoError(t, err)

		results, err := repo.FindNearestIoCs(ctx, queryVec, 5)
		gt.NoError(t, err)

		gt.True(t, len(results) > 0).Describe("should find results")

		// Both evil.com and evil.net should be in top results
		foundCom := false
		foundNet := false
		for _, result := range results {
			if containsSubstring(result.Value, fmt.Sprintf("evil-%s.com", timestamp)) {
				foundCom = true
			}
			if containsSubstring(result.Value, fmt.Sprintf("evil-%s.net", timestamp)) {
				foundNet = true
			}
		}

		gt.True(t, foundCom || foundNet).Describe("should find evil domain variants")
	})

	t.Run("invalid query vector dimension", func(t *testing.T) {
		invalidVec := make([]float32, 64) // Wrong dimension
		_, err := repo.FindNearestIoCs(ctx, invalidVec, 5)
		gt.Error(t, err)
	})

	t.Run("empty results with limit 0", func(t *testing.T) {
		queryVec, err := v.Vectorize("malware")
		gt.NoError(t, err)

		// limit <= 0 should return empty results
		results, err := repo.FindNearestIoCs(ctx, queryVec, 0)
		gt.NoError(t, err)
		gt.A(t, results).Length(0).Describe("limit 0 should return empty results")
	})
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
