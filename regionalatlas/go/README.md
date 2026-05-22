# Regionalatlas Go CLI

## Versions

- `v1/main.go`: preserved legacy raw dynamic-layer wrapper.
- `v2/main.go`: refactored research-oriented CLI.

## Build

```powershell
cd skills\regionalatlas\go\v2
go build -o ..\..\bin\regionalatlasctl-2.0.exe .
```

## Smoke Commands

```powershell
skills\regionalatlas\bin\regionalatlasctl-2.0.exe doctor
skills\regionalatlas\bin\regionalatlasctl-2.0.exe indicators search --term "Arbeitslosenquote" --limit 3
skills\regionalatlas\bin\regionalatlasctl-2.0.exe sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 3
```
