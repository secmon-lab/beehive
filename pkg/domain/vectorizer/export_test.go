package vectorizer

// Export internal functions for testing

// ExtractNGrams is exported for testing
func ExtractNGrams(text string, n int) map[string]int {
	return extractNGrams(text, n)
}

// HashToIndex is exported for testing
func HashToIndex(ngram string, dim int) int {
	return hashToIndex(ngram, dim)
}

// NormalizeL2 is exported for testing
func NormalizeL2(vec []float32) []float32 {
	return normalizeL2(vec)
}
