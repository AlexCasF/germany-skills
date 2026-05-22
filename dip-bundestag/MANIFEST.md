# DIP Bundestag manifest

## Original files moved

| Original path | New path |
| --- | --- |
| `agent/skills/api-dip-bundestag/SKILL.md` | `skills/dip-bundestag/SKILL.md` |
| `agent/skills/api-dip-bundestag/references/notes.md` | `skills/dip-bundestag/references/notes.md` |
| `agent/skills/api-dip-bundestag/references/openapi.yaml` | `skills/dip-bundestag/references/openapi.yaml` |
| `cli/cmd/dipctl/main.go` | `skills/dip-bundestag/go/v1/main.go` |
| `agent/bin/dipctl.exe` | `skills/dip-bundestag/bin/dipctl-legacy.exe` |

## Applied lessons

This refactor applies the CLI research lessons from:

- `docs/live-research-lessons.md`
- `docs/cli-agent-friendliness-plan.md`
- `docs/cli-tool-refactor-execution-plan.md`

Key implications:

- Keep legacy endpoint access available.
- Add `DIP_API_KEY` environment fallback and redact credentials.
- Add `doctor` for endpoint and auth health.
- Add safe source/text/dossier helpers instead of relying on ad hoc `curl`.
- Preserve official-source context, especially for plenary-session claims.
- Keep broad outputs bounded where helper commands summarize results.

## Final SKILL.md decisions

`SKILL.md` was finalized after implementation and tests, not before.

Agent-facing changes:

- Added `doctor` as the first auth/health check.
- Added `DIP_API_KEY` as the preferred auth path.
- Promoted `source`, `text`, and `dossier` commands as the evidence-first route.
- Added a warning that official plenary-session claims belong in `plenarprotokoll-text`, not news or outside quotes.
- Added the tested v2 command list.

## Current status

This folder has completed the first tool refactor pass.

## New files added

| Path | Purpose |
| --- | --- |
| `README.md` | Tool overview and runtime notes. |
| `references/research.md` | API purpose, data coverage, auth, and use-case notes. |
| `references/rate-limits-and-terms.md` | Auth, rate/fair-use, result-size, and reuse notes with source links. |
| `go/README.md` | Go v1/v2 notes. |
| `go/v2/main.go` | Standalone Go 2.0 CLI implementation. |
| `go/v2/go.mod` | Local Go module for standalone build. |
| `bin/dipctl-2.0.exe` | Built Go 2.0 executable. |
| `python/README.md` | Python run notes. |
| `python/dipctl.py` | Python stdlib CLI implementation. |
| `typescript/README.md` | TypeScript/Node.js build and run notes. |
| `typescript/package.json` | Local TypeScript build package. |
| `typescript/package-lock.json` | Locked TypeScript dev dependencies. |
| `typescript/tsconfig.json` | TypeScript compiler config. |
| `typescript/src/index.ts` | TypeScript/Node.js CLI source. |
| `typescript/dist/index.js` | Compiled Node.js CLI output. |
| `tests/test-plan.md` | Shared 10-case behavioral test plan. |
| `tests/go-2.0-results.md` | Go 2.0 test results. |
| `tests/python-results.md` | Python test results. |
| `tests/typescript-node-results.md` | TypeScript/Node.js test results. |

## Implemented behavior

The refactor preserves the legacy endpoint commands:

- `vorgang list|get`
- `drucksache list|get`
- `plenarprotokoll list|get`
- `person list|get`
- `aktivitaet list|get`

It also adds endpoint access for OpenAPI resources that existed in DIP but were
not exposed by the old wrapper:

- `vorgangsposition list|get`
- `drucksache-text list|get`
- `plenarprotokoll-text list|get`

Research-oriented additions:

- `doctor`
- `person search`
- `person dossier`
- `vorgang dossier`
- `source`
- `plenarprotokoll text`
- `drucksache text`
- `plenary speech search`

## Build and test status

| Implementation | Build | Tests |
| --- | --- | --- |
| Go 2.0 | Pass | 10/10 pass |
| Python | Not compiled | 10/10 pass |
| TypeScript/Node.js | Pass | 10/10 pass |

## Important notes

- `DIP_API_KEY` is preferred. `--apikey` remains supported for compatibility.
- Keys are not printed in normalized output.
- Legacy commands keep upstream-style JSON by default.
- Research commands return normalized JSON envelopes.
- TypeScript/Node.js uses Node's built-in `https` module, not global `fetch`, because Node 24 on Windows produced a libuv assertion after some successful `fetch` requests.
- PowerShell may wrap native stderr from failing commands. Bad-key tests were inspected through `cmd /c "... 2>&1"` to verify raw JSON error payloads.

## Open follow-ups

- The runtime currently no longer has `agent/bin/dipctl.exe` because the legacy binary moved to `skills/dip-bundestag/bin/dipctl-legacy.exe` and Go 2.0 lives at `skills/dip-bundestag/bin/dipctl-2.0.exe`.
- Before deploying, update runtime tool discovery to use `skills/dip-bundestag/bin/dipctl-2.0.exe` or add a packaging step that copies the selected binary into the runtime image.
- The `plenary speech search` command is intentionally conservative. For exact transcript snippets, prefer `plenarprotokoll text --document-number ... --grep ...`.

## Inspect first

- `go/v2/main.go`
- `python/dipctl.py`
- `typescript/src/index.ts`
- `tests/go-2.0-results.md`
- `tests/python-results.md`
- `tests/typescript-node-results.md`
