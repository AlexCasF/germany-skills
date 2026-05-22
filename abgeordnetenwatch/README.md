# Abgeordnetenwatch tool

This skill folder contains the CLI and guidance for abgeordnetenwatch.de
politician, mandate, side-job, and parliamentary-data research.

The API is useful for public profile data, parliaments, parliament periods,
politicians, candidacies and mandates, polls, votes, parties, side jobs, and
side-job organizations.

## Implementations

| Implementation | Path | Notes |
| --- | --- | --- |
| Go | `go/main.go` | Agent-friendly CLI with doctor, search, page, source, and dossier helpers. |
| Python | `python/abgeordnetenwatch.py` | Python parity implementation. |
| TypeScript / Node.js | `typescript/src/index.ts` | TypeScript source compiled to Node.js JavaScript. |

## Runtime data

The public API is unauthenticated and returns JSON.

The official API documentation says the data is provided under CC0 1.0.
No exact request-per-minute rate limit was found in the official API docs, so
the CLI uses small default limits and records fair-use warnings.
