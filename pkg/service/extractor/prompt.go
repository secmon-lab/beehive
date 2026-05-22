package extractor

import (
	"strings"

	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// systemPrompt is what we feed the LLM as the system role. Kept short
// and direct: list the IoC types we care about, ask for JSON, and warn
// against hallucinated context.
const systemPromptTemplate = `You are an IoC extraction assistant for security analysts.

Extract Indicators of Compromise (IoCs) from the supplied article text.
Return ONLY values that appear verbatim in the article — never invent or
generalise. Defanged forms ([.] / hxxp / etc.) are acceptable as Value
or Raw; do NOT refang them yourself, the post-processor handles that.

Supported IoC types (return type EXACTLY as one of these):
%s

For each IoC, populate:
  type        — one of the supported types above
  value       — the raw form as it appears in the article
  raw         — optional, defang variant or original surface form
  confidence  — 0.0 to 1.0, your subjective certainty

If an article contains no IoCs, return an empty array. Do not output any
prose, only the structured JSON array.`

// SystemPrompt returns the rendered system prompt, parameterised by the
// supported IoC type list at runtime so adding / removing types stays a
// single-source-of-truth edit (in pkg/domain/types).
func SystemPrompt() string {
	parts := make([]string, 0, len(types.AllIoCTypes()))
	for _, t := range types.AllIoCTypes() {
		parts = append(parts, "  - "+string(t))
	}
	return formatTemplate(systemPromptTemplate, strings.Join(parts, "\n"))
}

func formatTemplate(tmpl string, args ...any) string {
	out := tmpl
	for _, a := range args {
		if s, ok := a.(string); ok {
			out = strings.Replace(out, "%s", s, 1)
		}
	}
	return out
}

// MaxBodyChars caps how much article text we send to the LLM. Provider-
// agnostic ballpark: well within Gemini / Claude / GPT context windows
// while keeping cost predictable.
const MaxBodyChars = 20_000

// TrimForLLM returns the leading portion of body up to MaxBodyChars
// characters, sliced at a rune boundary so we never split a multi-byte
// character.
func TrimForLLM(body string) string {
	runes := []rune(body)
	if len(runes) <= MaxBodyChars {
		return body
	}
	return string(runes[:MaxBodyChars])
}
