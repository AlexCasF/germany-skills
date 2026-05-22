# Go 2.0 Test Results

Run date: 2026-05-19

## Build

```powershell
go build -o ..\..\bin\regionalatlasctl-2.0.exe .
```

Result: passed.

## Matrix

| Test | Exit | Expected | Result |
| --- | ---: | ---: | --- |
| root help | 0 | 0 | pass |
| sample help | 0 | 0 | pass |
| doctor | 0 | 0 | pass |
| raw query compatibility | 0 | 0 | pass |
| indicator search | 0 | 0 | pass |
| indicator get | 0 | 0 | pass |
| source metadata | 0 | 0 | pass |
| dossier | 0 | 0 | pass |
| explain field grep | 0 | 0 | pass |
| large output guard | 2 | 2 | pass |

Summary: 10/10 passed.
