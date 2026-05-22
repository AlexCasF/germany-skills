# Deutschlandatlas Tool

Self-contained skill and CLI bundle for the Deutschlandatlas public ArcGIS data
services.

## What changed in 2.0

- Preserved the legacy raw `table query` behavior.
- Added layer auto-discovery, because live services can use feature layers other
  than `0`.
- Added research commands: `doctor`, `tables search`, `table fields`,
  `table sample`, `table source`, `indicator dossier`, `query-builder`, and
  `explain-field`.
- Added safe output defaults: small limits, no geometry by default, and a
  structured error when broad requests exceed the safe maximum.
- Added Python and TypeScript/Node.js implementations with matching behavior.

## Quick start

```powershell
skills\deutschlandatlas\bin\deutschlandatlas-2.0.exe doctor
skills\deutschlandatlas\bin\deutschlandatlas-2.0.exe tables search --term "Arbeitslosenquote" --limit 5
skills\deutschlandatlas\bin\deutschlandatlas-2.0.exe indicator dossier --table alq_HA2023 --limit 3
```

## Sources

- Official home: https://www.deutschlandatlas.bund.de/DE/Home/home_node.html
- Official downloads: https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html
- OpenAPI wrapper: https://github.com/bundesAPI/deutschlandatlas-api
- Portal search: https://www.karto365.de/portal/sharing/rest/search?q=deutschlandatlas&f=json&num=100&start=1
