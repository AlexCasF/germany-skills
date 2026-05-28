# Dashboard Deutschland Notes

## What The API Provides

Dashboard Deutschland exposes curated dashboard sections and chart tiles from the public dashboard at `dashboard-deutschland.de`.

The useful live API shape is:

- `GET /api/dashboard/get`: returns dashboard sections with titles, descriptions, categories, tags, and `layoutTiles`.
- `GET /api/tile/indicators?ids=<id>`: returns indicator tiles. The `json` field is itself a JSON string containing chart configuration, sources, widgets, text, update dates, and tags.
- `GET /geojson/de-all.geo.json`: documented GeoJSON endpoint for Germany/Laender, but it returned `403 AccessDenied` during live testing.

## Live Observations

- `dashboard/get` returned 17 dashboard sections.
- The dashboard sections referenced about 100 unique indicator IDs.
- `tile/indicators` requires the `ids` parameter; without it, the endpoint returned a 500 error saying the required parameter is missing.
- An unknown `ids` value returned an empty array with HTTP 200.
- The indicator tile `<indicator-id>` returned data with embedded chart series, widgets, text, source links, and update metadata.

## Common Pitfalls

- Do not treat the `json` field as opaque text; parse it before interpretation.
- Chart config may include JavaScript formatter strings. These are presentation details, not data.
- Source links live inside the embedded tile JSON.
- The dashboard is curated and mixed-source; not every tile is pure Destatis data.
- GeoJSON is currently a diagnostic/failure path unless the endpoint behavior changes.

## Output Guidance

When summarizing a tile, preserve:

- tile ID
- title
- series name and ID
- latest points or requested points
- data version date
- last updated timestamp
- widgets and their descriptions
- source names and URLs
