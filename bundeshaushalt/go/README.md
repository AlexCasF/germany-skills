# Go Bundeshaushalt CLI

Build the 2.0 CLI:

```powershell
cd skills\bundeshaushalt\go\v2
go build -o ..\..\bin\bundeshaushaltctl-2.0.exe .
```

Run from the repository root:

```powershell
skills\bundeshaushalt\bin\bundeshaushaltctl-2.0.exe doctor
skills\bundeshaushalt\bin\bundeshaushaltctl-2.0.exe budget tree --year 2026 --account expenses --limit 5
```

The legacy generated wrapper is preserved in `go/v1/main.go` and `bin/bundeshaushaltctl-legacy.exe`.
