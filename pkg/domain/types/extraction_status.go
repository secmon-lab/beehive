package types

// ExtractionStatus is the LLM extraction lifecycle of an Article.
type ExtractionStatus string

const (
	ExtractionPending    ExtractionStatus = "pending"
	ExtractionInProgress ExtractionStatus = "in_progress"
	ExtractionCompleted  ExtractionStatus = "completed"
	ExtractionFailed     ExtractionStatus = "failed"
	// ExtractionSkipped is set when LLM extraction does not apply (e.g.
	// extraction is intentionally not run). Not used by the current MVP
	// providers but kept for future flexibility.
	ExtractionSkipped ExtractionStatus = "skipped"
)

func (s ExtractionStatus) String() string { return string(s) }
