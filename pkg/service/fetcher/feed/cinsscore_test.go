package feed_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

func TestParseCINSScore(t *testing.T) {
	body := []byte("# header\n1.2.3.4\n5.6.7.8\n\n# 9.9.9.9\n10.0.0.1\n")
	seeds := feed.ParseCINSScore(body)
	// 10.0.0.1 is private and dropped, header/comment lines are skipped.
	gt.A(t, seeds).Length(2)
	gt.Equal(t, seeds[0].Type, types.IoCTypeIPv4)
	gt.Equal(t, seeds[0].Value, "1.2.3.4")
}
