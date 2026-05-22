# Go Tagesschau CLI

Build the 2.0 CLI:

```powershell
cd skills\tagesschau\go\v2
go build -o ..\..\bin\tagesschauctl-2.0.exe .
```

Run from the repository root:

```powershell
& .\skills\tagesschau\bin\tagesschauctl-2.0.exe doctor
& .\skills\tagesschau\bin\tagesschauctl-2.0.exe search --text Bundestag --limit 5
```

The legacy generated wrapper is preserved in `go/v1/main.go` and `bin/tagesschauctl-legacy.exe`.
