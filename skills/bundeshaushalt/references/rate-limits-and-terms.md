# Rate Limits, Auth, And Terms

## Authentication

No authentication was required in live tests of:

```text
https://bundeshaushalt.de/internalapi/budgetData
```

The CLI therefore has no API key or credential configuration for this service.

## Published Rate Limits

No exact public request quota was found for the Bundeshaushalt Digital internal API.

The site's `robots.txt` publishes crawler guidance, including `Crawl-delay: 30`. Treat that as a fair-use signal for crawling-style workflows. Interactive, narrow API calls are different from crawling, but agents should still avoid repeated broad traversal.

Source:

- https://www.bundeshaushalt.de/robots.txt

## Fair-Use Guidance For Agents

- Prefer narrow `budget tree` and `title get` requests over broad traversal.
- Keep `search --limit` low.
- Increase `--max-requests` only when necessary.
- Cache repeated hierarchy traversals in higher-level workflows where possible.
- Preserve source URLs and retrieved timestamps in artifacts.
- Do not claim a hard quota unless an official quota is later published.

## Terms And Attribution Context

Use BMF/Bundeshaushalt attribution in final outputs. The most relevant official context and usage pages found during research were:

- Bundeshaushalt Digital: https://www.bundeshaushalt.de/DE/Bundeshaushalt-digital/bundeshaushalt-digital.html
- Bundeshaushalt user notes: https://www.bundeshaushalt.de/DE/Service/Benutzerhinweise/benutzerhinweise.html
- Bundeshaushalt imprint: https://www.bundeshaushalt.de/DE/Service/Impressum/impressum.html
- BMF data portal usage notes: https://www.bundesfinanzministerium.de/Datenportal/Nutzungshinweise/nutzungshinweise.html

The BMF data portal usage notes are adjacent public-data guidance, not a service-specific quota page for the internal Bundeshaushalt endpoint.
