# Go 2.0 test results

Run date: 2026-05-18

Command prefix:

`skills\deutschlandatlas\bin\deutschlandatlas-2.0.exe`

| # | Test | Result | Notes |
| --- | --- | --- | --- |
| 1 | Root help | Pass | Exit 0, text help. |
| 2 | `table sample --help` | Pass | Exit 0, explains safe samples and geometry. |
| 3 | `doctor` | Pass | Exit 0, portal reachable, sample service reachable, no auth required. |
| 4 | Legacy raw query | Pass | Exit 0, raw ArcGIS JSON returned from explicit layer 5. |
| 5 | `tables search --term Apotheken --limit 3` | Pass | Exit 0, compact portal results. |
| 6 | `table fields --table alq_HA2023` | Pass | Exit 0, likely indicator field `alq`. |
| 7 | `table source --table alq_HA2023` | Pass | Exit 0, source URLs emitted. |
| 8 | `indicator dossier --table alq_HA2023 --limit 2` | Pass | Exit 0, metadata, fields, sample, warnings, sources. |
| 9 | `query-builder --region Berlin` | Pass | Exit 0, URL built without fetching. |
| 10 | Unsafe broad sample | Pass | Exit 2, structured safe-limit error. |

Summary: 10/10 passed.
