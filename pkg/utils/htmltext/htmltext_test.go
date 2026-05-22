package htmltext_test

import (
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/utils/htmltext"
)

func TestExtract_PlainParagraph(t *testing.T) {
	in := `<html><head><title>Hello</title></head><body><p>Hello world</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.Title, "Hello")
	gt.Equal(t, got.BodyText, "Hello world")
	gt.Equal(t, got.Excerpt, "")
}

func TestExtract_StripsBoilerplateTags(t *testing.T) {
	in := `
<html>
  <head>
    <title>Page</title>
    <style>body{color:red;}</style>
    <script>var x = 1;</script>
  </head>
  <body>
    <nav>nav links</nav>
    <header>top</header>
    <aside>ads</aside>
    <main>
      <p>Real content here.</p>
      <noscript>fallback</noscript>
      <iframe>embedded</iframe>
    </main>
    <footer>copyright 2026</footer>
  </body>
</html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.BodyText, "Real content here.")
	// None of the boilerplate text should leak through.
	for _, banned := range []string{
		"var x", "color:red", "nav links", "top", "ads",
		"fallback", "embedded", "copyright",
	} {
		if strings.Contains(got.BodyText, banned) {
			t.Fatalf("BodyText should not contain %q, got: %q", banned, got.BodyText)
		}
	}
}

func TestExtract_BlockBoundariesProduceNewlines(t *testing.T) {
	in := `<html><body>
<p>First paragraph.</p>
<p>Second paragraph.</p>
<ul><li>one</li><li>two</li></ul>
</body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	// Paragraphs / list items must be separated by a newline so they
	// don't collapse into "First paragraph.Second paragraph.one two".
	lines := strings.Split(got.BodyText, "\n")
	wantOrder := []string{"First paragraph.", "Second paragraph.", "one", "two"}
	idx := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if idx >= len(wantOrder) {
			t.Fatalf("unexpected extra line %q in %q", l, got.BodyText)
		}
		if l != wantOrder[idx] {
			t.Fatalf("line %d: got %q, want %q (full=%q)", idx, l, wantOrder[idx], got.BodyText)
		}
		idx++
	}
	if idx != len(wantOrder) {
		t.Fatalf("missing lines, got %d/%d (full=%q)", idx, len(wantOrder), got.BodyText)
	}
}

func TestExtract_MetaDescription(t *testing.T) {
	in := `<html><head>
<title>T</title>
<meta name="description" content="short blurb">
</head><body><p>body</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.Excerpt, "short blurb")
}

func TestExtract_OGDescriptionFallback(t *testing.T) {
	in := `<html><head>
<meta property="og:description" content="og blurb">
</head><body><p>body</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.Excerpt, "og blurb")
}

func TestExtract_MetaDescriptionWinsOverOG(t *testing.T) {
	in := `<html><head>
<meta property="og:description" content="og blurb">
<meta name="description" content="real blurb">
</head><body><p>body</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.Excerpt, "real blurb")
}

func TestExtract_WhitespaceCollapsed(t *testing.T) {
	in := `<html><body><p>too   much     space\there</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	// \t in the source HTML is a literal backslash+t (raw string), not a
	// tab. But the runs of spaces must collapse.
	if strings.Contains(got.BodyText, "  ") {
		t.Fatalf("expected collapsed whitespace, got %q", got.BodyText)
	}
}

func TestExtract_MalformedHTMLDoesNotPanic(t *testing.T) {
	in := `<html><body><p>open paragraph with <b>unclosed tags</body>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	// html.Parse repairs malformed input; we expect best-effort body.
	if !strings.Contains(got.BodyText, "open paragraph") {
		t.Fatalf("expected body to contain repaired text, got %q", got.BodyText)
	}
}

func TestExtract_EmptyBodyReturnsEmpty(t *testing.T) {
	in := `<html><head><title>T</title></head><body></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.Title, "T")
	gt.Equal(t, got.BodyText, "")
}

func TestExtract_HTMLEntities(t *testing.T) {
	in := `<html><body><p>foo &amp; bar &lt;baz&gt;</p></body></html>`
	got, err := htmltext.Extract(strings.NewReader(in), nil)
	gt.NoError(t, err)
	gt.Equal(t, got.BodyText, "foo & bar <baz>")
}
