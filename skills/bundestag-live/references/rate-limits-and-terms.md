# Rate Limits, Auth, And Terms

## Authentication

No authentication is required for the tested Bundestag live/site XML endpoints.

Evidence:

- The OpenAPI spec in this skill contains no security scheme or API-key requirement.
- Live tests against member, committee, plenary, article, and WebTV endpoints succeeded without credentials.

## Published rate limits

No exact public request quota or numeric rate limit was found in the reviewed Bundestag or bundesAPI materials for this XML surface.

Practical fair-use guidance:

- Use `members search` and `committees search` before detail fetches.
- Keep `--limit`, `--item-limit`, `--member-limit`, and `--news-limit` small.
- Cache the member and committee indexes inside a larger orchestrated run when possible.
- Back off on `429`, `403`, `5xx`, connection timeouts, or unusually slow responses.
- Do not loop over every member biography unless the user explicitly asks for a bulk job and you have a caching/batching plan.

## Content and reuse terms

Bundestag website content is not automatically unrestricted. The Bundestag legal notice says published website content is copyright-protected and, unless otherwise specified, may only be downloaded or printed for private use; further use generally requires Bundestag permission. The same page notes that Bundestag printed papers and plenary protocols are official works without copyright protection under German law, but they remain subject to source attribution and alteration restrictions.

Source: https://www.bundestag.de/services/impressum

## Audio and video

The Bundestag audio/video terms allow use of parliamentary television material for parliamentary reporting and educational/cultural purposes, but not for commercial advertising. The source should be cited as `Deutscher Bundestag`, and the Bundestag does not guarantee availability or quality.

Source: https://www.bundestag.de/mediathek/nutzungsbedingungen-247892

## Images

Images can have separate terms. The Bundestag image database terms require source attribution and prohibit commercial advertising and campaign use in all cases.

Source: https://www.bundestag.de/bildnutz

## CLI behavior derived from this

- `doctor` reports no auth requirement and no exact published rate limit.
- Broad commands default to small limits.
- `video feed` always emits a media usage warning.
- Normalized outputs include `sources[]`, `warnings[]`, and `retrievedAt`.
- Research output should cite public Bundestag URLs, not just API URLs.
