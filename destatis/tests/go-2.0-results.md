# Go 2.0 test results

Implementation:

```powershell
skills\destatis\bin\destatisctl-2.0.exe
```

Environment:

- Go 1.26.2
- No personal Destatis credentials configured
- Auth fallback: `GAST/GAST`
- Test date: 2026-05-18

## Results

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help printed research guidance. |
| 2 | Pass | `table dossier --help` printed behavior and caveats. |
| 3 | Pass | `doctor` returned live logincheck and find-check details. |
| 4 | Pass | Legacy `find search` returned raw upstream JSON. |
| 5 | Pass | `search` returned compact results for `Arbeitslose`. |
| 6 | Pass | `table source` returned official source URLs for `12211-0900`. |
| 7 | Pass | `table dossier` returned a stable envelope and metadata availability/error. |
| 8 | Pass | `table sample` returned `partial` with upstream 401 caveat. |
| 9 | Pass | `variables explain` returned `partial` with upstream 401 caveat. |
| 10 | Pass | Missing table name returned structured `missing_name` error with exit 2. |

Credential redaction check: pass for all cases.
