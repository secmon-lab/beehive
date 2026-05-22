package source_catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// MinInterval is the floor enforced by validation. Cron-style "every
// second" abuse is not the goal — the smallest sensible interval for
// upstream-friendly polling is around a few minutes.
const MinInterval = 5 * time.Minute

// Problem is a single, human-readable validation finding. Multiple
// problems are aggregated by Validate so `beehive validate` can show
// them all at once.
type Problem struct {
	File    string
	Index   int
	ID      string
	Field   string
	Message string
	Hint    string
}

func (p Problem) Error() string {
	loc := "?"
	if p.File != "" {
		loc = fmt.Sprintf("%s[%d]", p.File, p.Index)
	}
	base := fmt.Sprintf("%s id=%q field=%s: %s", loc, p.ID, p.Field, p.Message)
	if p.Hint != "" {
		return base + " — " + p.Hint
	}
	return base
}

// Validate checks the catalog against the provider registry and global
// invariants. The pre-existing model.Source values are mutated to set
// `Kind` based on the resolved provider (blog vs feed). Returns the
// aggregated list of problems and the first error if any.
//
// `reg` MUST be non-nil — the registry is built once at startup
// (pkg/service/providers.All) and threaded through every caller. A nil
// registry indicates a wiring bug and is treated as such.
func Validate(cat *Catalog, reg *fetcher.Registry) []Problem {
	var problems []Problem
	seen := map[types.SourceID]LoadedSource{}

	for _, ls := range cat.Loaded {
		src := ls.Source

		if src.ID == "" {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index,
				Field: "id", Message: "required",
			})
			continue
		}
		if dup, ok := seen[src.ID]; ok {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field:   "id",
				Message: "duplicate id",
				Hint: fmt.Sprintf("already defined in %s[%d]",
					dup.File, dup.Index),
			})
			continue
		}
		seen[src.ID] = ls

		if src.Name == "" {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field: "name", Message: "required",
			})
		}
		if src.Type == "" {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field: "type", Message: "required",
			})
			continue
		}

		prov, err := reg.Resolve(string(src.Type))
		if err != nil {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field:   "type",
				Message: fmt.Sprintf("unknown provider %q", src.Type),
				Hint:    didYouMean(string(src.Type), reg.Types()),
			})
			continue
		}
		src.Kind = prov.Kind()
		if !src.Kind.Valid() {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field:   "type",
				Message: fmt.Sprintf("provider %q reported invalid kind %q", src.Type, src.Kind),
			})
		}
		if src.Kind == types.KindBlog && src.URL == "" {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field: "url", Message: "required for blog-kind providers",
			})
		}
		if src.Interval < MinInterval {
			problems = append(problems, Problem{
				File: ls.File, Index: ls.Index, ID: string(src.ID),
				Field:   "interval",
				Message: fmt.Sprintf("interval %s is below the minimum %s", src.Interval, MinInterval),
				Hint:    "use at least 5m",
			})
		}

		// Check kind selector — every BlogProvider / FeedProvider must
		// implement the matching interface.
		switch src.Kind {
		case types.KindBlog:
			if _, ok := prov.(interfaces.BlogProvider); !ok {
				problems = append(problems, Problem{
					File: ls.File, Index: ls.Index, ID: string(src.ID),
					Field:   "type",
					Message: fmt.Sprintf("provider %q reports blog kind but does not implement BlogProvider", src.Type),
				})
			}
		case types.KindFeed:
			if _, ok := prov.(interfaces.FeedProvider); !ok {
				problems = append(problems, Problem{
					File: ls.File, Index: ls.Index, ID: string(src.ID),
					Field:   "type",
					Message: fmt.Sprintf("provider %q reports feed kind but does not implement FeedProvider", src.Type),
				})
			}
		}
	}

	return problems
}

// AsError turns a problem list into a single tagged goerr suitable for
// returning up the stack. Returns nil when the list is empty.
//
// The error message is intentionally multiline — `err.Error()` on the
// return value lists every individual problem, one per line, so a
// caller that just prints `%s` already shows the operator what is
// wrong. Each problem is also exposed as a goerr.V (`problem.0`,
// `problem.1`, ...) so structured loggers can render the same data as
// key/value attributes.
func AsError(problems []Problem) error {
	if len(problems) == 0 {
		return nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d source config problem(s):", len(problems))
	for _, p := range problems {
		fmt.Fprintf(&sb, "\n  - %s", p.Error())
	}
	values := make([]goerr.Option, 0, len(problems)+1)
	values = append(values, goerr.T(errutil.TagInvalidInput))
	for i, p := range problems {
		values = append(values, goerr.V(fmt.Sprintf("problem.%d", i), p.Error()))
	}
	return goerr.New(sb.String(), values...)
}

// didYouMean returns a "did you mean foo?" hint when one candidate is
// reasonably close (edit distance ≤ 3). Returns empty string otherwise.
func didYouMean(input string, candidates []string) string {
	best := ""
	bestDist := 4
	for _, c := range candidates {
		d := editDistance(input, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf("did you mean %q?", best)
}

func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	return min(a, min(b, c))
}
