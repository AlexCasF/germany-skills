# Rechtsinformationen des Bundes tool

This skill folder contains the CLI and guidance for the official German
federal legal information preview API.

The API is useful for German federal legislation, federal court decisions,
legal literature metadata, administrative-directive metadata, cross-collection
legal search, and HTML/XML source renditions.

## Implementations

| Implementation | Path | Notes |
| --- | --- | --- |
| Go | `go/main.go` | Agent-friendly CLI with doctor, source, text, dossier, and cite helpers. |
| Python | `python/rechtsinformationen-bund.py` | Python parity implementation. |
| TypeScript / Node.js | `typescript/src/index.ts` | TypeScript source compiled to Node.js JavaScript. |

## Runtime data

The preview API is currently open and does not require an API key.

The documented rate limit is 600 requests per minute per client IP.

The service is in public test phase. Treat the data surface and endpoint
behavior as subject to change, and preserve retrieval dates in research
artifacts.

## Validation

The Go, Python, and TypeScript/Node.js implementations all passed the
