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
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// LLMConfig captures everything NewLLMClient needs to build a provider.
// Mirrors cli/config.LLM but kept independent so the extractor package
// does not depend on the CLI layer.
type LLMConfig struct {
	Provider string            // "gemini" | "claude"
	Model    string            // model id, REQUIRED — Validate forbids empty.
	APIKey   string            // API key for claude (Anthropic direct API). Empty when using Vertex AI.
	Args     map[string]string // provider-specific extras: project_id, location.
}

// claudeAuthMode tells the lazy builder which gollem constructor to call
// for provider="claude". Decided once in NewLLMClient from the
// (Args, APIKey) combination so ensureInner stays mechanical.
type claudeAuthMode int

const (
	claudeAuthVertex claudeAuthMode = iota + 1
	claudeAuthAPIKey
)

// NewLLMClient builds an interfaces.LLMClient (the narrow shape the
// extractor consumes) from cfg.
//
// Construction is **shape-only**: cfg is validated synchronously but the
// underlying provider client is created lazily on the first
// GenerateJSON call. This matters for environments that have valid env
// vars but no live ADC credentials (CI, validate-only invocations,
// tests of the HTTP boot path) — they must not hard-fail at startup.
//
// Supported providers: gemini (Vertex AI Gemini, ADC), claude (Vertex AI
// Claude via ADC, or Anthropic direct API via API key).
func NewLLMClient(_ context.Context, cfg LLMConfig) (interfaces.LLMClient, error) {
	if cfg.Model == "" {
		return nil, goerr.New("BEEHIVE_LLM_MODEL is required (caller must not rely on provider-internal defaults)",
			goerr.V("provider", cfg.Provider),
			goerr.T(errutil.TagInvalidInput))
	}

	switch cfg.Provider {
	case "gemini":
		if cfg.Args["project_id"] == "" || cfg.Args["location"] == "" {
			return nil, goerr.New("gemini requires BEEHIVE_LLM_ARGS=project_id=...,location=...",
				goerr.V("args", cfg.Args),
				goerr.T(errutil.TagInvalidInput))
		}
		return &gollemClient{cfg: cfg}, nil

	case "claude":
		mode, err := decideClaudeAuthMode(cfg)
		if err != nil {
			return nil, err
		}
		return &gollemClient{cfg: cfg, claudeAuth: mode}, nil

	default:
		return nil, goerr.New("unsupported BEEHIVE_LLM_PROVIDER (only gemini / claude)",
			goerr.V("provider", cfg.Provider),
			goerr.T(errutil.TagInvalidInput))
	}
}

// decideClaudeAuthMode inspects (Args, APIKey) and selects the auth
// path. Returns TagInvalidInput when both or neither are configured —
// we refuse to guess.
func decideClaudeAuthMode(cfg LLMConfig) (claudeAuthMode, error) {
	hasVertex := cfg.Args["project_id"] != "" && cfg.Args["location"] != ""
	hasAPIKey := cfg.APIKey != ""

	switch {
	case hasVertex && hasAPIKey:
		return 0, goerr.New("claude: BEEHIVE_LLM_ARGS=project_id/location and BEEHIVE_LLM_API_KEY are mutually exclusive — pick one auth path",
			goerr.T(errutil.TagInvalidInput))
	case hasVertex:
		return claudeAuthVertex, nil
	case hasAPIKey:
		return claudeAuthAPIKey, nil
	default:
		return 0, goerr.New("claude requires either BEEHIVE_LLM_ARGS=project_id=...,location=... (Vertex AI via ADC) or BEEHIVE_LLM_API_KEY (Anthropic direct API)",
			goerr.T(errutil.TagInvalidInput))
	}
}

// gollemClient adapts gollem.LLMClient to interfaces.LLMClient with
// lazy initialisation — the actual provider client is only built on
// the first GenerateJSON call, behind a sync.Once. Subsequent calls
// reuse the cached client (or its sticky init error).
type gollemClient struct {
	cfg        LLMConfig
	claudeAuth claudeAuthMode // only meaningful when cfg.Provider == "claude"

	once    sync.Once
	inner   gollem.LLMClient
	initErr error
}

func (g *gollemClient) ensureInner(ctx context.Context) (gollem.LLMClient, error) {
	g.once.Do(func() {
		switch g.cfg.Provider {
		case "gemini":
			client, err := gemini.New(ctx,
				g.cfg.Args["project_id"], g.cfg.Args["location"],
				gemini.WithModel(g.cfg.Model),
			)
			if err != nil {
				g.initErr = goerr.Wrap(err, "build gemini client",
					goerr.V("project_id", g.cfg.Args["project_id"]),
					goerr.V("location", g.cfg.Args["location"]),
					goerr.V("model", g.cfg.Model))
				return
			}
			g.inner = client

		case "claude":
			switch g.claudeAuth {
			case claudeAuthVertex:
				client, err := claude.NewWithVertex(ctx,
					g.cfg.Args["location"], g.cfg.Args["project_id"],
					claude.WithVertexModel(g.cfg.Model),
				)
				if err != nil {
					g.initErr = goerr.Wrap(err, "build claude vertex client",
						goerr.V("project_id", g.cfg.Args["project_id"]),
						goerr.V("location", g.cfg.Args["location"]),
						goerr.V("model", g.cfg.Model))
					return
				}
				g.inner = client
			case claudeAuthAPIKey:
				client, err := claude.New(ctx, g.cfg.APIKey,
					claude.WithModel(g.cfg.Model),
				)
				if err != nil {
					// API key value intentionally NOT logged.
					g.initErr = goerr.Wrap(err, "build claude api-key client",
						goerr.V("model", g.cfg.Model))
					return
				}
				g.inner = client
			default:
				g.initErr = goerr.New("claude: auth mode not decided (programming error)",
					goerr.V("mode", g.claudeAuth),
					goerr.T(errutil.TagInvalidInput))
			}

		default:
			g.initErr = goerr.New("unsupported BEEHIVE_LLM_PROVIDER",
				goerr.V("provider", g.cfg.Provider),
				goerr.T(errutil.TagInvalidInput))
		}
	})
	return g.inner, g.initErr
}

func (g *gollemClient) GenerateJSON(ctx context.Context, system, prompt string, _ []byte, out any) error {
	inner, err := g.ensureInner(ctx)
	if err != nil {
		return err
	}
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

	session, err := inner.NewSession(ctx,
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
