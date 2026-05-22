# Go implementation

## Versions

| Version | Path | Notes |
| --- | --- | --- |
| 1.x | `v1/main.go` | Original thin endpoint wrapper. |
| 2.0 | `v2/main.go` | Current agent-friendly V2 CLI. |

## Build

```powershell
cd skills\bundestag-lobbyregister\go\v2
go build -o ..\..\bin\lobbyregisterctl-2.0.exe .
```

## Auth

Set `LOBBYREGISTER_API_KEY` before live V2 calls. The CLI also accepts
`--apikey`, but output redacts key material.
