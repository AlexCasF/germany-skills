# Tagesschau API Research

## Overall Description

The Tagesschau API provides JSON access to current Tagesschau news and media feeds. It covers homepage selections, current news lists, channel listings, and search results. Detail URLs expose structured article content, including text/headline blocks and metadata.

The API is best used for:

- current-news discovery
- public context around a research topic
- finding article URLs for citation
- extracting short snippets for orientation
- following regional broadcaster context through `shareURL`

It should not replace primary sources for official claims. For parliamentary, legal, fiscal, statistical, or register facts, use DIP, Bundestag, Rechtsinformationen des Bundes, Bundeshaushalt, Destatis, Lobbyregister, or similar primary tools after using Tagesschau for context.

## Source Links

- API documentation: https://github.com/bundesAPI/tagesschau-api
- OpenAPI YAML: https://github.com/bundesAPI/tagesschau-api/raw/refs/heads/main/openapi.yaml
- Public service: https://www.tagesschau.de/
- RSS/reuse notice: https://www.tagesschau.de/infoservices/rssfeeds
- Creative Commons videos: https://www.tagesschau.de/multimedia/video/creative-commons-index-100.html

## Endpoint Notes

`homepage` returns selected current and breaking items plus regional items.

`news` supports filters:

- `ressort`: `inland`, `ausland`, `wirtschaft`, `sport`, `video`, `investigativ`, `wissen`
- `regions`: region codes `1` through `16`, comma-separated for multiple regions

`channels` returns livestream/program channel entries.

`search` supports:

- `searchText`
- `resultPage`
- `pageSize`, documented as 1-30

## Article Expansion

Feed and search results usually expose:

- `details`: API detail URL
- `detailsweb`: public article URL
- `shareURL`: sometimes a regional broadcaster source URL

The 2.0 CLIs accept either public `detailsweb` URLs or API `details` URLs and return both forms. Article expansion returns bounded snippets rather than full article dumps.
