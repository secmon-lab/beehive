// Package extractor turns an article body into normalised IoCs using a
// gollem-backed LLM. The two pieces (LLM call + post-processing) are
// kept separate so tests can drive each independently.
package extractor

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/normalize"
)

// Extractor is the concrete implementation of interfaces.Extractor.
type Extractor struct {
	llm interfaces.LLMClient
}

// New builds an Extractor bound to an LLM client.
func New(llm interfaces.LLMClient) *Extractor {
	return &Extractor{llm: llm}
}

// llmResponse is what we ask the model to emit. The shape must stay in
// sync with the schema we describe in the prompt.
type llmResponse struct {
	IoCs []interfaces.IoCSeed `json:"iocs"`
}

// schemaBytes describes the JSON shape we expect back. gollem accepts
// raw JSON schema bytes — keeping the schema inline avoids a third
// piece to keep in sync.
var schemaBytes = []byte(`{
  "type": "object",
  "properties": {
    "iocs": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type":       {"type": "string"},
          "value":      {"type": "string"},
          "raw":        {"type": "string"},
          "confidence": {"type": "number"}
        },
        "required": ["type", "value", "confidence"]
      }
    }
  },
  "required": ["iocs"]
}`)

// Extract runs the LLM and post-processes the response.
func (e *Extractor) Extract(ctx context.Context, body string) ([]*interfaces.IoCSeed, error) {
	if e == nil || e.llm == nil {
		return nil, goerr.New("extractor not initialised")
	}
	if body == "" {
		return nil, nil
	}

	var resp llmResponse
	if err := e.llm.GenerateJSON(ctx, SystemPrompt(), TrimForLLM(body), schemaBytes, &resp); err != nil {
		return nil, goerr.Wrap(err, "llm extract")
	}
	return Postprocess(resp.IoCs), nil
}

// Postprocess normalises every seed and drops the ones we cannot make
// sense of. Exposed so unit tests can drive it directly.
func Postprocess(seeds []interfaces.IoCSeed) []*interfaces.IoCSeed {
	out := make([]*interfaces.IoCSeed, 0, len(seeds))
	for i := range seeds {
		s := seeds[i] // copy
		if !s.Type.Valid() {
			continue
		}
		norm, err := normalize.IoC(s.Type, s.Value)
		if err != nil {
			continue
		}
		if s.Raw == "" {
			s.Raw = s.Value
		}
		s.Value = norm
		out = append(out, &s)
	}
	return out
}

// MarshalSeeds is a tiny helper used by integration tests / debug
// tooling to log what the LLM returned. Wraps json.Marshal so callers
// do not need to thread the encoding/json import.
func MarshalSeeds(seeds []*interfaces.IoCSeed) ([]byte, error) {
	if seeds == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(seeds)
}

// UnsupportedType returns true when the underlying normalize error is
// the "unsupported type" sentinel — primarily for tests that want to
// assert the specific failure mode.
func UnsupportedType(err error) bool {
	return errors.Is(err, normalize.ErrUnsupportedType)
}

// Used to keep the types import alive while keeping the file lint-clean
// in case future helpers move out of this file.
var _ = types.IoCTypeIPv4
