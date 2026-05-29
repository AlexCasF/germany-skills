# Regionalatlas Notes

## What The API Provides

Regionalatlas Deutschland provides official regional indicators from the statistical offices of the German federation and states. The public atlas catalog currently exposes 21 top-level topic areas and 73 indicator table groups in the tested catalog, with many individual attribute fields per table.

The service is useful for territorial comparisons across:

- Laender
- Regierungsbezirke/statistical regions
- Kreise and kreisfreie Staedte
- Gemeinden/Gemeindeverbaende

## Important Endpoints

- Catalog JSON: https://regionalatlas.statistikportal.de/taskrunner/services.json
- Thesaurus CSV: https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv
- ArcGIS MapServer metadata: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer?f=json
- Dynamic-layer query endpoint: https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer/dynamicLayer/query
- Interactive atlas: https://regionalatlas.statistikportal.de/

## Query Pattern

The practical query pattern is an ArcGIS `dynamicLayer/query` request using a generated `layer` JSON object. The CLI builds a queryTable layer that joins `verwaltungsgrenzen_gesamt` to an indicator table such as `ai008_1_5`.

Example conceptual join:

```sql
SELECT *
FROM verwaltungsgrenzen_gesamt
LEFT OUTER JOIN ai008_1_5
  ON ags = ags2 and jahr = jahr2
WHERE typ = 1
  AND jahr = 2024
  AND (jahr2 = 2024 OR jahr2 IS NULL)
```

## Common Pitfalls

- The `layer` parameter is JSON and easy to corrupt through shell quoting.
- Windows PowerShell can strip JSON quotes when passing a raw layer string to native commands; use `--layer-file`.
- PowerShell may write UTF-8 files with a BOM; the CLIs strip this for `--layer-file`.
- ArcGIS may return `exceededTransferLimit=true`; do not treat a sample as a full extract.
- `maxRecordCount` is very high, so accidental broad outputs can be enormous.
- A field code such as `<field-code>` needs metadata to be interpretable; inspect units and definitions.
- Available years vary by indicator.

## Output Guidance

When summarizing Regionalatlas results, include:

- indicator code and title
- field code and title
- unit
- year
- region level
- sample/limit status
- whether `exceededTransferLimit` was true
- source URLs emitted by the CLI
