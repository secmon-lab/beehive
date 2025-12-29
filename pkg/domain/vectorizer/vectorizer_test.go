package vectorizer_test

import (
	"math"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/vectorizer"
)

// cosineSimilarity calculates cosine similarity between two vectors
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

func TestExtractNGrams(t *testing.T) {
	t.Run("basic 3-gram extraction", func(t *testing.T) {
		ngrams := vectorizer.ExtractNGrams("hello", 3)
		gt.V(t, ngrams["hel"]).Equal(1).Describe("should extract 'hel'")
		gt.V(t, ngrams["ell"]).Equal(1).Describe("should extract 'ell'")
		gt.V(t, ngrams["llo"]).Equal(1).Describe("should extract 'llo'")
		gt.V(t, len(ngrams)).Equal(3).Describe("should have 3 unique n-grams")
	})

	t.Run("unicode string (Japanese)", func(t *testing.T) {
		ngrams := vectorizer.ExtractNGrams("こんにちは", 3)
		gt.V(t, ngrams["こんに"]).Equal(1).Describe("should extract Japanese 3-gram")
		gt.V(t, ngrams["んにち"]).Equal(1).Describe("should extract Japanese 3-gram")
		gt.V(t, ngrams["にちは"]).Equal(1).Describe("should extract Japanese 3-gram")
	})

	t.Run("empty string", func(t *testing.T) {
		ngrams := vectorizer.ExtractNGrams("", 3)
		gt.V(t, len(ngrams)).Equal(0).Describe("should return empty map")
	})

	t.Run("string shorter than n", func(t *testing.T) {
		ngrams := vectorizer.ExtractNGrams("ab", 3)
		gt.V(t, len(ngrams)).Equal(0).Describe("should return empty map for string shorter than n")
	})

	t.Run("repeated characters", func(t *testing.T) {
		ngrams := vectorizer.ExtractNGrams("aaa", 3)
		gt.V(t, ngrams["aaa"]).Equal(1).Describe("should extract 'aaa' once")
		gt.V(t, len(ngrams)).Equal(1).Describe("should have 1 unique n-gram")
	})
}

func TestHashToIndex(t *testing.T) {
	t.Run("dimension range", func(t *testing.T) {
		dim := 128
		for _, ngram := range []string{"abc", "xyz", "123", "hello", "world"} {
			idx := vectorizer.HashToIndex(ngram, dim)
			gt.True(t, idx >= 0 && idx < dim).Describe("index should be within dimension range")
		}
	})

	t.Run("same ngram produces same index", func(t *testing.T) {
		dim := 128
		idx1 := vectorizer.HashToIndex("test", dim)
		idx2 := vectorizer.HashToIndex("test", dim)
		gt.V(t, idx1).Equal(idx2).Describe("same n-gram should produce same index")
	})

	t.Run("different ngrams", func(t *testing.T) {
		dim := 128
		idx1 := vectorizer.HashToIndex("abc", dim)
		idx2 := vectorizer.HashToIndex("xyz", dim)
		// Note: collision is possible but unlikely for these specific strings
		gt.V(t, idx1).NotEqual(idx2).Describe("different n-grams typically produce different indices")
	})
}

func TestNormalizeL2(t *testing.T) {
	t.Run("vector norm becomes 1.0", func(t *testing.T) {
		vec := []float32{3, 4}
		normalized := vectorizer.NormalizeL2(vec)

		// Calculate norm
		var sumSquares float64
		for _, v := range normalized {
			sumSquares += float64(v) * float64(v)
		}
		norm := math.Sqrt(sumSquares)

		gt.True(t, math.Abs(norm-1.0) < 1e-6).Describe("normalized vector should have norm ~1.0")
	})

	t.Run("zero vector", func(t *testing.T) {
		vec := []float32{0, 0, 0}
		normalized := vectorizer.NormalizeL2(vec)
		gt.V(t, normalized).Equal(vec).Describe("zero vector should remain unchanged")
	})

	t.Run("negative values", func(t *testing.T) {
		vec := []float32{-3, 4}
		normalized := vectorizer.NormalizeL2(vec)

		var sumSquares float64
		for _, v := range normalized {
			sumSquares += float64(v) * float64(v)
		}
		norm := math.Sqrt(sumSquares)

		gt.True(t, math.Abs(norm-1.0) < 1e-6).Describe("normalized vector with negative values should have norm ~1.0")
	})
}

