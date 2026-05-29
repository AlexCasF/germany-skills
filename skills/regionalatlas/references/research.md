# Regionalatlas Research

Retrieved: 2026-05-19

## Overall Description

Regionalatlas Deutschland is an official regional-statistics atlas from the statistical offices of the German federation and states. It provides thematic indicators for comparing German territories across multiple administrative levels.

The public app and catalog expose data organized by topics and indicator table groups. Live testing found 21 top-level catalog topics and 73 indicator table groups in `services.json`. Individual table groups contain multiple attribute fields with labels, long descriptions, units, available years, and metadata text.

## Data Domains

Observed and documented domains include:

- population and demographic indicators
- labor market and unemployment indicators
- education and social indicators
- economy and public finance indicators
- building, housing, and land-use indicators
- transport and environment indicators
- sustainability indicators

## Technical Shape

The bundesAPI OpenAPI wrapper documents the ArcGIS dynamic-layer query endpoint. The practical official data surface is:

- catalog metadata from the Regionalatlas web app
- ArcGIS MapServer metadata
- ArcGIS `dynamicLayer/query`

The map service reports:

- `supportsDynamicLayers: true`
- `supportedQueryFormats: JSON, geoJSON, PBF`
- `capabilities: Map,Query,Data`
- `maxRecordCount: 2000000`
- spatial reference latest WKID `25832`

## Tested Example

The refactor tested unemployment-rate data using:

- indicator `<indicator-code>`
- field `<field-code>`
- year `2024`
- region level `1`

The sample query returned Laender rows such as Schleswig-Holstein, Hamburg, and Niedersachsen. The ArcGIS response also returned `exceededTransferLimit=true`, so the CLI warns that small samples are not complete extracts.

## Best Uses

Use Regionalatlas when the user asks for official regional comparisons, especially where the region level matters. It is strong for questions like:

- Which Laender have higher unemployment rates?
- How do districts differ on a demographic or land-use indicator?
- Which municipality-level values support a local/regional claim?
- What is the official unit/source/caveat for a regional indicator?

## Less Suitable Uses

Do not use Regionalatlas as the first tool for:

- broad national statistical tables outside the atlas catalog
- non-regional time series analysis
- legal, parliamentary, or register research
- complete municipality-level exports without a clear sampling/export plan

## Sources

- Regionalatlas app: https://regionalatlas.statistikportal.de/
- Regionalatlas catalog JSON: https://regionalatlas.statistikportal.de/taskrunner/services.json
- Regionalatlas thesaurus CSV: https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv
- Statistikportal Regionalatlas page: https://www.statistikportal.de/de/karten/regionalatlas-deutschland
- Destatis Regionalatlas page: https://www.destatis.de/DE/Service/Statistik-Visualisiert/RegionalatlasAktuell.html
- ArcGIS MapServer metadata: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer?f=json
- ArcGIS dynamic-layer query endpoint: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer/dynamicLayer/query
- bundesAPI Regionalatlas OpenAPI wrapper: https://github.com/bundesAPI/regionalatlas-api
