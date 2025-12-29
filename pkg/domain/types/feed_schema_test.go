package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestNewFeedSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    types.FeedSchema
		wantErr bool
	}{
		{
			name:    "abuse_ch_urlhaus",
			input:   "abuse_ch_urlhaus",
			want:    types.FeedSchemaAbuseCHURLhaus,
			wantErr: false,
		},
		{
			name:    "abuse_ch_threatfox",
			input:   "abuse_ch_threatfox",
			want:    types.FeedSchemaAbuseCHThreatFox,
			wantErr: false,
		},
		{
			name:    "invalid schema",
			input:   "unknown_schema",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := types.NewFeedSchema(tt.input)
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.V(t, schema).Equal(tt.want)
				gt.S(t, schema.String()).Equal(tt.input)
			}
		})
	}
}

func TestFeedSchemaDefaultURL(t *testing.T) {
	// DefaultURL() now returns empty string - actual URLs are in feed service
	schema := types.FeedSchemaAbuseCHURLhaus
	got := schema.DefaultURL()
	gt.S(t, got).Equal("")
}

func TestAllFeedSchemas(t *testing.T) {
	schemas := types.AllFeedSchemas()
	// Verify we have all 47 feed schemas registered
	gt.A(t, schemas).Length(47).Describe("should have all 47 feed schemas")

	// Verify some key schemas are present
	var hasURLhaus, hasMontysecurityAll, hasThreatViewIPHigh bool
	for _, s := range schemas {
		switch s {
		case types.FeedSchemaAbuseCHURLhaus:
			hasURLhaus = true
		case types.FeedSchemaMontysecurityAll:
			hasMontysecurityAll = true
		case types.FeedSchemaThreatViewIPHigh:
			hasThreatViewIPHigh = true
		}
	}

	gt.True(t, hasURLhaus).Describe("should contain abuse_ch_urlhaus")
	gt.True(t, hasMontysecurityAll).Describe("should contain montysecurity_all")
	gt.True(t, hasThreatViewIPHigh).Describe("should contain threatview_ip_high")
}