func TestVectorize(t *testing.T) {
	v := vectorizer.NewNGramVectorizer()

	t.Run("basic vectorization", func(t *testing.T) {
		vec, err := v.Vectorize("example.com")
		gt.NoError(t, err)
		gt.A(t, vec).Length(128).Describe("should produce 128-dimension vector")
	})

	t.Run("empty value", func(t *testing.T) {
		_, err := v.Vectorize("")
		gt.Error(t, err)
	})

	t.Run("all IoC types", func(t *testing.T) {
		testCases := []string{
			"https://example.com/path",
			"example.com",
			"192.168.1.1",
			"d41d8cd98f00b204e9800998ecf8427e",
			"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"user@example.com",
		}

		for _, value := range testCases {
			vec, err := v.Vectorize(value)
			gt.NoError(t, err)
			gt.A(t, vec).Length(128).Describef("should produce 128-dimension vector for %s", value)
		}
	})

	t.Run("query without type", func(t *testing.T) {
		vec, err := v.Vectorize("malware")
		gt.NoError(t, err)
		gt.A(t, vec).Length(128).Describe("should vectorize generic query")
	})
}

func TestSimilarity(t *testing.T) {
	v := vectorizer.NewNGramVectorizer()

	t.Run("TLD different, domain same", func(t *testing.T) {
		vec1, err1 := v.Vectorize("evil.com")
		vec2, err2 := v.Vectorize("evil.net")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// evil.com vs evil.net: only 4 chars different out of ~8 chars
		// Expect moderate similarity (>= 0.5 due to shared "evil." prefix)
		gt.True(t, similarity >= 0.5).Describef("evil.com and evil.net should be similar (%.4f)", similarity)
	})

	t.Run("typo tolerance", func(t *testing.T) {
		vec1, err1 := v.Vectorize("google.com")
		vec2, err2 := v.Vectorize("gogle.com")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// google.com vs gogle.com: 1 char difference
		// Expect high similarity
		gt.True(t, similarity > 0.8).Describef("google.com and gogle.com should be very similar (%.4f)", similarity)
	})

	t.Run("partial substring match", func(t *testing.T) {
		vec1, err1 := v.Vectorize("malware")
		vec2, err2 := v.Vectorize("malware-domain.com")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		gt.True(t, similarity > 0.4).Describef("malware and malware-domain.com should have moderate similarity (%.4f)", similarity)
	})

	t.Run("URL domain match", func(t *testing.T) {
		vec1, err1 := v.Vectorize("example")
		vec2, err2 := v.Vectorize("https://example.com/path")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		gt.True(t, similarity > 0.5).Describef("example and https://example.com/path should be similar (%.4f)", similarity)
	})

	t.Run("completely different strings", func(t *testing.T) {
		vec1, err1 := v.Vectorize("google.com")
		vec2, err2 := v.Vectorize("microsoft.com")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// Both contain ".com" so some overlap expected
		// Still should be less similar than intentionally similar pairs
		gt.True(t, similarity < 0.5).Describef("google.com and microsoft.com should be dissimilar (%.4f)", similarity)
	})

	t.Run("search scenario: phishing query", func(t *testing.T) {
		queryVec, err := v.Vectorize("phishing")
		gt.NoError(t, err)

		// Candidate IoCs
		candidates := []string{
			"phishing-site.com",
			"anti-phishing.org",
			"example.com",
			"malware.com",
		}

		var similarityScores []struct {
			value      string
			similarity float64
		}

		for _, value := range candidates {
			vec, err := v.Vectorize(value)
			gt.NoError(t, err)
			sim := cosineSimilarity(queryVec, vec)
			similarityScores = append(similarityScores, struct {
				value      string
				similarity float64
			}{value, sim})
		}

		// phishing-site.com and anti-phishing.org should have higher similarity than others
		phishingSite := similarityScores[0].similarity
		antiPhishing := similarityScores[1].similarity
		exampleCom := similarityScores[2].similarity
		malwareCom := similarityScores[3].similarity

		gt.True(t, phishingSite > exampleCom).Describef("phishing-site.com (%.4f) should be more similar than example.com (%.4f)", phishingSite, exampleCom)
		gt.True(t, antiPhishing > exampleCom).Describef("anti-phishing.org (%.4f) should be more similar than example.com (%.4f)", antiPhishing, exampleCom)
		gt.True(t, phishingSite > malwareCom).Describef("phishing-site.com (%.4f) should be more similar than malware.com (%.4f)", phishingSite, malwareCom)
	})

	t.Run("search scenario: IP prefix query", func(t *testing.T) {
		queryVec, err := v.Vectorize("192.168")
		gt.NoError(t, err)

		// Candidate IoCs
		candidates := []string{
			"192.168.1.1",
			"192.168.0.254",
			"10.0.0.1",
		}

		var similarityScores []struct {
			value      string
			similarity float64
		}

		for _, value := range candidates {
			vec, err := v.Vectorize(value)
			gt.NoError(t, err)
			sim := cosineSimilarity(queryVec, vec)
			similarityScores = append(similarityScores, struct {
				value      string
				similarity float64
			}{value, sim})
		}

		// 192.168.x.x should have higher similarity than 10.0.0.1
		match1 := similarityScores[0].similarity
		match2 := similarityScores[1].similarity
		noMatch := similarityScores[2].similarity

		gt.True(t, match1 > noMatch).Describef("192.168.1.1 (%.4f) should be more similar than 10.0.0.1 (%.4f)", match1, noMatch)
		gt.True(t, match2 > noMatch).Describef("192.168.0.254 (%.4f) should be more similar than 10.0.0.1 (%.4f)", match2, noMatch)
	})
}

