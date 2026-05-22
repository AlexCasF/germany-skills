# Destatis manifest

## Original files moved

| Original path | New path |
| --- | --- |
| `agent/skills/api-destatis/SKILL.md` | `skills/destatis/SKILL.md` |
| `agent/skills/api-destatis/references/notes.md` | `skills/destatis/references/notes.md` |
| `agent/skills/api-destatis/references/openapi.yaml` | `skills/destatis/references/openapi.yaml` |
| `cli/cmd/destatisctl/main.go` | `skills/destatis/go/v1/main.go` |
| `agent/bin/destatisctl.exe` | `skills/destatis/bin/destatisctl-legacy.exe` |

## Implementations

| Implementation | Path | Status |
| --- | --- | --- |
| Legacy Go | `go/v1/main.go` | Preserved unchanged for comparison. |
| Go 2.0 | `go/v2/main.go` | Built and tested. |
| Go 2.0 binary | `bin/destatisctl-2.0.exe` | Built locally with Go 1.26.2. |
| Python | `python/destatisctl.py` | Created and tested with Python 3.14.3. |
| TypeScript/Node.js | `typescript/src/index.ts` | Created, built, and tested with Node 24.13.1. |

## Applied lessons

This refactor applies the CLI research lessons from:

- `docs/live-research-lessons.md`
- `docs/cli-agent-friendliness-plan.md`
- `docs/cli-tool-refactor-execution-plan.md`

Key changes:

- Preserved the original endpoint command paths: `catalogue statistics`, `catalogue tables`, `catalogue variables`, `metadata table`, `metadata timeseries`, `data table`, `data timeseries`, and `find search`.
- Switched live requests to form `POST` in 2.0 because current live testing returned the JavaScript web app for `GET`, while `POST` works for `logincheck` and `find/find`.
- Added `DESTATIS_USERNAME` and `DESTATIS_PASSWORD` environment fallback.
- Added `GAST/GAST` discovery fallback when no credentials are configured.
- Added credential redaction for normalized output and structured errors.
- Added `doctor` for credential source, endpoint health, license, docs, and fair-use context.
- Added `search` as a compact alias for `find search`.
- Added `table source` for official source/citation metadata without requiring a data call.
- Added `table dossier`, `table sample`, `timeseries dossier`, and `variables explain` with controlled `partial` results when guest auth lacks access.
- Added safe default `pagelength` and `--limit` behavior for discovery.

## Research findings

- GENESIS-Online is the Destatis database for configurable official statistical tables.
- The Destatis Open Data page says GENESIS-Online is generally free and usable without registration under Data Licence Germany attribution 2.0.
- The Destatis Open Data page says the GENESIS webservice can integrate the GENESIS data stock into automated processes and currently offers a RESTful/JSON interface with many methods.
- The English Open Data page also lists SOAP/XML and RESTful/JSON as API interfaces.
- The official GENESIS contact page directs technical API questions to the GENESIS-Online user service.
- The local OpenAPI spec describes a broad `genesisWS/rest/2020` surface with catalogue, metadata, data, find, helloworld, and profile endpoints.
- Live testing on 2026-05-18 showed `POST /helloworld/logincheck` succeeds with `GAST/GAST`.
- Live testing on 2026-05-18 showed `POST /find/find` succeeds with `GAST/GAST`.
- Live testing on 2026-05-18 showed several catalogue/metadata/data endpoints returned `401 Unauthorized` with `GAST/GAST`; use personal credentials for those.
- No exact request-per-minute rate limit was found in official Destatis docs reviewed.
- `logincheck` returned a message warning that if there are more than 3 parallel requests, requests running longer than 15 minutes are terminated. The CLI therefore warns against broad parallel calls.

Primary references:

- `https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html`
- `https://www.destatis.de/EN/Service/OpenData/api-webservice.html`
- `https://www.destatis.de/DE/Service/Kontakt/Genesis/Servicekontakt-GENESIS.html`
- `https://www-genesis.destatis.de/datenbank/online`
- `https://github.com/bundesAPI/destatis-api`

## Test status

All three 2.0 implementations passed the 10-case test plan:

- `tests/test-plan.md`
- `tests/go-2.0-results.md`
- `tests/python-results.md`
- `tests/typescript-node-results.md`

Test fixture:

- Search term: `Arbeitslose`
- Table code: `12211-0900`
- Auth mode: no personal credentials configured, `GAST/GAST` fallback

## Final SKILL.md decisions

The finalized skill prioritizes:

- official statistics discovery before data retrieval
- env credentials and redaction
- `search -> table source -> table dossier -> metadata/sample` workflow
- strong caveats around guest-vs-personal credential access
- table/statistic code preservation
- small `pagelength` defaults and no broad data pulls

## Runtime note

The old runtime location `agent/bin/destatisctl.exe` has been moved into this
skill folder as the legacy binary. Deployment/runtime code that still expects
the old path should either:

- point to `skills/destatis/bin/destatisctl-2.0.exe`
- copy/symlink the selected binary into the runtime image path
- call the Python or Node.js fallback explicitly

Do this integration step when the runtime wiring is refactored; it was
intentionally not done as part of this isolated tool pass.
