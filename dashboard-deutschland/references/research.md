# Dashboard Deutschland Research

Retrieved: 2026-05-19

## Overall Description

Dashboard Deutschland is a publicly accessible dashboard offering from Destatis/Statistisches Bundesamt. It presents curated indicators on socially and economically relevant topics, supplemented by charts and text. The dashboard is intended to make current key figures and developments easier to understand.

The BMWE page describes the platform as a free public online offering developed on behalf of BMI, BMF, and BMWE, with more than 100 indicators from different data sources on topics such as health, economy, mobility, and finance.

## Data Domains

Live dashboard sections and API metadata showed domains including:

- labor market
- energy
- foreign trade
- industries and economic activity
- consumption
- prices
- national economy
- housing and construction
- finance and public finances
- mobility
- health and crisis-related indicators

## Technical Shape

The OpenAPI wrapper and live probes identify three endpoint families:

- Dashboard sections: `https://www.dashboard-deutschland.de/api/dashboard/get`
- Indicator tiles: `https://www.dashboard-deutschland.de/api/tile/indicators?ids=<id>`
- GeoJSON: `https://www.dashboard-deutschland.de/geojson/de-all.geo.json`

Live tests found:

- `dashboard/get` returned 17 dashboard sections.
- These sections referenced about 100 unique indicator IDs.
- `tile/indicators` requires `ids`; missing `ids` returned HTTP 500 with a required-parameter message.
- Unknown indicator IDs returned an empty array with HTTP 200.
- The documented GeoJSON endpoint returned HTTP 403 `AccessDenied`.

## Embedded Tile JSON

Each indicator item has a top-level `json` string. This string contains the important research payload:

- chart components
- Highcharts series and points
- compact widgets
- explanatory text
- source links
- tags
- `lastUpdated`
- `dataVersionDate`
- `dateUpload`

The CLI parses this field and exposes chart-ready data through `indicator data`.

## Tested Example

The refactor used `<indicator-id>`, "Indikator und offene Stellen", as the primary test tile.

Observed normalized contents included:

- chart series `Indikator`
- chart series `gemeldete offene Stellen`
- source link to Statistik der Bundesagentur fuer Arbeit
- source link to Macrobond
- widgets for unemployment rate, month-over-month unemployed persons, and open vacancies
- data version date `Maerz 2026`

## Best Uses

Use Dashboard Deutschland when the user wants:

- a fast curated overview indicator
- chart-ready recent series points
- source links for a dashboard tile
- high-level economic or social context
- quick comparison across dashboard topics

## Less Suitable Uses

Use another tool when the user needs:

- full GENESIS table configuration
- comprehensive statistical metadata
- granular regional atlas data
- legal/parliamentary/register source evidence

## Sources

- Dashboard Deutschland: https://www.dashboard-deutschland.de/
- Dashboard endpoint: https://www.dashboard-deutschland.de/api/dashboard/get
- Indicator endpoint: https://www.dashboard-deutschland.de/api/tile/indicators
- GeoJSON endpoint: https://www.dashboard-deutschland.de/geojson/de-all.geo.json
- Destatis dashboards page: https://www.destatis.de/DE/Ueber-uns/Aufgaben/dashboards.html
- BMWE Dashboard Deutschland page: https://www.bundeswirtschaftsministerium.de/Redaktion/DE/Dossier/WirtschaftlicheEntwicklung/dashboard-deutschland.html
- PyPI generated package docs: https://pypi.org/project/de-dashboarddeutschland/
- Dashboard Deutschland OpenAPI wrapper: https://github.com/AndreasFischer1985/dashboard-deutschland-api
