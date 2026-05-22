# HTTP API

All endpoints live under `/api/v1/`. The canonical OpenAPI spec is in [`openapi.yaml`](../openapi.yaml).

## Endpoints

| Method | Path                                            | Purpose                                                          |
|--------|-------------------------------------------------|------------------------------------------------------------------|
| GET    | `/api/v1/health`                                | Liveness probe.                                                  |
| GET    | `/api/v1/sources`                               | List Sources merged with their dynamic state.                    |
| GET    | `/api/v1/sources/{id}`                          | One Source.                                                      |
| PUT    | `/api/v1/sources/{id}/enabled_override`         | Set the operator override (`none` / `force_off` / `force_on`).   |
| POST   | `/api/v1/sources/{id}/fetch`                    | Force a single-source fetch (ignores the interval).              |
| POST   | `/api/v1/fetch`                                 | Fetch every due Source. **This is the scheduler entry point.**   |
| GET    | `/api/v1/runs/{id}`                             | Run detail.                                                      |
| GET    | `/api/v1/iocs/lookup?type=ipv4&value=1.2.3.4`   | Exact-match IoC lookup. Returns 404 when missing.                |

## Examples

### Trigger a fetch

```bash
curl -X POST http://localhost:8080/api/v1/fetch
# {"runId":"run-01HX...","total":12,"triggered":3,"skipped":9,"failed":0,"durationMs":1840,"status":"success"}
```

### Look up one IoC

```bash
curl 'http://localhost:8080/api/v1/iocs/lookup?type=domain&value=evil.example'
# {"id":"f1c0...","type":"domain","value":"evil.example","raw":"evil[.]example",...}
```

### Set an enabled override

```bash
curl -X PUT http://localhost:8080/api/v1/sources/krebsonsecurity/enabled_override \
    -H 'content-type: application/json' \
    -d '{"value":"force_off"}'
# 204 No Content
```

## Errors

All error responses use a small RFC 7807-flavoured envelope:

```json
{ "code": "not_found", "message": "...", "details": { "key": "value" } }
```

`code` is one of `not_found`, `invalid_input`, `conflict`, `busy`, `internal_error`. HTTP status codes map directly (404 / 400 / 409 / 503 / 500). Authn / authz mapping (`401` / `403`) is intentionally absent in MVP — the upcoming Rego layer will add it.
