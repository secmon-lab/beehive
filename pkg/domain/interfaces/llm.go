package interfaces

import (
	"context"
)

// LLMClient is the minimal contract the extractor needs. We keep it
// narrower than gollem.LLMClient so unit tests can fake responses without
// pulling in the full gollem surface.
//
// Implementations MUST honour ctx cancellation.
type LLMClient interface {
	// GenerateJSON sends `prompt` to the model and unmarshals the response
	// (which the model is constrained to emit in the shape described by
	// `schema`) into `out`. `schema` is the JSON Schema as a marshalled
	// byte slice — the extractor builds it from interfaces.IoCSeed.
	GenerateJSON(ctx context.Context, system, prompt string, schema []byte, out any) error
}
