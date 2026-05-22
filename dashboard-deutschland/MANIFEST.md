# Dashboard Deutschland Manifest

## Moved Files

| Original path | New path |
| --- | --- |
| `agent/skills/api-dashboard-deutschland/SKILL.md` | `skills/dashboard-deutschland/SKILL.md` |
| `agent/skills/api-dashboard-deutschland/references/notes.md` | `skills/dashboard-deutschland/references/notes.md` |
| `agent/skills/api-dashboard-deutschland/references/openapi.yaml` | `skills/dashboard-deutschland/references/openapi.yaml` |
| `cli/cmd/dashboardctl/main.go` | `skills/dashboard-deutschland/go/v1/main.go` |
| `agent/bin/dashboardctl.exe` | `skills/dashboard-deutschland/bin/dashboardctl-legacy.exe` |

## Added Files

| Path | Purpose |
| --- | --- |
| `skills/dashboard-deutschland/go/v2/main.go` | Go 2.0 CLI source. |
| `skills/dashboard-deutschland/go/v2/go.mod` | Go module file. |
| `skills/dashboard-deutschland/bin/dashboardctl-2.0.exe` | Built Go 2.0 executable. |
| `skills/dashboard-deutschland/python/dashboardctl.py` | Python CLI implementation. |
| `skills/dashboard-deutschland/typescript/src/index.ts` | TypeScript / Node.js CLI implementation. |
| `skills/dashboard-deutschland/typescript/package.json` | Node package metadata and scripts. |
| `skills/dashboard-deutschland/typescript/tsconfig.json` | TypeScript compiler configuration. |
| `skills/dashboard-deutschland/references/research.md` | API research summary. |
| `skills/dashboard-deutschland/references/rate-limits-and-terms.md` | Auth, rate-limit, and fair-use findings. |
| `skills/dashboard-deutschland/tests/test-plan.md` | Ten-case shared test plan. |
| `skills/dashboard-deutschland/tests/go-2.0-results.md` | Go test notes. |
| `skills/dashboard-deutschland/tests/python-results.md` | Python test notes. |
| `skills/dashboard-deutschland/tests/typescript-node-results.md` | TypeScript / Node.js test notes. |

## Applied Lessons

- Preserved the raw `dashboard get`, `indicators`, and `geo` commands.
- Added `doctor`, `dashboards list`, `dashboard dossier`, `indicator search`, `indicator get`, `indicator data`, and `indicator source`.
- Normalized the embedded tile `json` string into chart series, widgets, text snippets, sources, and update metadata.
- Added structured JSON envelopes with `sources`, `warnings`, and `nextActions`.
- Added explicit diagnostics for the currently failing GeoJSON endpoint.
- Kept chart data bounded with small default point limits.
- Documented the curated mixed-source nature of Dashboard Deutschland.
