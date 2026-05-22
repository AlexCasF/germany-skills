# Rechtsinformationen des Bundes notes

## What this API provides

The preview API exposes:

- German federal legislation, including current and historical expressions
- federal case law
- legal literature metadata
- administrative-directive metadata and future administrative-regulations work
- cross-collection document listing and search
- service statistics

## Response style

- JSON by default
- Hydra collection envelopes for list endpoints
- HTML and XML renditions for many entity endpoints
- some ZIP encodings advertised for legislation and case-law content packages

## Access notes

- docs state the API is currently open and does not require an API key
- docs state the service is in trial phase and the dataset is not yet complete
- docs rate limit: `600 requests per minute` per client IP

## High-value research patterns

- use `documents search` for broad legal search across collections
- use `case-law list` when court decisions are the primary target
- use `case-law courts` for quick court-level inventory counts
- use `legislation list` with `searchTerm`, `eli`, `temporalCoverageFrom`, `temporalCoverageTo`, or `mostRelevantOn` to find the right expression
- use entity HTML and XML variants when the user needs the full rendered text or source encoding

## Pagination and filtering

- list endpoints use `size` and `pageIndex`
- default page size documented in the guides is `100`
- full-text filtering uses `searchTerm`
- quoted phrases in `searchTerm` are treated as exact phrases
- date ranges are inclusive

## Known rough edges

- docs and live preview are not perfectly in sync
- `literature` and `administrative-directive` are currently exposed but appear empty in live statistics
- the docs still describe future or preview surfaces such as exports that are not yet live in the current preview API
