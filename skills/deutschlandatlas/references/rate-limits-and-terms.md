# Deutschlandatlas rate limits, auth, and fair use

## Authentication

No authentication was required for the public endpoints tested on 2026-05-18:

- ArcGIS portal search
- ArcGIS service metadata
- ArcGIS layer metadata
- ArcGIS query endpoint for the sample table `alq_HA2023`

Portal results inspected during testing were marked with `access: public`.

## Rate limits

No exact published API rate limit was found in the reviewed Deutschlandatlas,
OpenAPI wrapper, or live ArcGIS metadata materials.

The CLI therefore uses conservative defaults:

- result limits default to small values
- broad requests are capped at 100 unless `--allow-large-output` is passed
- geometry is disabled by default
- `doctor` reports the absence of an exact published limit
- sample commands warn when ArcGIS returns `exceededTransferLimit=true`

## Fair-use hints

- Search and inspect metadata before querying large result sets.
- Cache table metadata and field lists because they are relatively stable.
- Avoid tight polling loops.
- Use `where`, `outFields`, and small `resultRecordCount` values.
- Request geometry only when map shapes are needed.
- Back off on slow responses, 429, or 5xx responses.

## Data interpretation terms

The official downloads page states that the data for Deutschlandatlas indicators
is available free of charge and documents that missing values in tabular files
are represented as `-9999`.

For precise statistical interpretation, use the official indicator/source notes
and not just short field names from ArcGIS metadata.

## Source links

- Official downloads: https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html
- OpenAPI wrapper: https://github.com/bundesAPI/deutschlandatlas-api
- ArcGIS portal search: https://www.karto365.de/portal/sharing/rest/search?q=deutschlandatlas&f=json&num=100&start=1
