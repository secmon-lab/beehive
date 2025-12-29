package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func TestNewTag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid single char lowercase",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "valid single char uppercase",
			input:   "A",
			wantErr: false,
		},
		{
			name:    "valid simple",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "valid with uppercase",
			input:   "Test",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			input:   "threat-intel",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			input:   "test_tag",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			input:   "my-tag-123",
			wantErr: false,
		},
		{
			name:    "valid mixed case with underscore",
			input:   "My_Tag_123",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with hyphen",
			input:   "-test",
			wantErr: true,
		},
		{
			name:    "ends with hyphen",
			input:   "test-",
			wantErr: true,
		},
		{
			name:    "starts with underscore",
			input:   "_test",
			wantErr: true,
		},
		{
			name:    "ends with underscore",
			input:   "test_",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := types.NewTag(tt.input)
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.S(t, tag.String()).Equal(tt.input)
			}
		})
	}
}

func TestNewTags(t *testing.T) {
	t.Run("valid tags", func(t *testing.T) {
		input := []string{"vendor", "google", "threat-intel"}
		tags, err := types.NewTags(input)
		gt.NoError(t, err)
		gt.A(t, tags).Length(3)
		gt.V(t, tags.Strings()).Equal(input)
	})

	t.Run("invalid tag in list", func(t *testing.T) {
		input := []string{"vendor", "-Invalid", "threat-intel"}
		_, err := types.NewTags(input)
		gt.Error(t, err)
	})

	t.Run("empty list", func(t *testing.T) {
		input := []string{}
		tags, err := types.NewTags(input)
		gt.NoError(t, err)
		gt.A(t, tags).Length(0)
	})
}
