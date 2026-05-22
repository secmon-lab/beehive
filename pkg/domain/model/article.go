package model

import (
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Article is one fetched blog post / page. Only blog-kind sources produce
// Articles; feed-kind sources skip this collection entirely.
//
// Stored at Firestore path: articles/{ID}.
//
// ContentHash is sha256(BodyText) computed BEFORE any truncation, so that
// re-fetches can decide whether the page actually changed without being
// confused by the truncation marker. BodyText itself is kept as close to
// the full readable text as possible (truncated only near the 1 MiB
// document-size limit, with a "...[truncated]" suffix when so).
type Article struct {
	ID               types.ArticleID
	SourceID         types.SourceID
	URL              string
	Title            string
	PublishedAt      time.Time
	FetchedAt        time.Time
	ContentHash      string
	BodyText         string
	Summary          string
	ExtractionStatus types.ExtractionStatus
	ExtractedAt      time.Time
	ExtractorModel   string // gollem provider / model identifier when extracted
	IoCCount         int
	RunID            types.RunID
}
