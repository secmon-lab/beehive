package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestIoCTypeValid(t *testing.T) {
	for _, v := range types.AllIoCTypes() {
		gt.True(t, v.Valid())
	}
	gt.False(t, types.IoCType("nope").Valid())
	gt.False(t, types.IoCType("").Valid())
}

func TestSourceKindValid(t *testing.T) {
	gt.True(t, types.KindBlog.Valid())
	gt.True(t, types.KindFeed.Valid())
	gt.False(t, types.SourceKind("unknown").Valid())
}

func TestEnabledOverrideValid(t *testing.T) {
	gt.True(t, types.OverrideNone.Valid())
	gt.True(t, types.OverrideForceOn.Valid())
	gt.True(t, types.OverrideForceOff.Valid())
	gt.True(t, types.EnabledOverride("").Valid())
	gt.False(t, types.EnabledOverride("bogus").Valid())
}
