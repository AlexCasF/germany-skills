# Bundesrat Rate Limits, Auth, And Use Notes

## Authentication

No authentication requirement was found for the documented Bundesrat XML feeds.

Evidence:

- The OpenAPI file defines public GET endpoints on `https://www.bundesrat.de`.
- No `securitySchemes`, API-key parameter, OAuth flow, or auth header requirement appears in the local OpenAPI file.
- Live endpoint checks succeeded without credentials.

## Rate Limits

No exact public request quota was found for the Bundesrat XML feeds.

The strongest explicit fair-use signal found is `robots.txt`, which currently includes:

```text
Crawl-delay: 30
```

Treat that as guidance for crawling-like workflows, especially `page --url` expansion across many public HTML pages.

## Fair-Use Guidance For Agents

- Prefer one feed request plus small filtered expansions.
- Use `--limit` and `--top-limit` for broad commands.
- Do not repeatedly fetch the same full feed in a loop; cache within a research run where possible.
- Back off on 429, 403, 5xx, connection reset, or slow responses.
- Avoid bulk public-page crawling; expand only URLs needed for the current answer.
- Preserve source URLs, retrieval timestamps, and copyright/image metadata.

## Content And Attribution Notes

The Bundesrat website includes public information, source pages, images, PDFs, and media links. Public availability does not imply unrestricted reuse.

For final artifacts:

- cite Bundesrat and the specific URL
- distinguish XML feed data from public HTML page extraction
- preserve image/media copyright notices when image fields are used
- avoid reproducing long copyrighted page text verbatim

## Sources Checked

- Bundesrat OpenAPI wrapper: https://github.com/bundesAPI/bundesrat-api
- Bundesrat robots.txt: https://www.bundesrat.de/robots.txt
- Bundesrat impressum: https://www.bundesrat.de/DE/service-navi/impressum/impressum-node.html
- Bundesrat privacy policy: https://www.bundesrat.de/DE/service-navi/datenschutz/datenschutz-node.html
- Bundesrat website homepage: https://www.bundesrat.de/DE/homepage/homepage-node.html
