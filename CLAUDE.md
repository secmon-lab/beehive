# beehive

A pull-driven crawler that extracts IoCs from security blogs and IoC feeds. Designed for Cloud Run zero scale: an external scheduler hits `POST /api/v1/fetch` and beehive runs all due sources synchronously in-request.

This file holds project-specific rules for Claude Code sessions. Always read it together with `~/.claude/CLAUDE.md` and `~/.claude/rules/{go,frontend,testing,completion-check}.md`.

> All documents that land in this repository (CLAUDE.md, README.md, code comments, in-repo docs, etc.) MUST be written in English. The design spec under `.spec/` is the only exception and may stay in the working language because it is not committed.

## 0. Top priority: keep things simple

Before adding a feature, abstraction or config knob, ask "do we actually need this now?" The MVP intentionally does NOT ship:

- HTTP authn / authz (a separate Rego-based authz layer is planned and lives in a later spec).
- A BigQuery exporter (the Firestore → BigQuery path is delegated to GCP-managed plumbing).
- Bulk / streaming / search APIs for IoCs (exact-match `lookupIoC` only).
- CRUD APIs or UI for Sources (TOML IaC only).
- Article browsing, Dashboard, IoC detail pages (the frontend has 3 pages: Sources / IoCs / Runs).
- Per-source fetch timeout / max body / User-Agent knobs (hardcoded defaults are fine).
- Generic `parser_config` (no JSONPath / CSS selectors exposed to users). Each Source is implemented as a dedicated Go provider; users only pick a type.
- Cron-style schedules (each Source has an `interval` Duration only).
- Long-running background goroutines (Cloud Run zero scale — request-scoped goroutines only, see §6).

When in doubt, come back to this section.

## 1. Architecture overview

DDD layering (deps flow downward only):

```
cli  →  usecase  →  service / repository / domain (interfaces, model, types)
controller/http  →  usecase
```

Notable directories:

- `pkg/domain/{model,interfaces,types}` — entities, ports, ID and enum types.
- `pkg/repository/{firestore,memory}` — the two Repository implementations.
- `pkg/service/{source_catalog,fetcher,extractor,normalize}` — domain services.
- `pkg/usecase/` — fetch / extract / ioc_lookup / source_query / bootstrap.
- `pkg/controller/http/` — chi router (will host oapi-codegen output later).
- `pkg/cli/` — `serve` / `fetch` / `validate` subcommands.
- `pkg/utils/{logging,errutil,safe,async,id}` — cross-cutting helpers.
- `frontend/` — Vite + React 19 + TypeScript (3 pages).
- `examples/sources/example.toml` — OSS-bundled sample.
- `config/sources/*.toml` — operator-owned TOML. The OSS repo only ships `.gitkeep`.

## 2. Dependencies

Go:

- HTTP router: `github.com/go-chi/chi/v5`
- CLI: `github.com/urfave/cli/v3`
- Firestore: `cloud.google.com/go/firestore`
- OpenAPI gen: `github.com/oapi-codegen/oapi-codegen/v2`
- TOML: `github.com/pelletier/go-toml/v2`
- Feeds: `github.com/mmcdole/gofeed`
- Article body extraction: `github.com/go-shiori/go-readability`
- LLM: `github.com/m-mizutani/gollem`
- Logger: `github.com/m-mizutani/clog`
- Errors: `github.com/m-mizutani/goerr/v2`
- Parallelism: `golang.org/x/sync/errgroup`
- Test helpers: `github.com/m-mizutani/gt`

Frontend:

- Vite 6 + TypeScript 5
- React 19 + react-router-dom v7
- `openapi-fetch` + `openapi-typescript`
- TanStack Query v5
- shadcn/ui (Tailwind v4 + Radix)
- lucide-react

Distribution: a single Go binary with the frontend embedded via `//go:embed all:dist`, deployed to Cloud Run.

## 3. Firestore schema

| collection                | doc ID                                                    | purpose                                                       |
| ------------------------- | --------------------------------------------------------- | ------------------------------------------------------------- |
| `states`                  | `SourceID`                                                | Dynamic state per Source (catalog lives in memory).           |
| `articles`                | `ArticleID` (ULID)                                        | Blog-kind only.                                               |
| `iocs`                    | `sha256(Type + "\t" + Value)` (64 hex chars)              | Deduplicated indicators.                                      |
| `iocs/{ID}/refs`          | `sha256(SourceID + ":" + (ArticleID or RunID))`           | Occurrence sub-collection.                                    |
| `runs`                    | `RunID` (ULID)                                            | One document per `POST /api/v1/fetch` invocation.             |
| `locks/{Kind}/entries`    | `TargetID`                                                | Generic heartbeat-renewed locks.                              |

