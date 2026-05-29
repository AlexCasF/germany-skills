# Regionalatlas Rate Limits, Auth, And Terms

Retrieved: 2026-05-19

## Authentication

No authentication was required for the public endpoints tested:

- `https://regionalatlas.statistikportal.de/taskrunner/services.json`
- `https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv`
- `https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer?f=json`
- `https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer/dynamicLayer/query`

## Rate Limits

No exact public request-per-second or request-per-day limit was found in the reviewed official/API materials.

The ArcGIS service advertises `maxRecordCount: 2000000`, which is not a fair-use invitation. It means accidental broad queries can become enormous. One earlier live test showed how easily large outputs can happen, so the CLI defaults to small samples and refuses limits above 100 unless `--allow-large-output` is explicitly passed.

## Fair-Use Hints Implemented In The CLI

- Use catalog discovery before data queries.
- Default `sample` limit is 10.
- Default region level is `1` (Laender), not municipalities.
- Geometry is disabled unless requested.
- Broad limits require `--allow-large-output`.
- `doctor` surfaces the absence of exact rate-limit guidance.
- `exceededTransferLimit=true` is exposed as a warning.
- Raw dynamic-layer access is preserved but steered through safer helpers.

## Licensing And Reuse Notes

The Statistikportal Open Data page points to official open-data downloads and licensing context, including Datenlizenz Deutschland 2.0 for statistical data. Destatis maps/geodata pages provide additional context for map and geodata offerings. Because the Regionalatlas combines statistical data, metadata, and map/geodata presentation, cite the exact source URLs emitted by the CLI and preserve upstream attribution in research outputs.

## Sources

- Statistikportal Open Data: https://www.statistikportal.de/de/open-data
- Statistikportal Regionalatlas page: https://www.statistikportal.de/de/karten/regionalatlas-deutschland
- Destatis maps and geodata: https://www.destatis.de/DE/Service/OpenData/karten-geodaten.html
- Destatis Regionalatlas page: https://www.destatis.de/DE/Service/Statistik-Visualisiert/RegionalatlasAktuell.html
- ArcGIS MapServer metadata: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer?f=json
- ArcGIS dynamic-layer query endpoint: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer/dynamicLayer/query
