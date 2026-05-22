# Destatis research notes

## What the API provides

Destatis GENESIS-Online provides official German statistical data through a
large catalogue of tables, statistics, variables, values, time series, cubes,
charts, maps, result files, metadata, and data endpoints.

Use it for:

- official German statistical evidence
- table and statistic discovery
- metadata and variable interpretation
- time series and table retrieval
- source-aware statistical citations

## Current endpoint behavior

The local OpenAPI spec is for:

```text
https://www-genesis.destatis.de/genesisWS/rest/2020
```

Although the spec describes `GET`, live testing showed the current service
works as form `POST` for `helloworld/logincheck` and `find/find`. `GET`
returned the JavaScript web application rather than JSON during this pass.

The 2.0 CLI therefore uses form `POST` by default.

## Guest and personal credentials

With no credentials configured, the CLI uses `GAST/GAST`.

Observed behavior on 2026-05-18:

- `helloworld/logincheck` worked with `GAST/GAST`
- `find/find` worked with `GAST/GAST`
- catalogue, metadata, and data endpoints tested returned `401 Unauthorized`
  with `GAST/GAST`

For full metadata and data retrieval, configure:

```powershell
$env:DESTATIS_USERNAME = "<username>"
$env:DESTATIS_PASSWORD = "<password>"
```

## Common research workflow

1. Run `doctor`.
2. Search with `search --term "<concept>" --limit 5`.
3. Pick a table/statistic code.
4. Run `table source --name <code>`.
5. Run `table dossier --name <code>`.
6. If personal credentials are configured, add `--sample` or use `table sample`.
7. Preserve labels, units, dimensions, time periods, and table codes in the final answer.

## Interpretation caveats

Statistical values are unsafe without metadata. Always capture:

- table or time-series code
- statistic code
- variable/dimension labels
- unit
- reference period
- region level
- source date or retrieval date
- access caveat if only guest credentials were used

## Sources

- `https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html`
- `https://www.destatis.de/EN/Service/OpenData/api-webservice.html`
- `https://www.destatis.de/DE/Service/Kontakt/Genesis/Servicekontakt-GENESIS.html`
- `https://www-genesis.destatis.de/datenbank/online`
- `https://github.com/bundesAPI/destatis-api`
