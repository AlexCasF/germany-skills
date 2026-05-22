---
name: deutschlandatlas
description: Use this skill for Deutschlandatlas regional indicator data exposed through public ArcGIS MapServer services.
---

# Deutschlandatlas skill

## Purpose

Use this skill to discover, inspect, and query Deutschlandatlas indicator tables.
The data covers regional living conditions in Germany, including work, housing,
mobility, health care, education, infrastructure, demography, and public-service
access indicators.

## When to use

Use this skill when the user asks for:

- regional indicator values from the Deutschlandatlas
- comparisons between German municipalities, Gemeindeverbaende, districts, or cities
- map/table data behind a Deutschlandatlas card
- historical or year-specific atlas indicators
- field discovery for an ArcGIS indicator table

Do not use this as a general statistics API. If the user needs broader official
statistical tables, use Destatis/Regionalatlas first.

## Tool location

Primary Go binary:

`skills/deutschlandatlas/bin/deutschlandatlas-2.0.exe`

Alternative implementations:

- Python: `python skills/deutschlandatlas/python/deutschlandatlas.py`
- TypeScript/Node.js: `node skills/deutschlandatlas/typescript/dist/index.js`

## Best workflow

1. Run `deutschlandatlas doctor` if endpoint health or usage constraints matter.
2. Search for the table name with `tables search --term`.
3. Inspect fields with `table fields --table`.
4. Fetch a bounded sample with `table sample --table --limit 5`.
5. Build a compact bundle with `indicator dossier --table`.
6. Only request geometry if the user explicitly needs map-ready shapes.

## Commands

Discovery:

`deutschlandatlas tables search --term "Arbeitslosenquote" --limit 5`

Field inspection:

`deutschlandatlas table fields --table alq_HA2023`

Safe sample:

`deutschlandatlas table sample --table alq_HA2023 --fields name,alq --limit 5`

Dossier:

`deutschlandatlas indicator dossier --table alq_HA2023 --limit 3`

Source/citation URLs:

`deutschlandatlas table source --table alq_HA2023`

Build a query without fetching:

`deutschlandatlas query-builder --table alq_HA2023 --region Berlin --fields name,alq --limit 3`

Raw upstream query:

`deutschlandatlas table query --table alq_HA2023 --layer 5 --param where=1=1 --param outFields=* --limit 2`

## Important behavior

- JSON is the default output.
- Research commands return a stable envelope with `status`, `summary`, `items`,
  `sources`, `warnings`, and `nextActions`.
- `table query` returns raw upstream ArcGIS JSON for compatibility.
- Version 2.0 auto-discovers the feature layer. This matters because live
  services such as `alq_HA2023` use layer `5`, not layer `0`.
- Use `--layer 0` or `--legacy-layer-zero` only for explicit legacy probing.
- Result counts default small and are capped at 100 unless
  `--allow-large-output` is passed.
- `returnGeometry=false` is the default.

## Interpretation cautions

- Table schemas vary. Always inspect `table fields` before interpreting values.
- The portal snippet often carries the clearest unit/year description.
- Official download notes say missing tabular values are represented as `-9999`.
- ArcGIS can return `exceededTransferLimit=true`; do not treat a broad sample as
  a complete extract.
- No exact published API rate limit was found in the reviewed materials. Keep
  requests small, cache metadata, and avoid tight polling.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
