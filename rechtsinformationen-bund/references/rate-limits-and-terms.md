# Rechtsinformationen des Bundes rate limits and fair use

Retrieved: 2026-05-18

## Auth

The current API documentation says the API is open and does not require an API
key.

## Rate limit

The official rate-limiting guide documents:

- 600 requests per minute per client IP
- requests beyond the limit may receive `503 Service Unavailable`
- clients should implement exponential backoff
- clients should cache responses where applicable

Implementation consequence:

- Do not add bulk crawlers by default.
- Keep `doctor` to a very small number of requests.
- Use `size` and `pageIndex` deliberately.
- Preserve `view.next` links instead of auto-following every page.
- Consider caching for repeated source/text expansion later.

## Trial-service caveat

The documentation and website both describe this as a trial service. The API
may change as the service learns from use and feedback.

Implementation consequence:

- `doctor` should mention trial status.
- `SKILL.md` should tell agents to preserve retrieval dates.
- Tests should be resilient to changing counts and record inventories.
- Dossier commands should include source URLs and warnings.

## Fair-use hints

The service explicitly encourages testing and feedback, but the rate limit and
trial status make polite use important.

Recommended behavior:

- Prefer targeted searches.
- Avoid unnecessary repeated HTML/XML downloads.
- Use small `size` values during discovery.
- Respect `503` by backing off.
- Cite official source URLs and retrieval dates in research artifacts.

## Source links

- Rate limiting: https://docs.rechtsinformationen.bund.de/guides/rate-limiting
- Getting started: https://docs.rechtsinformationen.bund.de/get-started
- Portal trial notice: https://testphase.rechtsinformationen.bund.de/

