package source_catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

type stubBlog struct{}

func (stubBlog) Kind() types.SourceKind { return types.KindBlog }
func (stubBlog) Fetch(_ context.Context, _ *model.Source) ([]*interfaces.FetchedArticle, error) {
	return nil, nil
}

type stubFeed struct{}

func (stubFeed) Kind() types.SourceKind { return types.KindFeed }
func (stubFeed) Fetch(_ context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	return nil, nil
}

// registerStubs builds a fresh *fetcher.Registry populated with two
// stub providers (one blog, one feed). Tests receive both the
// registry and the chosen type ids — there is no global state.
func registerStubs(t *testing.T) (reg *fetcher.Registry, blogType, feedType string) {
	t.Helper()
	blogType = "stub_blog_" + id.NewULID()
	feedType = "stub_feed_" + id.NewULID()
	reg = fetcher.NewRegistry()
	gt.NoError(t, reg.Register(blogType, stubBlog{}))
	gt.NoError(t, reg.Register(feedType, stubFeed{}))
	return
}

func writeTOML(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	gt.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestLoad_MissingFileReturnsEmptyCatalog(t *testing.T) {
	cat, err := source_catalog.Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	gt.NoError(t, err)
	gt.A(t, cat.Sources).Length(0)
}

func TestLoad_ValidEntries(t *testing.T) {
	reg, blog, feed := registerStubs(t)
	dir := t.TempDir()
	path := writeTOML(t, dir, "config.toml", `
[[source]]
id = "blog-1"
name = "Blog One"
type = "`+blog+`"
url = "https://example.com/feed"
interval = "1h"

[[source]]
id = "feed-1"
name = "Feed One"
type = "`+feed+`"
interval = "30m"
`)

	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	gt.A(t, cat.Sources).Length(2)

	problems := source_catalog.Validate(cat, reg)
	gt.A(t, problems).Length(0)

	// Validate fills in Kind.
	got, ok := cat.ByID(types.SourceID("blog-1"))
	gt.True(t, ok)
	gt.Equal(t, got.Kind, types.KindBlog)
}

func TestValidate_DuplicateID(t *testing.T) {
	reg, blog, _ := registerStubs(t)
	dir := t.TempDir()
	path := writeTOML(t, dir, "dup.toml", `
[[source]]
id = "x"
name = "A"
type = "`+blog+`"
url = "https://a.example/feed"
interval = "1h"

[[source]]
id = "x"
name = "B"
type = "`+blog+`"
url = "https://b.example/feed"
interval = "1h"
`)
	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	problems := source_catalog.Validate(cat, reg)

	gt.True(t, hasProblem(problems, "id", "duplicate id"))
}

func TestValidate_UnknownProviderSuggestsDidYouMean(t *testing.T) {
	reg, blog, _ := registerStubs(t)
	dir := t.TempDir()
	path := writeTOML(t, dir, "unknown.toml", `
[[source]]
id = "y"
name = "Y"
type = "`+blog+`X"
url = "https://y.example/feed"
interval = "1h"
`)
	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	problems := source_catalog.Validate(cat, reg)
	gt.A(t, problems).Longer(0)
	// At least one of the problems should mention the registered blog id
	// as the suggestion.
	var sawHint bool
	for _, p := range problems {
		if p.Field == "type" && containsBoth(p.Hint, "did you mean", blog) {
			sawHint = true
			break
		}
	}
	gt.True(t, sawHint)
}

func TestValidate_IntervalBelowMinimum(t *testing.T) {
	reg, blog, _ := registerStubs(t)
	dir := t.TempDir()
	path := writeTOML(t, dir, "fast.toml", `
[[source]]
id = "z"
name = "Z"
type = "`+blog+`"
url = "https://z.example/feed"
interval = "1m"
`)
	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	problems := source_catalog.Validate(cat, reg)
	gt.True(t, hasProblem(problems, "interval", ""))
}

func TestValidate_BlogRequiresURL(t *testing.T) {
	reg, blog, _ := registerStubs(t)
	dir := t.TempDir()
	path := writeTOML(t, dir, "no-url.toml", `
[[source]]
id = "no-url"
name = "no-url"
type = "`+blog+`"
interval = "1h"
`)
	cat, err := source_catalog.Load(path)
	gt.NoError(t, err)
	problems := source_catalog.Validate(cat, reg)
	gt.True(t, hasProblem(problems, "url", "required"))
}

func TestLoad_BadTOMLIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, "bad.toml", "[[source\nid = ")
	_, err := source_catalog.Load(path)
	gt.Error(t, err)
}

// ---- helpers ----

func hasProblem(ps []source_catalog.Problem, field, substr string) bool {
	for _, p := range ps {
		if p.Field == field {
			if substr == "" || contains(p.Message, substr) {
				return true
			}
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func containsBoth(s, a, b string) bool { return contains(s, a) && contains(s, b) }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
