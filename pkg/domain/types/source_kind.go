package types

// SourceKind classifies a Source into one of two pipelines.
//
//   - KindBlog : provider fetches articles (with HTML body) and LLM extracts IoCs.
//   - KindFeed : provider returns IoC seeds directly; no LLM is involved.
type SourceKind string

const (
	KindBlog SourceKind = "blog"
	KindFeed SourceKind = "feed"
)

func (k SourceKind) String() string { return string(k) }

func (k SourceKind) Valid() bool {
	switch k {
	case KindBlog, KindFeed:
		return true
	default:
		return false
	}
}
