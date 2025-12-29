package extractor_test

import (
	"context"
	"math"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/extractor"
	"github.com/secmon-lab/beehive/pkg/domain/vectorizer"
)

// cosineSimilarity calculates cosine similarity between two vectors
// This is a test helper function
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func TestExtractorWithNGramVectorizer(t *testing.T) {
	ctx := context.Background()

	t.Run("use n-gram vectorizer for embedding", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		// Generate embedding for a domain
		embedding, err := e.GenerateEmbedding(ctx, "example.com")
		gt.NoError(t, err)
		gt.A(t, embedding).Length(128).Describe("should generate 128-dimension embedding")

		// Verify it's not a zero vector
		hasNonZero := false
		for _, val := range embedding {
			if val != 0 {
				hasNonZero = true
				break
			}
		}
		gt.True(t, hasNonZero).Describe("embedding should not be all zeros")
	})

	t.Run("generate embedding for query", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		// Generate embedding for a search query
		embedding, err := e.GenerateEmbedding(ctx, "malware")
		gt.NoError(t, err)
		gt.A(t, embedding).Length(128).Describe("should generate 128-dimension embedding for query")
	})

	t.Run("generate embedding for URL", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		embedding, err := e.GenerateEmbedding(ctx, "https://malware.com/payload.exe")
		gt.NoError(t, err)
		gt.A(t, embedding).Length(128).Describe("should generate embedding for URL")
	})

	t.Run("generate embedding for IP address", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		embedding, err := e.GenerateEmbedding(ctx, "192.168.1.1")
		gt.NoError(t, err)
		gt.A(t, embedding).Length(128).Describe("should generate embedding for IP")
	})
}

func TestExtractorVectorizerIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("same domain produces similar embeddings", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		emb1, err1 := e.GenerateEmbedding(ctx, "evil.com")
		emb2, err2 := e.GenerateEmbedding(ctx, "evil.net")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		// Calculate cosine similarity
		similarity := cosineSimilarity(emb1, emb2)
		gt.True(t, similarity >= 0.5).Describef("evil.com and evil.net should be similar (%.4f)", similarity)
	})

	t.Run("query matches relevant IoC", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		e := extractor.New(nil, extractor.WithNGramVectorizer(v))

		queryEmb, err1 := e.GenerateEmbedding(ctx, "malware")
		iocEmb, err2 := e.GenerateEmbedding(ctx, "malware-download.com")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(queryEmb, iocEmb)
		gt.True(t, similarity > 0.4).Describef("malware query should match malware domain (%.4f)", similarity)
	})
}