func TestVectorizeBatch(t *testing.T) {
	v := vectorizer.NewNGramVectorizer()

	t.Run("batch vectorization", func(t *testing.T) {
		texts := []string{"example.com", "google.com", "192.168.1.1"}

		vecs, err := v.VectorizeBatch(texts)
		gt.NoError(t, err)
		gt.A(t, vecs).Length(3).Describe("should produce 3 vectors")

		for i, vec := range vecs {
			gt.A(t, vec).Length(128).Describef("vector %d should have 128 dimensions", i)
		}
	})
}

func TestDimension(t *testing.T) {
	t.Run("default dimension", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer()
		gt.V(t, v.Dimension()).Equal(128).Describe("default dimension should be 128")
	})

	t.Run("custom dimension", func(t *testing.T) {
		v := vectorizer.NewNGramVectorizer(vectorizer.WithDimension(256))
		gt.V(t, v.Dimension()).Equal(256).Describe("custom dimension should be 256")
	})
}

func TestCustomNGramSize(t *testing.T) {
	v := vectorizer.NewNGramVectorizer(vectorizer.WithNGramSize(4))

	vec, err := v.Vectorize("example.com")
	gt.NoError(t, err)
	gt.A(t, vec).Length(128).Describe("should produce 128-dimension vector with 4-gram")
}

func TestRealDataSimilarity(t *testing.T) {
	v := vectorizer.NewNGramVectorizer()

	t.Run("real URLs with same domain different paths", func(t *testing.T) {
		url1 := "https://malware-download.com/payload.exe"
		url2 := "https://malware-download.com/trojan.bin"

		vec1, err1 := v.Vectorize(url1)
		vec2, err2 := v.Vectorize(url2)
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// URLs with same domain have high similarity (domain + different paths)
		gt.True(t, similarity > 0.6).Describef("same domain URLs should be similar (%.4f)", similarity)
	})

	t.Run("real domains with TLD difference", func(t *testing.T) {
		vec1, err1 := v.Vectorize("malware.com")
		vec2, err2 := v.Vectorize("malware.net")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		gt.True(t, similarity >= 0.5).Describef("malware.com and malware.net should be similar (%.4f)", similarity)
	})

	t.Run("real IP addresses in same subnet", func(t *testing.T) {
		vec1, err1 := v.Vectorize("192.168.1.1")
		vec2, err2 := v.Vectorize("192.168.0.254")
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// 192.168 prefix match gives moderate similarity
		gt.True(t, similarity > 0.6).Describef("IPs in 192.168.x.x should be similar (%.4f)", similarity)
	})

	t.Run("real hashes - completely different", func(t *testing.T) {
		hash1 := "d41d8cd98f00b204e9800998ecf8427e"
		hash2 := "5d41402abc4b2a76b9719d911017c592"

		vec1, err1 := v.Vectorize(hash1)
		vec2, err2 := v.Vectorize(hash2)
		gt.NoError(t, err1)
		gt.NoError(t, err2)

		similarity := cosineSimilarity(vec1, vec2)
		// Hashes are random-looking, should have low similarity
		gt.True(t, similarity < 0.6).Describef("different hashes should be dissimilar (%.4f)", similarity)
	})

	t.Run("query malware matches malware domains", func(t *testing.T) {
		queryVec, err := v.Vectorize("malware")
		gt.NoError(t, err)

		// Test against multiple domains
		domains := []string{
			"malware.com",
			"malware-download.com",
			"phishing-site.com",
			"example.com",
		}

		var similarities []float64
		for _, domain := range domains {
			vec, err := v.Vectorize(domain)
			gt.NoError(t, err)
			sim := cosineSimilarity(queryVec, vec)
			similarities = append(similarities, sim)
		}

		// malware.com should be most similar
		// malware-download.com should be second
		// phishing-site.com and example.com should be less similar
		gt.True(t, similarities[0] > similarities[2]).Describef("malware.com (%.4f) > phishing-site.com (%.4f)", similarities[0], similarities[2])
		gt.True(t, similarities[1] > similarities[3]).Describef("malware-download.com (%.4f) > example.com (%.4f)", similarities[1], similarities[3])
	})
}
