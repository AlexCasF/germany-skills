# Deutschlandatlas notes

## What this API provides

The Deutschlandatlas exposes public ArcGIS MapServer services for regional
indicator tables. The official product describes equal-living-conditions
indicators across Germany, and the download page also provides tabular files and
method/source notes.

## Important endpoint pattern

The historical OpenAPI wrapper documents:

`/{table}/MapServer/0/query`

Live testing showed that current services are not always on layer `0`. For
example, `alq_HA2023` exposes the feature layer at `/MapServer/5`. The CLI
therefore discovers the first feature layer from service metadata by default.

## Good research sequence

1. `tables search --term <topic>`
2. `table fields --table <table>`
3. `table sample --table <table> --limit 5`
4. `indicator dossier --table <table>`
5. Use source URLs in `sources[]` for citation and follow-up.

## Common fields

Many regional layers include fields like:

- `name`: region label
- `GEN`: short region name
- `BEZ`: area type
- `Gebietskennziffer`: regional key
- one short indicator field, such as `alq` for unemployment rate
- `Shape_Length` and `Shape_Area` for geometry-derived values

Do not assume the indicator field name. Inspect fields first.

## Common pitfalls

- Portal search returns map services and basemap services; pick indicator tables,
  not basemaps.
- The table name often encodes the topic/year, but the portal snippet is more
  reliable for human interpretation.
- Broad queries can hit ArcGIS transfer limits and still return partial data.
- Geometry can make outputs very large.
- Missing values in official tabular downloads are documented as `-9999`.
