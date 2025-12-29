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
	tests := []struct {
		name   string
		schema types.FeedSchema
		want   string
	}{
		{
			name:   "URLhaus",
			schema: types.FeedSchemaAbuseCHURLhaus,
			want:   "https://urlhaus.abuse.ch/downloads/csv_recent/",
		},
		{
			name:   "ThreatFox",
			schema: types.FeedSchemaAbuseCHThreatFox,
			want:   "https://threatfox.abuse.ch/export/csv/recent/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schema.DefaultURL()
			gt.S(t, got).Equal(tt.want)
		})
	}
}

func TestAllFeedSchemas(t *testing.T) {
	schemas := types.AllFeedSchemas()
	gt.A(t, schemas).Length(2)
	gt.A(t, schemas).At(0, func(t testing.TB, v types.FeedSchema) {
		gt.V(t, v).Equal(types.FeedSchemaAbuseCHURLhaus)
	})
	gt.A(t, schemas).At(1, func(t testing.TB, v types.FeedSchema) {
		gt.V(t, v).Equal(types.FeedSchemaAbuseCHThreatFox)
	})
}
