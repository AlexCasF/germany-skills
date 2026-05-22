# DIP rate limits and terms

Retrieved: 2026-05-18

## Auth

Authentication is required for all API requests. The official documentation
allows either an `Authorization: ApiKey <key>` header or an `apikey` query
parameter.

The public API help page provides a temporary public key. For production use,
prefer a personalized key requested from the Bundestag Parlamentsdokumentation.

Do not commit keys. Use `DIP_API_KEY`.

## Rate and concurrency guidance

The official short documentation does not publish a full request-per-minute
limit. It does state that clients should not execute more than 25 concurrent
API requests, to preserve overall system stability and availability.

The same document says additional stability/rate-limiting details beyond those
published there are not communicated publicly.

Implementation consequence:

- Do not add parallel fetching by default.
- Keep `doctor` and test cases to one request unless a key is missing.
- For future pagination helpers, process sequentially and use explicit limits.
- Avoid background bulk downloads unless the user explicitly asks.

## Result-size guidance

The official short documentation says normal list requests return at most 100
entities per request. Full-text resource types are usually limited to 10
entities. Additional results are loaded with the `cursor` parameter.

Implementation consequence:

- Broad helper commands should default to a small client-side `--limit`.
- Full corpus traversal should be an explicit command, not default behavior.
- Preserve cursor values and suggest next actions.

## Reuse and citation terms

The terms say DIP data is provided free of charge. Machine-controlled access
must use the API.

For reuse beyond personal use:

- API machine-readable data may be used and processed broadly.
- Reuse and redistribution must include source attribution.
- The required source is `Deutscher Bundestag/Bundesrat - DIP`.
- For Bundestag printed papers and plenary protocols, cite the document type
  and number, such as `BT-Drs.` or `BT-PlPr.`.
- Changes, annotations, or excerpting should be identifiable as such.
- Misleading or degrading use contexts are prohibited by the DIP terms.

## Source links

- API help page: https://dip.bundestag.de/%C3%BCber-dip/hilfe/api
- API short documentation PDF: https://dip.bundestag.de/documents/informationsblatt_zur_dip_api.pdf
- Terms PDF: https://dip.bundestag.de/documents/nutzungsbedingungen_dip.pdf

