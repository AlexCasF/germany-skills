# Bundestag Lobbyregister manifest

## Original files moved

| Original path | New path |
| --- | --- |
| `agent/skills/api-bundestag-lobbyregister/SKILL.md` | `skills/bundestag-lobbyregister/SKILL.md` |
| `agent/skills/api-bundestag-lobbyregister/references/notes.md` | `skills/bundestag-lobbyregister/references/notes.md` |
| `agent/skills/api-bundestag-lobbyregister/references/openapi.yaml` | `skills/bundestag-lobbyregister/references/openapi.yaml` |
| `cli/cmd/lobbyregisterctl/main.go` | `skills/bundestag-lobbyregister/go/v1/main.go` |
| `agent/bin/lobbyregisterctl.exe` | `skills/bundestag-lobbyregister/bin/lobbyregisterctl-legacy.exe` |

## Implementations

| Implementation | Path | Status |
| --- | --- | --- |
| Legacy Go | `go/v1/main.go` | Preserved unchanged for comparison. |
| Go 2.0 | `go/v2/main.go` | Built and tested. |
| Go 2.0 binary | `bin/lobbyregisterctl-2.0.exe` | Built locally with Go 1.26.2. |
| Python | `python/lobbyregisterctl.py` | Created and tested with Python 3.14.3. |
| TypeScript/Node.js | `typescript/src/index.ts` | Created, built, and tested with Node 24.13.1. |

## Applied lessons

This refactor applies the CLI research lessons from:

- `docs/live-research-lessons.md`
- `docs/cli-agent-friendliness-plan.md`
- `docs/cli-tool-refactor-execution-plan.md`

Key changes:

- Added `doctor` for auth, endpoint health, docs, fair-use, and rate-limit context.
- Added `statistics` for compact register-wide metrics.
- Added safe V2 `search` with default small limits and compact summaries.
- Added `entry get` for precise V2 register-entry lookup.
- Added `entry source` for official API, public page, PDF, annual-report, and statement URLs.
- Added `entry dossier` with identity, activity, finance, regulatory projects, statements, sources, warnings, and next actions.
- Added `financial summary` to normalize expense ranges, funding sources, donations, membership fees, allowances, annual reports, and caveats.
- Added `statements list` with optional `--grep` over embedded statement text.
- Preserved the old thin wrapper as legacy source/binary and exposed a `v1 search` compatibility command in 2.0.
- Added redaction for API keys in normalized output and structured errors.

## Research findings

- The current official deployment is API V2 under `https://api.lobbyregister.bundestag.de/rest/v2`.
- The official Open Data/API page says all public register contents can be queried through the API.
- The official Open Data/API page says an API key is necessary and that individual durable keys can be requested by email.
- The official Open Data/API page says API V2 went live with release R2.21 on 2025-06-23 and replaced API V1.
- The V2 OpenAPI document exposes `GET /registerentries`, `GET /registerentries/{registerNumber}`, `GET /registerentries/{registerNumber}/{version}`, and `GET /statistics/registerentries`.
- The V2 OpenAPI document supports API-key auth by `Authorization: ApiKey ...` header or `apikey` query parameter.
- No exact request-per-minute rate limit was found in the official docs reviewed.
- The upstream search endpoint returns large full-detail records, so compact local slicing is important.

Primary references:

- `https://www.lobbyregister.bundestag.de/informationen-und-hilfe/open-data-1049716`
- `https://api.lobbyregister.bundestag.de/rest/v2/swagger-ui/`
- `https://api.lobbyregister.bundestag.de/rest/v2/R2.21-de.yaml`
- `https://github.com/bundesAPI/bundestag-lobbyregister-api`

## Test status

All three 2.0 implementations passed the 10-case test plan:

- `tests/test-plan.md`
- `tests/go-2.0-results.md`
- `tests/python-results.md`
- `tests/typescript-node-results.md`

Additional smoke check:

- `statements list --register-number R001255 --grep Soziokultur --limit 2` returned one matching statement and did not leak the API key.

## Final SKILL.md decisions

The finalized skill prioritizes:

- auth from `LOBBYREGISTER_API_KEY`
- search -> exact register number -> source -> dossier workflow
- finance ranges and statement text as first-class evidence
- strong caveats for self-reported register data
- small default limits for broad searches
- cross-checking parliamentary claims with DIP/Bundestag tools

## Runtime note

The old runtime location `agent/bin/lobbyregisterctl.exe` has been moved into
this skill folder as the legacy binary. Deployment/runtime code that still
expects the old path should either:

- point to `skills/bundestag-lobbyregister/bin/lobbyregisterctl-2.0.exe`
- copy/symlink the selected binary into the runtime image path
- call the Python or Node.js fallback explicitly

Do this integration step when the runtime wiring is refactored; it was
intentionally not done as part of this isolated tool pass.
