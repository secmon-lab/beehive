# Sources

Source definitions live in a single TOML file. The default path is `./config.toml` and can be overridden with `--config <path>` / `-c <path>` or the `BEEHIVE_CONFIG` environment variable. There is no API or UI to create, edit, or delete a Source — change the TOML, run `beehive validate`, and redeploy. The OSS repository ships a sample at `examples/config.toml`; copy it as a starting point.

## Schema

```toml
[[source]]
id       = "trendmicro-research"      # stable human-readable id; required
name     = "Trend Micro Research"     # display name; required
type     = "rss"                      # provider id; required
url      = "https://www.trendmicro.com/en_us/research.rss"  # blog kind only
interval = "1h"                       # Go duration; required, minimum 5m
disabled = false                      # optional; default false (enabled)
```

The same `type` field decides whether the source is a blog or a feed — the provider's `Kind()` method drives the runtime pipeline.

## Built-in providers

| `type`                          | Kind  | Notes                                                         |
|---------------------------------|-------|---------------------------------------------------------------|
| `rss`                           | blog  | gofeed auto-detects RSS, Atom, and JSON Feed.                |
| `abuseipdb_blacklist`           | feed  | Requires `BEEHIVE_PROVIDER_ABUSEIPDB_API_KEY`.               |
| `cinsscore_badguys`             | feed  | Plain-text list of malicious IPs.                            |
| `feodotracker_ipblocklist`      | feed  | Plain-text list of IPs from abuse.ch.                        |
| `urlhaus_recent`                | feed  | Recent malicious URLs from abuse.ch (CSV).                   |
| `threatfox_recent`              | feed  | Recent indicators from abuse.ch (JSON, mixed types).         |

Adding a new provider is a small Go change — see CLAUDE.md §4.

## Operator overrides

If you need to pause a source without editing the TOML, set the override via the API:

```bash
curl -X PUT http://localhost:8080/api/v1/sources/<source-id>/enabled_override \
    -H 'content-type: application/json' \
    -d '{"value":"force_off"}'
```

`force_off` disables the source even if TOML says enabled; `force_on` enables it even if TOML says `disabled = true`; `none` returns to the TOML default. Overrides live in Firestore (`states/{SourceID}.EnabledOverride`), survive container restarts, and are visible on the Sources UI page.

## Orphans

If you remove a source from TOML, its state document stays behind in Firestore (the historical record of past Runs and Articles still references it). beehive logs a `WARN` at startup listing every orphan SourceID so you can clean them up out of band.
