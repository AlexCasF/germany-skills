# DIP Bundestag tool

This skill folder contains the CLI and guidance for the Bundestag DIP API.

DIP is the official Documentation and Information System for Parliamentary
Material. It is best used for Bundestag and Bundesrat parliamentary materials:
proceedings, printed papers, plenary protocols, activities, person master data,
and related full-text records.

## Implementations

| Implementation | Path | Notes |
| --- | --- | --- |
| Go 2.0 | `go/main.go` | Agent-friendly CLI with env auth, doctor, source, text, and dossier helpers. |
| Python | `python/dip-bundestag.py` | Python parity implementation. |
| TypeScript / Node.js | `typescript/src/index.ts` | TypeScript source compiled to Node.js JavaScript. |

## Runtime data

Use `DIP_API_KEY` for authentication. The key can still be passed with
`--apikey` for backward compatibility, but environment configuration is the
preferred path.

Do not commit API keys.

