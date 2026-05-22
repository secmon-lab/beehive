// Package extractor: gollem-backed LLMClient implementation.
//
// This file is the only place that imports gollem provider packages —
// the rest of the codebase depends on the small interfaces.LLMClient
// shape so unit tests can drive the extractor without bringing gollem
// into scope.
//
// Construction goes through NewLLMClient, which is told which provider
// to use by the cli/config.LLM struct. Adding a new provider means
// editing the switch in NewLLMClient (and nothing else).

package extractor

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// LLMConfig captures everything NewLLMClient needs to build a provider.
// Mirrors cli/config.LLM but kept independent so the extractor package
// does not depend on the CLI layer.
type LLMConfig struct {
	Provider string            // "gemini" | "openai" | "claude"
	Model    string            // model id (e.g. "gemini-1.5-pro-002")
	APIKey   string            // API key for openai / claude (Vertex AI on gemini uses ADC; key is optional)
	Args     map[string]string // provider-specific extras (e.g. project_id, location)
}

// NewLLMClient builds an interfaces.LLMClient (the narrow shape the
// extractor consumes) from cfg.
//
// MVP supports gemini only — additional providers can be added to the
// switch here as the wiring layer matures. Returns a tagged
// invalid-input error when the configuration is incomplete.
func NewLLMClient(ctx context.Context, cfg LLMConfig) (interfaces.LLMClient, error) {
	switch cfg.Provider {
	case "gemini":
		projectID := cfg.Args["project_id"]
		location := cfg.Args["location"]
		if projectID == "" || location == "" {
			return nil, goerr.New("gemini requires BEEHIVE_LLM_ARGS=project_id=...,location=...",
				goerr.V("args", cfg.Args),
				goerr.T(errutil.TagInvalidInput))
		}
		opts := []gemini.Option{}
		if cfg.Model != "" {
			opts = append(opts, gemini.WithModel(cfg.Model))
		}
		client, err := gemini.New(ctx, projectID, location, opts...)
		if err != nil {
			return nil, goerr.Wrap(err, "build gemini client",
				goerr.V("project_id", projectID),
				goerr.V("location", location))
		}
		return &gollemClient{inner: client}, nil

	default:
		return nil, goerr.New("unsupported BEEHIVE_LLM_PROVIDER (only \"gemini\" is wired)",
			goerr.V("provider", cfg.Provider),
			goerr.T(errutil.TagInvalidInput))
	}
}

// gollemClient adapts gollem.LLMClient to interfaces.LLMClient. Stays
// minimal — one method, no state beyond the wrapped client.
type gollemClient struct {
	inner gollem.LLMClient
}

func (g *gollemClient) GenerateJSON(ctx context.Context, system, prompt string, _ []byte, out any) error {
	// gollem prefers a *gollem.Parameter for the response schema, which
	// it builds via reflection from a Go value. We pass the destination
	// pointer (or its element type) and let gollem.ToSchema derive the
	// schema. The raw JSON-Schema bytes our domain interface still
	// accepts are kept for parity with the rest of the project but
	// ignored here — `out`'s type is the single source of truth.
	schema, err := gollem.ToSchema(out)
	if err != nil {
		return goerr.Wrap(err, "build response schema from out type")
	}

	session, err := g.inner.NewSession(ctx,
		gollem.WithSessionSystemPrompt(system),
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return goerr.Wrap(err, "open llm session")
	}

	resp, err := session.Generate(ctx, []gollem.Input{gollem.Text(prompt)})
	if err != nil {
		return goerr.Wrap(err, "generate llm response")
	}
	if len(resp.Texts) == 0 {
		return goerr.New("llm returned no text",
			goerr.V("input_tokens", strconv.Itoa(resp.InputToken)),
			goerr.V("output_tokens", strconv.Itoa(resp.OutputToken)))
	}

	if err := json.Unmarshal([]byte(resp.Texts[0]), out); err != nil {
		return goerr.Wrap(err, "decode llm json",
			goerr.V("body", resp.Texts[0]))
	}
	return nil
}
