# Abgeordnetenwatch rate limits and terms

Retrieved: 2026-05-18

## Authentication

No API key or token requirement was found in the official API documentation.
Live API requests worked without authentication.

## License

The official API documentation states that API data is provided under CC0 1.0.
The API response metadata also includes:

- `licence: CC0 1.0`
- `licence_link: https://creativecommons.org/publicdomain/zero/1.0/deed.de`

## Result limits

The official API documentation says:

- list responses default to 100 returned entities
- `range_start` and `range_end` can bound returned results
- `range_end` supports up to 1,000 returned results
- `page` plus `pager_limit` can request paginated result pages
- `pager_limit` supports up to 1,000 results per page
- `related_data` integrations are limited to a maximum of 1,000 entities

## Rate limits

No exact request-per-minute or request-per-day rate limit was found in the
official API documentation or live response headers during this pass.

Tool behavior should therefore be conservative:

- use small default limits
- avoid unnecessary page/profile fetches
- do not crawl profile pages broadly
- prefer exact IDs after search
- preserve retrieval dates and source URLs
- back off if the service returns 429, 500, 502, 503, or 504

## Sources

- https://www.abgeordnetenwatch.de/api
- https://www.abgeordnetenwatch.de/api/response
- https://www.abgeordnetenwatch.de/api/version-changelog/aktuell
- https://www.abgeordnetenwatch.de/api/entitaeten/sidejob
- https://www.abgeordnetenwatch.de/api/entitaeten/sidejob-organization
