# Abgeordnetenwatch manifest

## Original files moved

| Original path | New path |
| --- | --- |
| `agent/skills/api-abgeordnetenwatch/SKILL.md` | `skills/abgeordnetenwatch/SKILL.md` |
| `agent/skills/api-abgeordnetenwatch/references/docs.md` | `skills/abgeordnetenwatch/references/docs.md` |
| `agent/skills/api-abgeordnetenwatch/references/notes.md` | `skills/abgeordnetenwatch/references/notes.md` |
| `cli/cmd/abgeordnetenwatchctl/main.go` | `skills/abgeordnetenwatch/go/v1/main.go` |
| `agent/bin/abgeordnetenwatchctl.exe` | `skills/abgeordnetenwatch/bin/abgeordnetenwatchctl-legacy.exe` |

## Implementations

| Implementation | Path | Status |
| --- | --- | --- |
| Legacy Go | `go/v1/main.go` | Preserved unchanged for comparison. |
| Go 2.0 | `go/v2/main.go` | Built and tested. |
| Go 2.0 binary | `bin/abgeordnetenwatchctl-2.0.exe` | Built locally with Go 1.26.2. |
| Python | `python/abgeordnetenwatchctl.py` | Created and tested with Python 3.14.3. |
| TypeScript/Node.js | `typescript/src/index.ts` | Created, built, and tested with Node 24.13.1. |

## Applied lessons

This refactor applies the CLI research lessons from:

- `docs/live-research-lessons.md`
- `docs/cli-agent-friendliness-plan.md`
- `docs/cli-tool-refactor-execution-plan.md`

Key changes:

- Kept all legacy `entity list|get` endpoint access.
- Added `doctor` for endpoint health, auth status, license, result-limit, and fair-use context.
- Added `politicians search` with compact source-rich results.
- Added `politicians source` for API/profile/mandate source URLs.
- Added `politicians page` for public profile-page extraction and grep snippets.
- Added `politicians dossier` with API record, mandates, side jobs, profile-page snippets, warnings, and next actions.
- Added `mandates for-politician` and `sidejobs for-politician` to make the politician -> mandate -> sidejob join explicit.
- Added sidejob and related entity endpoint access beyond the old five-entity wrapper, while keeping old commands intact.

## Research findings

- The API is unauthenticated and returns JSON.
- The API metadata reports CC0 1.0 licensing.
- No exact request-per-minute rate limit was found in official docs or live headers.
- Official docs describe range/pager limits up to 1,000 returned entities.
- Broad endpoints default to 100 returned entities; the 2.0 CLI uses smaller discovery defaults.
- Invalid IDs may return upstream HTTP 500 rather than 404.
- Side jobs are best queried through mandate IDs, not directly by politician ID.

Primary references:

- `https://www.abgeordnetenwatch.de/api`
- `https://www.abgeordnetenwatch.de/api/response`
- `https://www.abgeordnetenwatch.de/api/version-changelog/aktuell`
- `https://www.abgeordnetenwatch.de/api/entitaeten/sidejob`
- `https://www.abgeordnetenwatch.de/api/entitaeten/sidejob-organization`

## Test status

All three 2.0 implementations passed the 10-case test plan:

- `tests/test-plan.md`
- `tests/go-2.0-results.md`
- `tests/python-results.md`
- `tests/typescript-node-results.md`

The Python implementation initially exposed a Windows console encoding issue; stdout/stderr are now explicitly UTF-8.

## Final SKILL.md decisions

The finalized skill prioritizes:

- when to use abgeordnetenwatch as transparency/profile context
- when to switch to official DIP/Bundestag/legal sources
- search -> source -> page -> dossier workflow
- side-job interpretation caveats
- small limits and no broad page crawling

## Runtime note

The old runtime location `agent/bin/abgeordnetenwatchctl.exe` has been moved into this skill folder as the legacy binary. Deployment/runtime code that still expects the old path should either:

- point to `skills/abgeordnetenwatch/bin/abgeordnetenwatchctl-2.0.exe`
- copy/symlink the selected binary into the runtime image path
- call the Python or Node.js fallback explicitly

Do this integration step when the runtime wiring is refactored; it was intentionally not done as part of this isolated tool pass.
