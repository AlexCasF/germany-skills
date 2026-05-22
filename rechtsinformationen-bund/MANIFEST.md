# Rechtsinformationen des Bundes manifest

## Original files moved

| Original path | New path |
| --- | --- |
| `agent/skills/api-rechtsinformationen-bund/SKILL.md` | `skills/rechtsinformationen-bund/SKILL.md` |
| `agent/skills/api-rechtsinformationen-bund/references/docs.md` | `skills/rechtsinformationen-bund/references/docs.md` |
| `agent/skills/api-rechtsinformationen-bund/references/notes.md` | `skills/rechtsinformationen-bund/references/notes.md` |
| `agent/skills/api-rechtsinformationen-bund/references/openapi.json` | `skills/rechtsinformationen-bund/references/openapi.json` |
| `cli/cmd/rechtsinformationenctl/main.go` | `skills/rechtsinformationen-bund/go/v1/main.go` |
| `agent/bin/rechtsinformationenctl.exe` | `skills/rechtsinformationen-bund/bin/rechtsinformationenctl-legacy.exe` |

## Implementations

| Implementation | Path | Status |
| --- | --- | --- |
| Legacy Go | `go/v1/main.go` | Preserved unchanged for comparison. |
| Go 2.0 | `go/v2/main.go` | Built and tested. |
| Go 2.0 binary | `bin/rechtsinformationenctl-2.0.exe` | Built locally with Go 1.26.2. |
| Python | `python/rechtsinformationenctl.py` | Created and tested with Python 3.14.3. |
| TypeScript/Node.js | `typescript/src/index.ts` | Created, built, and tested with Node 24.13.1. |

## Applied lessons

This refactor applies the CLI research lessons from:

- `docs/live-research-lessons.md`
- `docs/cli-agent-friendliness-plan.md`
- `docs/cli-tool-refactor-execution-plan.md`

Key changes:

- Kept legacy endpoint access available.
- Added `doctor` for endpoint health, auth, rate limit, and live collection counts.
- Added `source` / `documents source` to expand known records into API, HTML, XML, and ZIP URLs.
- Added `documents text` for HTML/XML source extraction and grep snippets.
- Added `documents dossier` for compact evidence bundles.
- Added `cite` for citation-oriented output.
- Normalized compact search results so agents get identifiers, text-match hints, and source URLs immediately.
- Preserved source categories: legislation, case law, literature, administrative directives.
- Kept ELI, ECLI, document numbers, dates, and court/publication context visible.

## Research findings

- The API is an official German federal legal information trial service.
- It currently provides federal legislation and federal case law most usefully; literature and administrative-directive counts were zero during tests.
- It does not require authentication.
- The documented rate limit is 600 requests per minute per client IP.
- Exceeding the rate limit may return HTTP 503.
- The dataset is not complete and endpoint behavior may change during the test phase.

Primary references:

- `https://testphase.rechtsinformationen.bund.de/`
- `https://docs.rechtsinformationen.bund.de/`
- `https://docs.rechtsinformationen.bund.de/get-started`
- `https://docs.rechtsinformationen.bund.de/guides/rate-limiting`
- `https://testphase.rechtsinformationen.bund.de/openapi.json`

## Test status

All three 2.0 implementations passed the 10-case test plan:

- `tests/test-plan.md`
- `tests/go-2.0-results.md`
- `tests/python-results.md`
- `tests/typescript-node-results.md`

The Go implementation initially exposed a regex panic in HTML stripping; this was fixed and retested.

## Runtime note

The old runtime location `agent/bin/rechtsinformationenctl.exe` has been moved into this skill folder as the legacy binary. Deployment/runtime code that still expects the old path should either:

- point to `skills/rechtsinformationen-bund/bin/rechtsinformationenctl-2.0.exe`
- copy/symlink the selected binary into the runtime image path
- call the Python or Node.js fallback explicitly

Do this integration step when the runtime wiring is refactored; it was intentionally not done as part of this isolated tool pass.
