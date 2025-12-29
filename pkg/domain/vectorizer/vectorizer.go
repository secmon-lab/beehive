package vectorizer

import (
	"hash/fnv"
	"math"
	"strings"

	"github.com/m-mizutani/goerr/v2"
)

const (
	// DefaultDimension is the default vector dimension (Firestore constraint)
	DefaultDimension = 128
)

var (
	// ErrVectorizationFailed is returned when vectorization fails
	ErrVectorizationFailed = goerr.New("vectorization failed")
)

// Vectorizer generates fixed-dimension vectors from text (IoC values or queries)
type Vectorizer interface {
	// Vectorize converts any text (IoC value or query) to a fixed-dimension vector
	Vectorize(value string) ([]float32, error)

	// VectorizeBatch vectorizes multiple texts efficiently
	VectorizeBatch(texts []string) ([][]float32, error)

	// Dimension returns the output vector dimension
	Dimension() int
}

// NGramVectorizer implements Vectorizer using unified character n-gram approach
type NGramVectorizer struct {
	dimension int
	ngramSize int
}

// Option configures NGramVectorizer
type Option func(*NGramVectorizer)

// WithDimension sets the output vector dimension (default: 128)
func WithDimension(dim int) Option {
	return func(v *NGramVectorizer) {
		v.dimension = dim
	}
}

// WithNGramSize sets the n-gram size (default: 3)
func WithNGramSize(size int) Option {
	return func(v *NGramVectorizer) {
		v.ngramSize = size
	}
}

// NewNGramVectorizer creates a new n-gram vectorizer
func NewNGramVectorizer(opts ...Option) *NGramVectorizer {
	v := &NGramVectorizer{
		dimension: DefaultDimension, // Default: 128
		ngramSize: 3,                // Default: 3-gram
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// Vectorize converts any text (IoC value or query) to a fixed-dimension vector
func (v *NGramVectorizer) Vectorize(value string) ([]float32, error) {
	if value == "" {
		return nil, goerr.Wrap(ErrVectorizationFailed, "empty value")
	}

	// Simple normalization: trim and lowercase only
	// No type-specific processing to keep query and IoC in same vector space
	normalized := strings.ToLower(strings.TrimSpace(value))

	// Extract n-grams
	ngrams := extractNGrams(normalized, v.ngramSize)

	// Vectorize using hash trick
	vec := vectorize(ngrams, v.dimension)

	// L2 normalize
	vec = normalizeL2(vec)

	return vec, nil
}

// VectorizeBatch vectorizes multiple texts efficiently
func (v *NGramVectorizer) VectorizeBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := v.Vectorize(text)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to vectorize batch item",
				goerr.V("index", i),
				goerr.V("text", text))
		}
		results[i] = vec
	}

	return results, nil
}

// Dimension returns the output vector dimension
func (v *NGramVectorizer) Dimension() int {
	return v.dimension
}

// extractNGrams extracts character n-grams from text
// Returns map[ngram]frequency
func extractNGrams(text string, n int) map[string]int {
	ngrams := make(map[string]int)

	// Convert to runes for proper Unicode handling
	runes := []rune(text)

	// Extract n-grams
	for i := 0; i <= len(runes)-n; i++ {
		ngram := string(runes[i : i+n])
		ngrams[ngram]++
	}

	return ngrams
}

// hashToIndex maps n-gram to vector dimension using FNV-1a hash
func hashToIndex(ngram string, dim int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(ngram)) // hash.Hash.Write never returns an error
	// #nosec G115 - dimension is fixed at 128, conversion is safe
	hashValue := h.Sum32() % uint32(dim)
	return int(hashValue)
}

// vectorize converts n-gram frequency map to fixed-dimension vector
func vectorize(ngrams map[string]int, dim int) []float32 {
	vec := make([]float32, dim)

	// Hash trick: map each n-gram to a dimension and add its frequency
	for ngram, freq := range ngrams {
		idx := hashToIndex(ngram, dim)
		vec[idx] += float32(freq)
	}

	return vec
}

// normalizeL2 applies L2 normalization to vector
func normalizeL2(vec []float32) []float32 {
	// Calculate L2 norm
	var sumSquares float64
	for _, v := range vec {
		sumSquares += float64(v * v)
	}

	norm := math.Sqrt(sumSquares)

	// Avoid division by zero
	if norm < 1e-10 {
		return vec
	}

	// Normalize
	normalized := make([]float32, len(vec))
	for i, v := range vec {
		normalized[i] = float32(float64(v) / norm)
	}

	return normalized
}
