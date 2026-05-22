# Go 2.0 Test Results

Run date: 2026-05-19

## Build

```powershell
go build -o ..\bin\dashboard-deutschland-2.0.exe .
```

Result: passed.

## Matrix

| Test | Exit | Expected | Result |
| --- | ---: | ---: | --- |
| root help | 0 | 0 | pass |
| indicator data help | 0 | 0 | pass |
| doctor degraded geo | 0 | 0 | pass |
| legacy dashboard get | 0 | 0 | pass |
| legacy indicators | 0 | 0 | pass |
| dashboards list | 0 | 0 | pass |
| indicator get | 0 | 0 | pass |
| indicator data series | 0 | 0 | pass |
| dashboard dossier | 0 | 0 | pass |
| geo failure diagnostic | 1 | 1 | pass |

Summary: 10/10 passed.
