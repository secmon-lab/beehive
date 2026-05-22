package id_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

func TestNewULID_IsMonotonicAndUnique(t *testing.T) {
	const n = 100
	seen := map[string]struct{}{}
	var prev string
	for range n {
		v := id.NewULID()
		gt.Equal(t, len(v), 26) // Crockford base32, 26 chars
		_, dup := seen[v]
		gt.False(t, dup)
		seen[v] = struct{}{}
		if prev != "" {
			gt.True(t, v >= prev)
		}
		prev = v
	}
}
