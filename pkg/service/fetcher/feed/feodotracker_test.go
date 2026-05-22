package feed_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

func TestParseFeodoTracker(t *testing.T) {
	body := []byte("# Feodo Tracker\n# Updated\n198.51.100.10\n203.0.113.42\n")
	seeds := feed.ParseFeodoTracker(body)
	gt.A(t, seeds).Length(2)
	gt.Equal(t, seeds[0].Confidence, 0.95)
}
