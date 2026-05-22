package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestComputeIoCID_Deterministic(t *testing.T) {
	a := model.ComputeIoCID(types.IoCTypeIPv4, "1.2.3.4")
	b := model.ComputeIoCID(types.IoCTypeIPv4, "1.2.3.4")
	gt.Equal(t, a, b)
	// Length should be 64 hex chars (full sha256).
	gt.Equal(t, len(string(a)), 64)
}

func TestComputeIoCID_DistinguishesTypeAndValue(t *testing.T) {
	cases := []struct {
		t1, t2 types.IoCType
		v1, v2 string
	}{
		{types.IoCTypeIPv4, types.IoCTypeIPv6, "1.2.3.4", "1.2.3.4"},
		{types.IoCTypeDomain, types.IoCTypeURL, "example.com", "example.com"},
		{types.IoCTypeIPv4, types.IoCTypeIPv4, "1.2.3.4", "1.2.3.5"},
	}
	for _, c := range cases {
		gt.NotEqual(t,
			model.ComputeIoCID(c.t1, c.v1),
			model.ComputeIoCID(c.t2, c.v2),
		)
	}
}

func TestComputeRefID_DistinguishesBlogVsFeed(t *testing.T) {
	src := types.SourceID("src-1")
	a := model.ComputeRefID(src, types.ArticleID("article-1"), "")
	b := model.ComputeRefID(src, "", types.RunID("run-1"))
	gt.NotEqual(t, a, b)
	gt.Equal(t, len(string(a)), 64)
}

func TestComputeRefID_Idempotent(t *testing.T) {
	src := types.SourceID("src-1")
	art := types.ArticleID("art-1")
	gt.Equal(t,
		model.ComputeRefID(src, art, ""),
		model.ComputeRefID(src, art, ""),
	)
}
