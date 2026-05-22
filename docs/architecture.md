# Architecture

beehive is a pull-driven IoC extractor: an external scheduler (Cloud Scheduler in production) hits `POST /api/v1/fetch` on a cadence, and beehive runs every due Source synchronously inside that request. There is no internal cron, no background goroutine that survives the response — the design assumes the process is suspended between fetches (Cloud Run zero scale).

## Layers

```
cli            →  usecase  →  service / repository / domain
controller/http →  usecase
```

- `pkg/domain/{model,interfaces,types}` — entities, ports, ID and enum types. No imports from outside the package; this is the contract that everything else binds to.
- `pkg/repository/{firestore,memory}` — two implementations of the `interfaces.Repository` contract. The shared harness in `pkg/repository/repository_test.go` runs every test against both.
- `pkg/service/` — domain services: source catalog (TOML loader), fetchers (per-provider), the LLM extractor, the IoC normalizer, the defang helper.
- `pkg/usecase/` — request-scoped flows: `bootstrap`, `fetch_all`, `fetch`, `ioc_lookup`, `source_query`. The HTTP handlers and the CLI subcommands both consume these.
- `pkg/controller/http/` — chi router that exposes the OpenAPI surface.
- `pkg/cli/` — `serve`, `fetch`, `validate` subcommands.
- `pkg/utils/` — cross-cutting helpers (`logging`, `errutil`, `safe`, `async`, `id`).
- `frontend/` — Vite + React 19, served from `//go:embed all:dist`.

## Source kinds

| Kind   | Pipeline                                                                                        |
|--------|-------------------------------------------------------------------------------------------------|
| `blog` | Provider returns `[]*FetchedArticle`. UseCase persists each Article, runs the LLM extractor on the body, normalises seeds, then upserts IoCs. |
| `feed` | Provider returns `[]*IoCSeed` directly. No LLM is invoked.                                       |

The TOML `type` field selects the provider implementation. Adding a new Source = adding a new file under `pkg/service/fetcher/{blog,feed}/`.

## Storage

| Collection                | Purpose                                                       |
|---------------------------|---------------------------------------------------------------|
| `states`                  | Per-Source dynamic state (the TOML catalog lives in memory). |
| `articles`                | Blog articles (`blog` kind only).                            |
| `iocs`                    | Deduplicated indicators, ID = full sha256 of `(Type, Value)`. |
| `iocs/{ID}/refs`          | Occurrence records, ID = sha256 of `(SourceID, ArticleID or RunID)`. |
| `runs`                    | One document per `POST /api/v1/fetch` invocation.            |
| `locks/{Kind}/entries`    | Heartbeat-renewed mutual exclusion locks.                    |

Write-cost discipline is in CLAUDE.md §3. The short version: skip identical-value writes, use `firestore.Increment` for counters, and never `Set(MergeAll)` to overwrite a whole document.

## Trigger model

```
Cloud Scheduler ─► POST /api/v1/fetch ─► fetchAll() ─► errgroup over due sources
```

`fetchAll` reads the in-memory catalog, looks up each Source's state, and queues the due ones into an `errgroup` with a concurrency cap of 8. Skipped sources still get a `RunSource` entry (with `skip_reason`) so the operator can see them in the audit.

For the lock lifecycle, refresh interval, and override semantics see CLAUDE.md sections 5 and 6.
