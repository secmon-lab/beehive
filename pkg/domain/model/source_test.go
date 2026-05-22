package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestEffectiveEnabled(t *testing.T) {
	cases := []struct {
		name     string
		disabled bool
		override types.EnabledOverride
		want     bool
	}{
		{"default enabled", false, types.OverrideNone, true},
		{"disabled in toml", true, types.OverrideNone, false},
		{"force_on overrides disabled toml", true, types.OverrideForceOn, true},
		{"force_off overrides enabled toml", false, types.OverrideForceOff, false},
		{"empty override == none", false, types.EnabledOverride(""), true},
		{"force_off wins over disabled false", false, types.OverrideForceOff, false},
		{"force_on wins over disabled true", true, types.OverrideForceOn, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &model.Source{Disabled: c.disabled}
			state := &model.SourceState{EnabledOverride: c.override}
			gt.Equal(t, s.EffectiveEnabled(state), c.want)
		})
	}
}

func TestEffectiveEnabledNilState(t *testing.T) {
	gt.True(t, (&model.Source{Disabled: false}).EffectiveEnabled(nil))
	gt.False(t, (&model.Source{Disabled: true}).EffectiveEnabled(nil))
}
