package model_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

func TestTruncateBodyText_NoTruncationWhenSmall(t *testing.T) {
	in := "small body"
	got := model.TruncateBodyText(in)
	gt.Equal(t, got, in)
}

func TestTruncateBodyText_TruncatesWhenOverLimit(t *testing.T) {
	in := strings.Repeat("a", model.MaxBodyTextBytes+100)
	got := model.TruncateBodyText(in)
	gt.True(t, len(got) <= model.MaxBodyTextBytes)
	gt.True(t, strings.HasSuffix(got, model.TruncatedSuffix))
}

func TestTruncateBodyText_KeepsUTF8Boundary(t *testing.T) {
	// Position the cut point (MaxBodyTextBytes - len(TruncatedSuffix))
	// inside the middle byte of a "あ" (3 bytes in UTF-8). Without
	// rune-awareness the result would be invalid UTF-8 and Firestore
	// would refuse the write.
	cut := model.MaxBodyTextBytes - len(model.TruncatedSuffix)
	prefix := strings.Repeat("a", cut-1)
	// Pad enough trailing data that the input exceeds MaxBodyTextBytes
	// — otherwise the function returns the original string untouched.
	in := prefix + strings.Repeat("あ", 200)

	got := model.TruncateBodyText(in)
	gt.True(t, utf8.ValidString(got))
	gt.True(t, strings.HasSuffix(got, model.TruncatedSuffix))
	gt.True(t, len(got) <= model.MaxBodyTextBytes)
}
