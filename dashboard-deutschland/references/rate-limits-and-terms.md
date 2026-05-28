# Dashboard Deutschland Rate Limits, Auth, And Terms

Retrieved: 2026-05-19

## Authentication

No authentication was required for the public endpoints tested:

- `https://www.dashboard-deutschland.de/api/dashboard/get`
- `https://www.dashboard-deutschland.de/api/tile/indicators?ids=<indicator-id>`

The generated package documentation on PyPI states that all documented endpoints do not require authorization.

## Rate Limits

No exact public request-per-second or request-per-day limit was found in the reviewed official/API materials.

The API is served behind CDN/cache infrastructure and returns cache/security headers. The CLI therefore uses conservative behavior:

- discover dashboards first
- fetch indicators by explicit ID
- batch indicator search in chunks
- default chart data output to 10 points per series
- treat 429, 5xx, and gateway/object-storage errors as signals to back off

## GeoJSON Endpoint Status

The documented `GET /geojson/de-all.geo.json` endpoint returned:

- HTTP status `403`
- XML body with `AccessDenied`
- CDN/object-storage response headers

This is exposed by `doctor` and `geo` as a structured diagnostic rather than hidden.

## Usage And Attribution Notes

Dashboard Deutschland is a curated mixed-source dashboard. Indicator-level sources can include official statistical sources, public agencies, or selected data providers. Always cite the source links emitted by `indicator get`, `indicator data`, or `indicator source`.

For broader statistical data licensing context, official German statistical open-data pages point to Datenlizenz Deutschland 2.0. Because individual Dashboard Deutschland tiles can combine different sources, preserve tile-specific source attribution.

## Sources

- PyPI generated package docs: https://pypi.org/project/de-dashboarddeutschland/
- Dashboard Deutschland: https://www.dashboard-deutschland.de/
- Dashboard endpoint: https://www.dashboard-deutschland.de/api/dashboard/get
- Indicator endpoint: https://www.dashboard-deutschland.de/api/tile/indicators
- GeoJSON endpoint: https://www.dashboard-deutschland.de/geojson/de-all.geo.json
- Destatis dashboards page: https://www.destatis.de/DE/Ueber-uns/Aufgaben/dashboards.html
- BMWE Dashboard Deutschland page: https://www.bundeswirtschaftsministerium.de/Redaktion/DE/Dossier/WirtschaftlicheEntwicklung/dashboard-deutschland.html
- Statistikportal Open Data licensing context: https://www.statistikportal.de/de/open-data
