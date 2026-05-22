# Bundestag Live Manifest

## Moved Files

| Original path | New path |
| --- | --- |
| `agent/skills/api-bundestag-live/SKILL.md` | `skills/bundestag-live/SKILL.md` |
| `agent/skills/api-bundestag-live/references/notes.md` | `skills/bundestag-live/references/notes.md` |
| `agent/skills/api-bundestag-live/references/openapi.yaml` | `skills/bundestag-live/references/openapi.yaml` |
| `cli/cmd/bundestagctl/main.go` | `skills/bundestag-live/go/v1/main.go` |
| `agent/bin/bundestagctl.exe` | `skills/bundestag-live/bin/bundestagctl-legacy.exe` |

## Added Files

| Path | Purpose |
| --- | --- |
| `skills/bundestag-live/go/v2/main.go` | Go 2.0 CLI source. |
| `skills/bundestag-live/go/v2/go.mod` | Go module file. |
| `skills/bundestag-live/bin/bundestagctl-2.0.exe` | Built Go 2.0 executable. |
| `skills/bundestag-live/python/bundestagctl.py` | Python CLI implementation. |
| `skills/bundestag-live/typescript/src/index.ts` | TypeScript / Node.js CLI implementation. |
| `skills/bundestag-live/typescript/dist/index.js` | Built TypeScript / Node.js CLI output. |
| `skills/bundestag-live/typescript/package.json` | Node package metadata and scripts. |
| `skills/bundestag-live/typescript/package-lock.json` | Locked Node dependency graph. |
| `skills/bundestag-live/typescript/tsconfig.json` | TypeScript compiler configuration. |
| `skills/bundestag-live/references/research.md` | API research summary. |
| `skills/bundestag-live/references/rate-limits-and-terms.md` | Auth, rate-limit, and fair-use findings. |
| `skills/bundestag-live/tests/test-plan.md` | Ten-case shared test plan. |
| `skills/bundestag-live/tests/go-2.0-results.md` | Go test notes. |
| `skills/bundestag-live/tests/python-results.md` | Python test notes. |
| `skills/bundestag-live/tests/typescript-node-results.md` | TypeScript / Node.js test notes. |

## Applied Lessons

- Preserved raw XML access through `--raw`.
- Added `doctor`, `members search`, `members dossier`, `committees search`, `committees dossier`, `article page`, `source`, and `examples`.
- Normalized large XML feeds into compact JSON envelopes.
- Added source URLs, warnings, retrieved timestamps, and next actions.
- Added safe default limits for broad member, committee, and agenda flows.
- Added `--grep` support for source snippets.
- Documented the difference between this live/site API and DIP.
