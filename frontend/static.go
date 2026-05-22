// Package frontend embeds the built React assets into the Go binary.
// The dist/ directory is populated by `pnpm build` before `go build`
// runs (Dockerfile stages the order). When dist/ is missing — typical
// during local Go-only development — the embed silently produces an
// empty filesystem.
package frontend

import "embed"

// `all:` keeps dot-prefixed files (notably `.gitkeep`) so a clean
// checkout that has not yet run `pnpm build` still produces a non-empty
// embedded filesystem.
//
//go:embed all:dist
var StaticFiles embed.FS
