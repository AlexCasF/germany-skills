# Destatis tool

This skill folder contains the CLI and guidance for official German statistics
from Destatis GENESIS-Online.

GENESIS-Online provides a large catalogue of official statistical tables, time
series, variables, values, metadata, maps, charts, cubes, and result files. The
most useful agent workflows are discovery-first: search for a concept, inspect
table/statistic codes, then request metadata or data with small bounds.

## Implementations

| Implementation | Path | Notes |
| --- | --- | --- |
| Go 2.0 | `go/main.go` | Agent-friendly CLI with env auth, doctor, search alias, source, dossier, sample, and variable helpers. |
| Python | `python/destatis.py` | Python parity implementation. |
| TypeScript / Node.js | `typescript/src/index.ts` | TypeScript source compiled to Node.js JavaScript. |

## Runtime data

Prefer:

```powershell
$env:DESTATIS_USERNAME = "<username>"
$env:DESTATIS_PASSWORD = "<password>"
```

If these are unset, the 2.0 CLIs use `GAST/GAST` for public discovery. Live
testing showed `GAST/GAST` works for `logincheck` and `find/find`, but
catalogue/metadata/data endpoints can return `401 Unauthorized`. Use a personal
GENESIS account for full metadata/data retrieval.

## Primary sources

- `https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html`
- `https://www.destatis.de/EN/Service/OpenData/api-webservice.html`
- `https://www.destatis.de/DE/Service/Kontakt/Genesis/Servicekontakt-GENESIS.html`
- `https://www-genesis.destatis.de/datenbank/online`
- `https://github.com/bundesAPI/destatis-api`
