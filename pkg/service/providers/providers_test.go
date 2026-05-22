package providers_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/blog"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
	"github.com/secmon-lab/beehive/pkg/service/providers"
)

// TestAll_RegistersEveryProvider asserts the wiring layer hands back a
// registry containing every type id the codebase knows about. If a new
// provider is added but the wiring is forgotten, this test fails.
func TestAll_RegistersEveryProvider(t *testing.T) {
	reg, err := providers.All(nil)
	gt.NoError(t, err)

	cases := []struct {
		typeID string
		want   types.SourceKind
	}{
		{blog.RSSTypeID, types.KindBlog},
		{feed.AbuseIPDBTypeID, types.KindFeed},
		{feed.CINSScoreTypeID, types.KindFeed},
		{feed.FeodoTrackerTypeID, types.KindFeed},
		{feed.URLhausTypeID, types.KindFeed},
		{feed.ThreatFoxTypeID, types.KindFeed},
	}
	for _, c := range cases {
		t.Run(c.typeID, func(t *testing.T) {
			prov, err := reg.Resolve(c.typeID)
			gt.NoError(t, err)
			gt.Equal(t, prov.Kind(), c.want)
		})
	}
}

func TestAll_BlogContract(t *testing.T) {
	reg, err := providers.All(nil)
	gt.NoError(t, err)
	prov, err := reg.Resolve(blog.RSSTypeID)
	gt.NoError(t, err)
	_, ok := prov.(interfaces.BlogProvider)
	gt.True(t, ok)
}

func TestAll_FeedContract(t *testing.T) {
	reg, err := providers.All(nil)
	gt.NoError(t, err)
	prov, err := reg.Resolve(feed.CINSScoreTypeID)
	gt.NoError(t, err)
	_, ok := prov.(interfaces.FeedProvider)
	gt.True(t, ok)
}
