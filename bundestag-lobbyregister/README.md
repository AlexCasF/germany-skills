# Bundestag Lobbyregister tool

This skill folder contains the CLI and guidance for official Bundestag
Lobbyregister research.

The current API provides read-only access to public lobby-register contents for
interests represented toward the German Bundestag and the Federal Government.
It is useful for registered organizations/persons, activity descriptions,
fields of interest, financial expense ranges, funding sources, donations,
membership fees, public allowances, contracts, regulatory projects, statements,
public detail pages, PDFs, and statistics.

## Implementations

| Implementation | Path | Notes |
| --- | --- | --- |
| Go | `go/main.go` | Agent-friendly CLI with auth, safe search, source, dossier, statistics, financial, and statement helpers. |
| Python | `python/bundestag-lobbyregister.py` | Python parity implementation. |
| TypeScript / Node.js | `typescript/src/index.ts` | TypeScript source compiled to Node.js JavaScript. |

## Runtime data

The official current API requires an API key. Use `LOBBYREGISTER_API_KEY` in the
environment. `--apikey` is supported for local compatibility, but normalized
outputs redact key material.

No exact request-per-minute rate limit was found in the official docs reviewed.
Use small limits, avoid broad repeated searches, and prefer exact register
numbers after discovery.

## Primary sources

- `https://www.lobbyregister.bundestag.de/informationen-und-hilfe/open-data-1049716`
- `https://api.lobbyregister.bundestag.de/rest/v2/swagger-ui/`
- `https://api.lobbyregister.bundestag.de/rest/v2/R2.21-de.yaml`
- `https://github.com/bundesAPI/bundestag-lobbyregister-api`