### Repository implementation policy (non-negotiable)

1. **No `firestore:"..."` struct tags.** Exported Go field names map 1:1 to Firestore field names. Use explicit convert functions when shapes differ.
   - Exception: LLM I/O structs (e.g. `IoCSeed`) MAY use `json:"..."` tags because they double as gollem structured-response schemas. Such structs MUST NOT be persisted directly.
2. **Field names follow Go conventions in PascalCase** (initialisms stay uppercase: `ID`, `URL`, `IoC`). JSON / OpenAPI may have their own conventions.
3. Complex conversions go into explicit `*FromDoc` / `*ToDoc` helpers under `pkg/repository/firestore/`.

### Write-cost discipline

Firestore writes cost more than reads. Treat every Update as suspect:

- Skip identical-value writes. Use targeted `Update([]firestore.Update{...})`, never `Set(MergeAll)` to overwrite everything.
- Use `firestore.Increment(n)` for counters to avoid races.
- `iocs/{ID}`: on re-occurrence, update only `LastSeenAt`; `Raw` is set once on creation and never overwritten.
- `iocs/{ID}/refs/{RefID}`: existing ref → no write (the natural key already dedupes the occurrence).
- `articles/{ID}`: look up by URL; if `ContentHash` matches, do nothing (read-only path).
- `states/{SourceID}`: if all fields are unchanged, skip the write.
- `locks/{Kind}/entries/{TargetID}`: heartbeat updates only timestamp fields; `Release` is a single Delete.

## 4. Adding a Source provider

1. Create `pkg/service/fetcher/{blog,feed}/<name>.go`.
2. Register the provider in `init()` via `fetcher.Register(typeID, factory)`.
3. If the provider needs credentials, read them from environment variables named `BEEHIVE_PROVIDER_<UPPER_SNAKE>_*`.
4. Implement `interfaces.BlogProvider` (returns `[]*FetchedArticle`) or `interfaces.FeedProvider` (returns `[]*IoCSeed`).
5. For blog providers, fetch the entry URL with `pkg/service/fetcher/http_client.go` and run the HTML through `go-shiori/go-readability` to strip boilerplate before returning the body.
6. Add `*_test.go` with frozen-fixture parsing tests.

`IoCSeed` is the pre-normalization intermediate type returned by providers. It carries `json` tags so it doubles as the gollem structured-response schema; never persist it directly.

## 5. Trigger model

- **No long-running scheduler.** `POST /api/v1/fetch` is the entry point; an external scheduler (Cloud Scheduler) hits it on a cadence.
- Inside the handler, walk the in-memory catalog and filter by `State.LastFetchedAt + Interval < now()`. Eligible sources are processed synchronously via `errgroup.SetLimit(8)`.
- One request = one `runs/{RunID}` document. Per-source results accumulate in `Sources[]`.
- `triggerSourceFetch` (per-source endpoint) ignores the interval and forces a fetch.
- Do NOT spawn goroutines that outlive the request. Cloud Run may kill the container as soon as the response is sent.

## 6. Locks (heartbeat)

- `LockManager.Acquire(ctx, kind, targetID)` returns an `*Lock`. A heartbeat goroutine bumps `ExpiresAt = now + 20s` every 20 seconds.
- `lock.Release(ctx)` stops the heartbeat and deletes the Firestore document. Always `defer` it.
- On instance death, `ExpiresAt` lapses in ~20 seconds and another instance may take over. Fetches are idempotent so a redundant run is acceptable.

## 7. LLM (gollem) configuration

- Env vars: `BEEHIVE_LLM_PROVIDER`, `BEEHIVE_LLM_MODEL`, `BEEHIVE_LLM_API_KEY`, `BEEHIVE_LLM_ARGS`.
- `BEEHIVE_LLM_ARGS` is a comma-separated `key=value` list (e.g. for Vertex AI: `project_id=foo,location=us-central1`). Always use `project_id` (never bare `project`) for the GCP project key.
- Only blog-kind sources invoke the LLM. Use gollem with a JSON-Schema-constrained response so `IoCSeed[]` parses cleanly.
- After the LLM returns, normalize in Go: `netip.ParseAddr` for IPs, `idna.ToASCII` + lowercase for domains, `url.Parse` for URLs, length check for hashes.

## 8. CLI

- `beehive serve` — boots the HTTP server only (no scheduler).
- `beehive fetch [source-id]` — ad-hoc synchronous fetch from the CLI (not via HTTP).
- `beehive validate` — validates the whole config (TOML sources, provider registry, required env vars, LLM config) and prints a colourised, aggregated report. Required to pass in CI.

## 9. Development workflow

