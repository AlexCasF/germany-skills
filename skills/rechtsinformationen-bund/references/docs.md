# Rechtsinformationen des Bundes documentation reference

Official documentation sources:

- `https://docs.rechtsinformationen.bund.de/`
- `https://docs.rechtsinformationen.bund.de/sitemap.xml`
- `https://docs.rechtsinformationen.bund.de/get-started`
- `https://docs.rechtsinformationen.bund.de/endpoints`
- `https://docs.rechtsinformationen.bund.de/guides/formats`
- `https://docs.rechtsinformationen.bund.de/guides/pagination`
- `https://docs.rechtsinformationen.bund.de/guides/filters`
- `https://docs.rechtsinformationen.bund.de/guides/rate-limiting`
- `https://docs.rechtsinformationen.bund.de/guides/error-codes`
- `https://docs.rechtsinformationen.bund.de/resourceArchive/legislation`
- `https://docs.rechtsinformationen.bund.de/resourceArchive/case-law`
- `https://docs.rechtsinformationen.bund.de/resourceArchive/search`
- `https://docs.rechtsinformationen.bund.de/resourceArchive/export`
- `https://docs.rechtsinformationen.bund.de/standards`
- `https://docs.rechtsinformationen.bund.de/changelog`
- `https://docs.rechtsinformationen.bund.de/contact`

Important implementation notes from the crawl:

- The docs site is a VitePress SPA with clean URLs.
- The prose docs say the API is currently open and does not require an API key.
- The rate-limiting guide documents `600 requests per minute` per client IP and says excess traffic may receive `503 Service Unavailable`.
- Pagination docs describe Hydra collection responses with `view.first`, `view.previous`, `view.next`, and `view.last`.
- Formats docs describe JSON by default, plus `.xml` and `.html` variants for entity retrieval.
- Error docs show structured `errors` arrays for `403`, `404`, `422`, `500`, and `503`.

Observed doc-to-API mismatches on 2026-04-14:

- `/v1/search` in the docs returns `404`; the live preview serves `/v1/document/lucene-search`.
- `/v1/exports` in the docs returns `404`; the current OpenAPI does not list an export endpoint.
- Some docs examples still use `limit`, but the live preview uses `size`.
- Some case-law detail examples use ECLI-style paths that currently return `404`; the live preview works with document numbers such as `KORE301642020`.
