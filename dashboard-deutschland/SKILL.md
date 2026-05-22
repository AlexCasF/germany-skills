---
name: api-dashboard-deutschland
description: Use this skill for curated Dashboard Deutschland indicators, dashboard sections, chart-ready tile data, source links, and endpoint diagnostics from the public dashboard API.
---

# Dashboard Deutschland Skill

## Purpose

Use this skill when a task needs curated high-level indicators from Dashboard Deutschland. The dashboard combines official and selected non-official sources into chart tiles on topics such as labor market, energy, prices, foreign trade, housing, mobility, finance, health, and economic activity.

Use `destatisctl` instead when the user needs deep configurable GENESIS tables or full statistical metadata beyond the dashboard tile.

## Primary Tool

Use the Go 2.0 binary first:

```powershell
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe --help
```

Alternative implementations with the same command surface:

```powershell
python skills\dashboard-deutschland\python\dashboardctl.py --help
node skills\dashboard-deutschland\typescript\dist\index.js --help
```

## Best Workflow

1. Check endpoint health and caveats:

```powershell
dashboardctl doctor
```

2. Discover dashboards or indicators:

```powershell
dashboardctl dashboards list --limit 5
dashboardctl indicator search --term "Arbeitslosigkeit" --limit 5
```

3. Inspect one indicator:

```powershell
dashboardctl indicator get --id tile_1666958835081
dashboardctl indicator source --id tile_1666958835081
```

4. Extract chart-ready points:

```powershell
dashboardctl indicator data --id tile_1666958835081 --limit 10
dashboardctl indicator data --id tile_1666958835081 --series "Arbeitslose" --limit 5
```

5. Use a dashboard dossier when starting from a theme:

```powershell
dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3
```

## Command Map

- `doctor`: reports dashboard, indicator, and GeoJSON endpoint health.
- `dashboards list`: lists dashboard sections with indicator IDs.
- `dashboard dossier`: bundles one dashboard section and a few normalized indicator summaries.
- `indicator search`: searches normalized tile titles, tags, text snippets, and source metadata.
- `indicator get`: returns parsed tile metadata, widgets, text snippets, chart series summaries, and sources.
- `indicator data`: extracts chart-ready series points from embedded Highcharts config.
- `indicator source` / `source`: returns canonical API/source URLs for a tile.
- `dashboard get`: preserved raw legacy dashboard endpoint wrapper.
- `indicators`: preserved raw legacy indicator endpoint wrapper.
- `geo`: preserved legacy GeoJSON wrapper; currently returns a structured 403 diagnostic.

## Safety Rules

- Fetch indicators by explicit ID whenever possible.
- Keep `indicator data --limit` small; the default is 10 points per series.
- Use `indicator get` before interpreting values so units, widgets, update dates, and sources are visible.
- Treat Dashboard Deutschland as curated mixed-source evidence, not a complete statistical warehouse.
- The documented GeoJSON endpoint returned `403 AccessDenied` in live tests; rely on `doctor` for current status.

## Citation Guidance

Always cite the tile's emitted source links and the Dashboard Deutschland API URL. For chart claims, include the indicator title, series name, data version date, last updated timestamp, and source organization.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
