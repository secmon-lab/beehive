package extractor_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/extractor"
)

// fakeLLM returns a pre-baked JSON payload regardless of the prompt.
type fakeLLM struct {
	payload string
	err     error
}

func (f *fakeLLM) GenerateJSON(_ context.Context, _ string, _ string, _ []byte, out any) error {
	if f.err != nil {
		return f.err
	}
	return json.NewDecoder(strings.NewReader(f.payload)).Decode(out)
}

func TestExtractor_Extract(t *testing.T) {
	llm := &fakeLLM{payload: `{
        "iocs": [
            {"type": "ipv4",   "value": "8.8.8.8",            "confidence": 0.9},
            {"type": "domain", "value": "Evil[.]Example",     "confidence": 0.8},
            {"type": "md5",    "value": "DEADBEEFDEADBEEFDEADBEEFDEADBEEF", "confidence": 0.7},
            {"type": "ipv4",   "value": "127.0.0.1",          "confidence": 0.5},
            {"type": "bogus",  "value": "x",                  "confidence": 0.5}
        ]
    }`}
	e := extractor.New(llm)
	got, err := e.Extract(context.Background(), "article body")
	gt.NoError(t, err)
	// 127.0.0.1 (private), bogus type — both dropped. The other three remain.
	gt.A(t, got).Length(3)

	gt.Equal(t, got[0].Type, types.IoCTypeIPv4)
	gt.Equal(t, got[1].Type, types.IoCTypeDomain)
	gt.Equal(t, got[1].Value, "evil.example") // refanged + idna
	gt.Equal(t, got[2].Type, types.IoCTypeMD5)
	gt.Equal(t, got[2].Value, "deadbeefdeadbeefdeadbeefdeadbeef")
}

func TestExtractor_EmptyBody(t *testing.T) {
	e := extractor.New(&fakeLLM{payload: `{"iocs":[]}`})
	got, err := e.Extract(context.Background(), "")
	gt.NoError(t, err)
	gt.A(t, got).Length(0)
}

func TestExtractor_LLMError(t *testing.T) {
	llm := &fakeLLM{err: errFake}
	_, err := extractor.New(llm).Extract(context.Background(), "body")
	gt.Error(t, err)
}

func TestSystemPromptContainsAllTypes(t *testing.T) {
	prompt := extractor.SystemPrompt()
	for _, ty := range types.AllIoCTypes() {
		gt.True(t, strings.Contains(prompt, string(ty)))
	}
}

func TestTrimForLLM(t *testing.T) {
	long := strings.Repeat("a", extractor.MaxBodyChars+100)
	out := extractor.TrimForLLM(long)
	gt.Equal(t, len(out), extractor.MaxBodyChars)
	gt.Equal(t, extractor.TrimForLLM("short"), "short")
}

var errFake = newErr("boom")

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }
func newErr(s string) error      { return &fakeErr{msg: s} }
