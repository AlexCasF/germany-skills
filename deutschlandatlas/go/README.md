# Deutschlandatlas Go CLI

## Versions

- `v1/main.go`: legacy wrapper source, preserved unchanged after migration.
- `v2/main.go`: refactored research-oriented CLI.

## Build

```powershell
cd skills\deutschlandatlas\go\v2
go build -o ..\..\bin\deutschlandatlasctl-2.0.exe .
```

## Smoke test

```powershell
skills\deutschlandatlas\bin\deutschlandatlasctl-2.0.exe doctor
skills\deutschlandatlas\bin\deutschlandatlasctl-2.0.exe table sample --table alq_HA2023 --limit 2
```
