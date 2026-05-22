# Bundeshaushalt Skill Manifest

## Purpose

This folder contains the Bundeshaushalt Digital skill and CLI implementations.

## Layout

| Path | Purpose |
| --- | --- |
| `SKILL.md` | Agent-facing instructions for using the Bundeshaushalt tool. |
| `references/openapi.yaml` | Original OpenAPI reference. Useful but stale for year availability. |
| `references/notes.md` | API behavior and interpretation notes. |
| `references/research.md` | Web/API research summary and source links. |
| `references/rate-limits-and-terms.md` | Auth, rate-limit, and fair-use findings. |
| `go/v1/main.go` | Legacy generated Go wrapper source. |
| `go/v2/main.go` | Refactored Go 2.0 CLI source. |
| `go/v2/go.mod` | Go module for the 2.0 CLI. |
| `bin/bundeshaushaltctl-legacy.exe` | Preserved legacy binary. |
| `bin/bundeshaushaltctl-2.0.exe` | Built Go 2.0 binary. |
| `python/bundeshaushaltctl.py` | Python CLI mirror. |
| `typescript/src/index.ts` | TypeScript/Node CLI mirror source. |
| `typescript/dist/index.js` | Built TypeScript/Node CLI. |
| `tests/*.md` | Shared test plan and runtime results. |

## 2.0 Improvements

- Preserved the legacy endpoint-style `budget-data` command.
- Fixed the original unexplained HTTP 405 behavior by using the verified live `GET /internalapi/budgetData` request shape.
- Added `doctor`, `source`, `fields`, `years list`, `budget tree`, `budget sample`, `search`, `title get`, and `compare`.
- Added compact JSON envelopes with `summary`, `items`, `sources`, `warnings`, and `nextActions`.
- Added bounded traversal for `search`.
- Added retry handling for transient 429/502/503/504 responses.
- Added parsing for title-level `related` categories.
- Added Python and TypeScript/Node versions with the same command vocabulary.

## Current Validation

The Go 2.0, Python, and TypeScript/Node versions all passed the 10-case test plan on 2026-05-19.
