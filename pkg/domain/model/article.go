package model

import (
	"time"
	"unicode/utf8"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// MaxBodyTextBytes is the upper bound on the BodyText field before it
// is persisted. Firestore caps each document at 1 MiB and a single
// payload field at ~1,048,487 bytes, so we leave generous headroom for
// the other fields (URL / Title / Summary / hashes / timestamps).
const MaxBodyTextBytes = 900 * 1024

// TruncatedSuffix is appended whenever TruncateBodyText shortens the
// input. Kept exported so tests / debug logs can detect it.
const TruncatedSuffix = "\n...[truncated]"

// TruncateBodyText shortens body so that the result fits in
// MaxBodyTextBytes. Cuts on a UTF-8 boundary to avoid leaving a half
// rune in Firestore. Returns the original string untouched when it is
// already small enough.
func TruncateBodyText(body string) string {
	if len(body) <= MaxBodyTextBytes {
		return body
	}
	keep := MaxBodyTextBytes - len(TruncatedSuffix)
	if keep < 0 {
		keep = 0
	}
	// Walk back to the previous rune start so we never produce invalid
	// UTF-8 — Firestore rejects malformed strings on write.
	for keep > 0 && !utf8.RuneStart(body[keep]) {
		keep--
	}
	return body[:keep] + TruncatedSuffix
}

// Article is one fetched blog post / page. Only blog-kind sources produce
// Articles; feed-kind sources skip this collection entirely.
//
// Stored at Firestore path: articles/{ID}.
//
// ContentHash is sha256(BodyText) computed BEFORE any truncation, so that
// re-fetches can decide whether the page actually changed without being
// confused by the truncation marker. BodyText itself is kept as close to
// the full readable text as possible (truncated by TruncateBodyText
// near the 1 MiB document-size limit, with a "...[truncated]" suffix
// when so).
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
