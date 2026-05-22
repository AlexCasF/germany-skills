# Bundesrat Live Manifest

## Moved Files

| Original path | New path |
| --- | --- |
| `agent/skills/api-bundesrat-live/SKILL.md` | `skills/bundesrat-live/SKILL.md` |
| `agent/skills/api-bundesrat-live/references/notes.md` | `skills/bundesrat-live/references/notes.md` |
| `agent/skills/api-bundesrat-live/references/openapi.yaml` | `skills/bundesrat-live/references/openapi.yaml` |
| `cli/cmd/bundesratctl/main.go` | `skills/bundesrat-live/go/v1/main.go` |
| `agent/bin/bundesratctl.exe` | `skills/bundesrat-live/bin/bundesratctl-legacy.exe` |

## Added Files

| Path | Purpose |
| --- | --- |
| `skills/bundesrat-live/go/v2/main.go` | Go 2.0 CLI source. |
| `skills/bundesrat-live/go/v2/go.mod` | Go module file. |
| `skills/bundesrat-live/bin/bundesratctl-2.0.exe` | Built Go 2.0 executable. |
| `skills/bundesrat-live/python/bundesratctl.py` | Python CLI implementation. |
| `skills/bundesrat-live/typescript/src/index.ts` | TypeScript / Node.js CLI implementation. |
| `skills/bundesrat-live/typescript/dist/index.js` | Built TypeScript / Node.js CLI output. |
| `skills/bundesrat-live/typescript/package.json` | Node package metadata and scripts. |
| `skills/bundesrat-live/typescript/package-lock.json` | Locked Node dependency graph. |
| `skills/bundesrat-live/typescript/tsconfig.json` | TypeScript compiler configuration. |
| `skills/bundesrat-live/references/research.md` | API research summary. |
| `skills/bundesrat-live/references/rate-limits-and-terms.md` | Auth, rate-limit, and fair-use findings. |
| `skills/bundesrat-live/tests/test-plan.md` | Ten-case shared test plan. |
| `skills/bundesrat-live/tests/go-2.0-results.md` | Go test notes. |
| `skills/bundesrat-live/tests/python-results.md` | Python test notes. |
| `skills/bundesrat-live/tests/typescript-node-results.md` | TypeScript / Node.js test notes. |

## Applied Lessons

- Preserved raw XML access through `--raw`.
- Added `doctor`, `examples`, `news search`, `news page`, `dates search`, `dates page`, `members search`, `members dossier`, `plenum dossier`, `votes summary`, `page`, and `source`.
- Normalized XML feeds into compact JSON envelopes.
- Added source URLs, warnings, retrieved timestamps, and next actions.
- Added safe default limits for broad feed, member, and plenary flows.
- Added `--grep` support for public-page and embedded-detail snippets.
- Documented the difference between current Bundesrat app/site data and archive-grade DIP research.
