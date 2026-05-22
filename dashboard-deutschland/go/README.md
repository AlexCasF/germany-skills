# Dashboard Deutschland Go CLI

## Versions

- `v1/main.go`: preserved legacy raw endpoint wrapper.
- `v2/main.go`: refactored research-oriented CLI.

## Build

```powershell
cd skills\dashboard-deutschland\go\v2
go build -o ..\..\bin\dashboardctl-2.0.exe .
```

## Smoke Commands

```powershell
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe doctor
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe dashboards list --limit 3
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe indicator data --id tile_1666958835081 --limit 3
```
