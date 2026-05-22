// Package htmltext provides a minimal HTML-to-text extractor for feeding
// blog article bodies into the LLM-based IoC extractor. It intentionally
// avoids Mozilla Readability-style content scoring: the LLM tolerates
// boilerplate, so a simple "strip script/style/nav/footer/etc. and
// collect the rest" is enough.
package htmltext

import (
	"io"
	"net/url"
	"strings"
	"unicode"

	"github.com/m-mizutani/goerr/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Article is the trimmed-down result of Extract. Field names mirror the
// fields used by the old go-readability call site so the diff at the
// caller stays tiny.
type Article struct {
	Title    string
	BodyText string
	Excerpt  string
}

// Extract parses HTML from r and returns Title (from <title>), BodyText
// (text content with boilerplate elements removed and whitespace
// normalised) and Excerpt (from meta description / og:description).
//
// base is unused today but kept in the signature for future link
// resolution; pass nil if you don't have one.
func Extract(r io.Reader, base *url.URL) (Article, error) {
	_ = base
	doc, err := html.Parse(r)
	if err != nil {
		return Article{}, goerr.Wrap(err, "parse html")
	}

	var out Article
	out.Title = strings.TrimSpace(findTitle(doc))
	out.Excerpt = strings.TrimSpace(findMetaDescription(doc))

	var buf strings.Builder
	walk(doc, &buf)
	out.BodyText = normalizeWhitespace(buf.String())
	return out, nil
}

// skipTags lists elements whose subtree contributes no useful body text.
// We include the obvious boilerplate (nav/header/footer/aside) plus
// non-content technical tags (script/style/template/svg/iframe/form).
var skipTags = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Style:    true,
	atom.Noscript: true,
	atom.Iframe:   true,
	atom.Nav:      true,
	atom.Header:   true,
	atom.Footer:   true,
	atom.Aside:    true,
	atom.Form:     true,
	atom.Template: true,
	atom.Svg:      true,
	atom.Head:     true,
}

// blockTags marks elements whose boundaries should produce a newline so
// that paragraphs/list items don't get smashed onto one line.
var blockTags = map[atom.Atom]bool{
	atom.P:          true,
	atom.Div:        true,
	atom.Br:         true,
	atom.Li:         true,
	atom.Ul:         true,
	atom.Ol:         true,
	atom.H1:         true,
	atom.H2:         true,
	atom.H3:         true,
	atom.H4:         true,
	atom.H5:         true,
	atom.H6:         true,
	atom.Tr:         true,
	atom.Td:         true,
	atom.Th:         true,
	atom.Pre:        true,
	atom.Blockquote: true,
	atom.Section:    true,
	atom.Article:    true,
	atom.Hr:         true,
	atom.Table:      true,
	atom.Tbody:      true,
	atom.Thead:      true,
	atom.Dd:         true,
	atom.Dt:         true,
	atom.Dl:         true,
	atom.Figure:     true,
	atom.Figcaption: true,
}

func walk(n *html.Node, buf *strings.Builder) {
	if n == nil {
		return
	}
	switch n.Type {
	case html.TextNode:
		buf.WriteString(n.Data)
		return
	case html.ElementNode:
		if skipTags[n.DataAtom] {
			return
		}
		block := blockTags[n.DataAtom]
		if block {
			buf.WriteByte('\n')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, buf)
		}
		if block {
			buf.WriteByte('\n')
		}
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, buf)
	}
}

func findTitle(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.ElementNode && n.DataAtom == atom.Title {
		var sb strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				sb.WriteString(c.Data)
			}
		}
		return sb.String()
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

// findMetaDescription returns the first <meta name="description"> or
// <meta property="og:description"> content. Plain name="description"
// wins over og:description when both are present.
func findMetaDescription(n *html.Node) string {
	desc, og := findMetaDescriptionWalk(n)
	if desc != "" {
		return desc
	}
	return og
}

func findMetaDescriptionWalk(n *html.Node) (desc, og string) {
	if n == nil {
		return "", ""
	}
	if n.Type == html.ElementNode && n.DataAtom == atom.Meta {
		var name, property, content string
		for _, a := range n.Attr {
			switch strings.ToLower(a.Key) {
			case "name":
				name = strings.ToLower(a.Val)
			case "property":
				property = strings.ToLower(a.Val)
			case "content":
				content = a.Val
			}
		}
		if name == "description" {
			desc = content
		} else if property == "og:description" {
			og = content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d, o := findMetaDescriptionWalk(c)
		if desc == "" {
			desc = d
		}
		if og == "" {
			og = o
		}
		if desc != "" && og != "" {
			break
		}
	}
	return desc, og
}

// normalizeWhitespace collapses runs of inline whitespace to a single
// space, collapses runs of blank lines to a single newline, and trims
// surrounding whitespace.
func normalizeWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	var (
		pendingNewline bool
		pendingSpace   bool
		wroteAnything  bool
	)
	flushSeparators := func() {
		if !wroteAnything {
			pendingNewline = false
			pendingSpace = false
			return
		}
		if pendingNewline {
			b.WriteByte('\n')
		} else if pendingSpace {
			b.WriteByte(' ')
		}
		pendingNewline = false
		pendingSpace = false
	}

	for _, r := range s {
		switch {
		case r == '\n' || r == '\r':
			pendingNewline = true
		case unicode.IsSpace(r):
			// Covers ASCII space/tab/FF/VT plus Unicode whitespace
			// such as U+00A0 (NBSP, decoded from &nbsp;).
			pendingSpace = true
		default:
			flushSeparators()
			b.WriteRune(r)
			wroteAnything = true
		}
	}
	return strings.TrimSpace(b.String())
}
