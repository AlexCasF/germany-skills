# Go implementation

## Versions

| Version | Path | Notes |
| --- | --- | --- |
| 1.x | `v1/main.go` | Original thin endpoint wrapper. |
| 2.0 | `v2/main.go` | Current agent-friendly GENESIS CLI. |

## Build

```powershell
cd skills\destatis\go\v2
go build -o ..\..\bin\destatisctl-2.0.exe .
```

## Auth

Set `DESTATIS_USERNAME` and `DESTATIS_PASSWORD` for full metadata/data access.
If unset, the CLI uses `GAST/GAST` for public discovery.
