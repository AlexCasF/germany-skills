# Regionalatlas Manifest

## Moved Files

| Original path | New path |
| --- | --- |
| `agent/skills/api-regionalatlas/SKILL.md` | `skills/regionalatlas/SKILL.md` |
| `agent/skills/api-regionalatlas/references/notes.md` | `skills/regionalatlas/references/notes.md` |
| `agent/skills/api-regionalatlas/references/openapi.yaml` | `skills/regionalatlas/references/openapi.yaml` |
| `cli/cmd/regionalatlasctl/main.go` | `skills/regionalatlas/go/v1/main.go` |
| `agent/bin/regionalatlasctl.exe` | `skills/regionalatlas/bin/regionalatlasctl-legacy.exe` |

## Added Files

| Path | Purpose |
| --- | --- |
| `skills/regionalatlas/go/v2/main.go` | Go 2.0 CLI source. |
| `skills/regionalatlas/go/v2/go.mod` | Go module file. |
| `skills/regionalatlas/bin/regionalatlasctl-2.0.exe` | Built Go 2.0 executable. |
| `skills/regionalatlas/python/regionalatlasctl.py` | Python CLI implementation. |
| `skills/regionalatlas/typescript/src/index.ts` | TypeScript / Node.js CLI implementation. |
| `skills/regionalatlas/typescript/package.json` | Node package metadata and scripts. |
| `skills/regionalatlas/typescript/tsconfig.json` | TypeScript compiler configuration. |
| `skills/regionalatlas/typescript/README.md` | TypeScript runtime notes. |
| `skills/regionalatlas/references/research.md` | API research summary. |
| `skills/regionalatlas/references/rate-limits-and-terms.md` | Auth, rate-limit, and fair-use findings. |
| `skills/regionalatlas/tests/test-plan.md` | Ten-case shared test plan. |
| `skills/regionalatlas/tests/go-2.0-results.md` | Go test notes. |
| `skills/regionalatlas/tests/python-results.md` | Python test notes. |
| `skills/regionalatlas/tests/typescript-node-results.md` | TypeScript / Node.js test notes. |

## Applied Lessons

- Kept raw endpoint access as `query`, but added safer researcher-facing commands first.
- Added `doctor`, `indicators search`, `fields`, `sample`, `source`, `dossier`, `query-builder`, and `explain-field`.
- Added structured JSON envelopes with `sources`, `warnings`, and `nextActions`.
- Added hard output caps and explicit `--allow-large-output`.
- Added `--layer-file` because raw JSON shell quoting is fragile, especially on Windows.
- Stripped UTF-8 BOM from `--layer-file` content after testing revealed ArcGIS rejects BOM-prefixed layer JSON.
- Preserved the original Go source and executable for comparison.
