---
name: regionalatlas
description: Use this skill for official Regionalatlas Deutschland indicators across Laender, Regierungsbezirke/statistical regions, Kreise, kreisfreie Staedte, and Gemeinden through the Regionalatlas catalog and ArcGIS dynamic-layer query endpoint.
---

# Regionalatlas Skill

## Purpose

Use this skill when a task needs official regional statistical indicators for Germany at administrative levels below or across the national level.

The Regionalatlas is useful for comparing regions by population, labor market, education, land use, economy, housing, transport, environment, and related topics. It is not a general-purpose full GENESIS table client; use `destatis` when the user needs broader statistical tables outside the atlas/map context.

## Primary Tool

Use the Go binary first:

```powershell
skills\regionalatlas\bin\regionalatlas.exe --help
```

Alternative implementations with the same command surface:

```powershell
python skills\regionalatlas\python\regionalatlas.py --help
node skills\regionalatlas\typescript\dist\index.js --help
```

## Best Workflow

1. Check service status and fair-use hints:

```powershell
regionalatlas doctor
```

2. Search the indicator catalog:

```powershell
regionalatlas indicators search --term "Indikator" --limit 5
```

3. Inspect fields, units, available years, and metadata:

```powershell
regionalatlas fields --indicator <indicator-code>
regionalatlas explain-field --indicator <indicator-code> --field <field-code> --grep Quelle
```

4. Fetch a small bounded sample:

```powershell
regionalatlas sample --indicator <indicator-code> --field <field-code> --year 2024 --region-level 1 --limit 5
```

5. Use `dossier` when you need an evidence bundle:

```powershell
regionalatlas dossier --indicator <indicator-code> --field <field-code> --year 2024 --region-level 1 --limit 5
```

## Command Map

- `doctor`: checks catalog/API health, auth status, rate-limit findings, and fair-use hints.
- `indicators list`: lists catalog indicators with safe limits.
- `indicators search`: searches indicator titles, attributes, units, and metadata.
- `indicator get`: returns one indicator's metadata and fields.
- `fields`: shows available fields, units, years, and region-level labels.
- `sample`: runs a small safe ArcGIS dynamic-layer query.
- `source`: prints canonical source URLs and citation hints.
- `dossier`: combines metadata, source URLs, field snippets, and a small sample.
- `query-builder`: builds the encoded ArcGIS request URL without fetching.
- `explain-field`: extracts field metadata snippets, optionally filtered by `--grep`.
- `query`: preserves the raw raw dynamic-layer query escape hatch.

## Safety Rules

- Default to `--region-level 1` unless the user explicitly needs districts or municipalities.
- Never request municipality-level full output casually; use small `--limit` values first.
- Geometry is off by default. Only use `--geometry true` for map-shape work.
- Limits above 100 require `--allow-large-output`.
- Treat `exceededTransferLimit=true` as a warning that the result is only a partial sample.
- Prefer `--layer-file` for raw `query` on Windows because shell quoting can corrupt JSON.

## Region Levels

- `1`: Laender
- `2`: Regierungsbezirke/statistical regions
- `3`: Kreise and kreisfreie Staedte
- `5`: Gemeinden/Gemeindeverbaende

## Citation Guidance

Always cite the relevant source URLs emitted by the CLI. For statistical interpretation, include the field label, unit, year, region level, and any metadata caveats returned by `fields`, `explain-field`, or `dossier`.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
