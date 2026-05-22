# Tagesschau Skill Manifest

## Purpose

This folder contains the Tagesschau skill and CLI implementations for current-news context and bounded article expansion.

## Layout

| Path | Purpose |
| --- | --- |
| `SKILL.md` | Agent-facing instructions for using the Tagesschau tool. |
| `references/openapi.yaml` | Original OpenAPI reference. |
| `references/notes.md` | API behavior and interpretation notes. |
| `references/research.md` | Web/API research summary and source links. |
| `references/rate-limits-and-terms.md` | Auth, rate-limit, reuse, and fair-use findings. |
| `go/v1/main.go` | Legacy generated Go wrapper source. |
| `go/v2/main.go` | Refactored Go 2.0 CLI source. |
| `go/v2/go.mod` | Go module for the 2.0 CLI. |
| `bin/tagesschauctl-legacy.exe` | Preserved legacy binary. |
| `bin/tagesschauctl-2.0.exe` | Built Go 2.0 binary. |
| `python/tagesschauctl.py` | Python CLI mirror. |
| `typescript/src/index.ts` | TypeScript/Node CLI mirror source. |
| `typescript/dist/index.js` | Built TypeScript/Node CLI. |
| `tests/*.md` | Shared test plan and runtime results. |

## 2.0 Improvements

- Preserved homepage, news, channels, and search command families.
- Preserved legacy search parameter behavior with `--param searchText=...`.
- Added `doctor`, `source`, and `fields`.
- Added compact JSON envelopes with `summary`, `items`, `sources`, `warnings`, and `nextActions`.
- Added safe `--limit` handling for broad feed/search commands.
- Added `article source`, `article get`, and `article dossier`.
- Added public/API article URL conversion between `detailsweb` and `/api2u/...json`.
- Added bounded snippet extraction and `--grep`.
- Added Python and TypeScript/Node versions with the same command vocabulary.

## Current Validation

The Go 2.0, Python, and TypeScript/Node versions all passed the shared test plan on 2026-05-19.
