package extractor_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/service/extractor"
)

func TestNewLLMClient_Gemini_OK(t *testing.T) {
	c, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-pro",
		Args:     map[string]string{"project_id": "p", "location": "global"},
	})
	gt.NoError(t, err)
	gt.NotNil(t, c)

	cfg, ok := extractor.LLMConfigOf(c)
	gt.True(t, ok)
	gt.Equal(t, cfg.Model, "gemini-2.5-pro")
	gt.Equal(t, cfg.Args["project_id"], "p")
	gt.Equal(t, cfg.Args["location"], "global")
}

func TestNewLLMClient_Gemini_MissingArgs(t *testing.T) {
	_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-pro",
	})
	gt.Error(t, err)
}

func TestNewLLMClient_RequiresModel(t *testing.T) {
	// Empty Model must error regardless of provider — we refuse to
	// fall back to gollem's provider-internal default model.
	for _, prov := range []string{"gemini", "claude"} {
		t.Run(prov, func(t *testing.T) {
			_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
				Provider: prov,
				Args:     map[string]string{"project_id": "p", "location": "global"},
				APIKey:   "k",
			})
			gt.Error(t, err)
		})
	}
}

func TestNewLLMClient_Claude_Vertex_OK(t *testing.T) {
	c, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4@20250514",
		Args:     map[string]string{"project_id": "p", "location": "global"},
	})
	gt.NoError(t, err)
	gt.NotNil(t, c)

	cfg, ok := extractor.LLMConfigOf(c)
	gt.True(t, ok)
	gt.Equal(t, cfg.Model, "claude-sonnet-4@20250514")
	gt.Equal(t, cfg.Args["project_id"], "p")
	gt.Equal(t, cfg.Args["location"], "global")
	gt.Equal(t, cfg.APIKey, "")
}

func TestNewLLMClient_Claude_APIKey_OK(t *testing.T) {
	c, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   "sk-test",
	})
	gt.NoError(t, err)
	gt.NotNil(t, c)

	cfg, ok := extractor.LLMConfigOf(c)
	gt.True(t, ok)
	gt.Equal(t, cfg.Model, "claude-sonnet-4-5-20250929")
	gt.Equal(t, cfg.APIKey, "sk-test")
}

func TestNewLLMClient_Claude_BothPaths_Conflict(t *testing.T) {
	_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4@20250514",
		Args:     map[string]string{"project_id": "p", "location": "global"},
		APIKey:   "sk-test",
	})
	gt.Error(t, err)
}

func TestNewLLMClient_Claude_NeitherPath(t *testing.T) {
	_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4-5-20250929",
	})
	gt.Error(t, err)
}

func TestNewLLMClient_Claude_PartialVertex(t *testing.T) {
	// project_id without location must NOT silently fall through to
	// the API-key path. Without an API key it must error.
	_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4@20250514",
		Args:     map[string]string{"project_id": "p"},
	})
	gt.Error(t, err)
}

func TestNewLLMClient_UnsupportedProvider(t *testing.T) {
	_, err := extractor.NewLLMClient(context.Background(), extractor.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "sk-test",
	})
	gt.Error(t, err)
}