- Generate: `task gen` (`oapi-codegen` + `openapi-typescript`).
- Build: `task build` (frontend first, then Go binary with embedded dist).
- Tests: `task test` (`go test ./...` + `pnpm test` + `pnpm lint`).
- Run locally: `task run`.
- Firestore-backed tests: set `TEST_FIRESTORE_PROJECT_ID` and `TEST_FIRESTORE_DATABASE_ID` (typically pointing at a local emulator). Tests skip automatically when these are unset.

## 10. Testing (mandatory)

**Every feature ships with tests in the same PR.** No exceptions. CI runs `go test ./...` on every push.

Conventions:

- **File-level 1:1**: `xyz.go` pairs with `xyz_test.go`. Do not invent `xyz_e2e_test.go` or similar suffixes.
  - Exception: pure declaration files (`pkg/domain/interfaces/*.go` interface definitions) need no test file.
  - Exception: a single `_test.go` may cover a family of related type / enum / model files in the same package (e.g. `pkg/domain/types/types_test.go` covers all of `id.go`, `source_kind.go`, `ioc_type.go`, `run_status.go`, `enabled_override.go`) — provided every exported function or branch is reached.
- **External test packages**: write tests as `package {name}_test`. Use `export_test.go` if you need to surface internals.
- **Test helpers**: `github.com/m-mizutani/gt` (`gt.NoError(t, err)`, `gt.Equal(t, got, want)`, `gt.A(t, slice).Length(N)`, `gt.Value(t, v).Equal(...)`).
- **No hardcoded IDs**: derive from `time.Now().UnixNano()` or `pkg/utils/id` ULIDs to avoid collisions under parallel runs.
- **`t.Skip()` only for env vars**: skip on missing `TEST_FIRESTORE_*`, never for unimplemented features (implement them or delete the test).

### Repository: shared test harness (Memory + Firestore)

Repository tests cover both backends from a **single test body** so that the Memory and Firestore implementations cannot drift. The pattern (taken from `secmon-lab/warren` and `secmon-lab/hecatoncheires`):

```go
// pkg/repository/repository_test.go
package repository_test

func TestSourceState(t *testing.T) {
    testFn := func(t *testing.T, repo interfaces.Repository) {
        ctx := t.Context()
        // ... assertions ...
    }

    t.Run("Memory", func(t *testing.T) {
        testFn(t, memory.New())
    })

    t.Run("Firestore", func(t *testing.T) {
        testFn(t, newFirestoreRepo(t)) // skips when TEST_FIRESTORE_* unset
    })
}
```

Rules for this harness:

- All repository test bodies accept `interfaces.Repository` (not the concrete type), so the same assertions run against both implementations.
- The Firestore entry helper checks `TEST_FIRESTORE_PROJECT_ID` and `TEST_FIRESTORE_DATABASE_ID`; it calls `t.Skip("set TEST_FIRESTORE_* to run")` if either is missing.
- Use random IDs (ULID / `time.Now().UnixNano()`) so parallel runs against a shared Firestore database do not collide.
- Always assert all fields you care about after writes — do not stop at "row exists". Verify timestamps with `< time.Second` tolerance for Firestore round-trip precision.

### Integration & E2E layout

- `tests/integration/` — Go-level integration tests that boot `httptest.NewServer` against the in-memory backend. Touches HTTP, repository and catalog together; runs in normal `go test ./...`.
- `frontend/e2e/` — browser-side end-to-end suite (Playwright). Added together with the React frontend in a later PR.

## 11. CLAUDE.global.md / rules/ reminders

- Errors: `goerr.Wrap`, `errors.Is/As`. Never inspect message text.
- **Every non-fatal error MUST go through `errutil.Handle`** — direct `logger.Warn("...", "error", err)` / `logger.Error(...)` / `_ = someCall()` for errors are forbidden. `errutil.Handle` is the single place where structured attributes (`goerr.Values`, tags, stack info) are extracted and forwarded to the logger / Sentry; bypassing it loses that metadata.
  - The only acceptable exception is an error that is **truly safe to ignore** (e.g. a cleanup whose only failure mode is "already gone"). Such cases MUST carry a comment explaining why. When in doubt, route through `errutil.Handle`.
- Logger: always obtain the context-scoped logger with `logging.From(ctx)`.
- Resources: nil-safe `safe.Close`; `_ = x.Close()` is forbidden.
- Async: launch goroutines through `async.Dispatch` for panic recovery and logger propagation. No goroutine outlives its originating request (see §0 / §5).
- Frontend: i18n is mandatory; design tokens (`tokens.css`) only; guard IME composition (`event.nativeEvent.isComposing`) on Enter handlers.
