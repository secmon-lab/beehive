package config

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/llm/openai"
	"github.com/urfave/cli/v3"
)

var (
	errUnsupportedProvider = goerr.New("unsupported LLM provider")
)

// LLM represents LLM configuration
type LLM struct {
	Provider       string
	GeminiProject  string
	GeminiLocation string
	OpenAIAPIKey   string
	ClaudeAPIKey   string
	Model          string
}

// Flags returns CLI flags for LLM configuration
func (l *LLM) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "llm-provider",
			Usage:       "LLM provider (gemini, openai, claude)",
			Value:       "gemini",
			Destination: &l.Provider,
			Sources:     cli.EnvVars("BEEHIVE_LLM_PROVIDER"),
		},
		&cli.StringFlag{
			Name:        "gemini-project",
			Usage:       "Google Cloud project ID for Gemini",
			Destination: &l.GeminiProject,
			Sources:     cli.EnvVars("BEEHIVE_GEMINI_PROJECT"),
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "Google Cloud location for Gemini",
			Value:       "us-central1",
			Destination: &l.GeminiLocation,
			Sources:     cli.EnvVars("BEEHIVE_GEMINI_LOCATION"),
		},
		&cli.StringFlag{
			Name:        "openai-api-key",
			Usage:       "OpenAI API key",
			Destination: &l.OpenAIAPIKey,
			Sources:     cli.EnvVars("BEEHIVE_OPENAI_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "claude-api-key",
			Usage:       "Claude API key",
			Destination: &l.ClaudeAPIKey,
			Sources:     cli.EnvVars("BEEHIVE_CLAUDE_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "llm-model",
			Usage:       "LLM model name",
			Destination: &l.Model,
			Sources:     cli.EnvVars("BEEHIVE_LLM_MODEL"),
		},
	}
}

// NewLLMClient creates a new LLM client based on the configuration
func (l *LLM) NewLLMClient(ctx context.Context) (gollem.LLMClient, error) {
	switch l.Provider {
	case "gemini":
		opts := []gemini.Option{
			gemini.WithThinkingBudget(0),
		}
		if l.Model != "" {
			opts = append(opts, gemini.WithModel(l.Model))
		}
		client, err := gemini.New(ctx, l.GeminiProject, l.GeminiLocation, opts...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create Gemini client",
				goerr.V("project", l.GeminiProject),
				goerr.V("location", l.GeminiLocation))
		}
		return client, nil

	case "openai":
		opts := []openai.Option{}
		if l.Model != "" {
			opts = append(opts, openai.WithModel(l.Model))
		}
		client, err := openai.New(ctx, l.OpenAIAPIKey, opts...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create OpenAI client")
		}
		return client, nil

	case "claude":
		opts := []claude.Option{}
		if l.Model != "" {
			opts = append(opts, claude.WithModel(l.Model))
		}
		client, err := claude.New(ctx, l.ClaudeAPIKey, opts...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create Claude client")
		}
		return client, nil

	default:
		return nil, goerr.Wrap(errUnsupportedProvider, "invalid provider",
			goerr.V("provider", l.Provider))
	}
}
