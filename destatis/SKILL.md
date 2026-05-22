---
name: destatis
description: Use this skill for official German statistics from Destatis GENESIS-Online, including catalogue search, table and time-series codes, metadata, variables, source links, and cautious statistical data retrieval.
---

# Destatis skill

## Purpose

Use this skill to search and cite official German statistics from Destatis
GENESIS-Online. The core job is to find the right statistical table or time
series, preserve its code and metadata, and avoid misreading values without
units, dimensions, regions, and time periods.

## Service facts

- Base URL: `https://www-genesis.destatis.de/genesisWS/rest/2020`
- Web UI: `https://www-genesis.destatis.de/datenbank/online`
- Official Open Data page: `https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html`
- Auth: credential-shaped API; use `DESTATIS_USERNAME` and `DESTATIS_PASSWORD`
- Guest fallback: `GAST/GAST` works for public discovery in current tests
- License: Data Licence Germany attribution 2.0 per Destatis Open Data page
- Exact published rate limit: not found in official docs reviewed

## Use this when

- the user needs official German statistical evidence
- the user asks for Destatis, GENESIS, federal statistics, tables, or time series
- the user needs to discover table/statistic codes for a topic
- the user needs metadata, variables, dimensions, or source URLs for a statistical table
- the user needs a cautious evidence trail before making a numerical claim

## Do not use this when

- the user needs parliamentary records; prefer DIP/Bundestag tools
- the user needs regional ArcGIS indicator layers; prefer Deutschlandatlas or Regionalatlas tools
- the user needs a quick news/context source; Destatis is for statistics, not reporting
- you only have a broad concept and no table code yet; search first

## Preferred tool

Prefer the 2.0 CLI contract.

Use the local executable when available:

```powershell
skills\destatis\bin\destatis-2.0.exe doctor
```

Portable fallbacks:

```powershell
python skills\destatis\python\destatis.py doctor
node skills\destatis\typescript\dist\index.js doctor
```

If the runtime exposes the binary as `destatis`, use that shorter name.

## Auth

Prefer:

```powershell
$env:DESTATIS_USERNAME = "<username>"
$env:DESTATIS_PASSWORD = "<password>"
```

The CLI also accepts `--username` and `--password`, but prefer environment
variables so credentials do not appear in command previews.

If no credentials are configured, the CLI uses `GAST/GAST`. Current tests show
guest discovery works for `doctor` and `search`, but metadata/data endpoints may
return `401 Unauthorized`.

## Preferred workflow

1. Run `doctor` to check credential source and endpoint behavior.
2. Search with `search --term "<concept>" --limit 5`.
3. Pick an exact table or statistic code.
4. Run `table source --name <table-code>` for official source URLs.
5. Run `table dossier --name <table-code>` for metadata/caveats.
6. If full credentials are configured, use `table sample --name <table-code>` or `table dossier --sample`.
7. Use `variables explain --table <table-code>` before interpreting dimensions.
8. Preserve table code, labels, units, regions, time periods, and retrieval date in final answers.

## Best commands

Health:

```powershell
destatis doctor
```

Discovery:

```powershell
destatis search --term "Arbeitslose" --limit 5
destatis find search --param "term=Arbeitslose" --limit 5
```

Source and metadata:

```powershell
destatis table source --name 12211-0900
destatis table dossier --name 12211-0900
destatis variables explain --table 12211-0900
```

Data sample:

```powershell
destatis table sample --name 12211-0900
```

Legacy raw endpoint access remains available:

```powershell
destatis catalogue statistics --limit 10
destatis catalogue tables --param "selection=arbeit" --limit 10
destatis metadata table --param "name=12211-0900"
destatis data table --param "name=12211-0900" --param "area=all"
```

## Output expectations

Research commands return JSON envelopes with:

- `tool`
- `command`
- `status`
- `retrievedAt`
- `request`
- `summary`
- `items` where relevant
- `sources`
- `warnings`
- `nextActions`

Legacy endpoint commands return upstream JSON on success. If a protected
endpoint fails, 2.0 commands return structured errors or `partial` envelopes.

## Evidence caveats

- Never cite a statistical value without table code, unit, time period, and dimension labels.
- Guest credentials are useful for discovery but may not be enough for metadata/data retrieval.
- Current live behavior works with form `POST`; plain `GET` returned the web app during testing.
- Keep `pagelength` small while exploring.
- Avoid parallel broad requests; live `logincheck` warns about behavior above three parallel requests.
- Distinguish official Destatis statistics from derived commentary or news summaries.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
- `tests/test-plan.md`
- `MANIFEST.md`
