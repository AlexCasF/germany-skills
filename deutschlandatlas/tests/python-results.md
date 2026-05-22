# Python test results

Run date: 2026-05-18

Command prefix:

`python skills\deutschlandatlas\python\deutschlandatlas.py`

| # | Test | Result | Notes |
| --- | --- | --- | --- |
| 1 | Root help | Pass | Exit 0, text help. |
| 2 | `table sample --help` | Pass | Exit 0. |
| 3 | `doctor` | Pass | Exit 0, public endpoints reachable. |
| 4 | Legacy raw query | Pass | Exit 0, raw ArcGIS JSON returned. |
| 5 | `tables search --term Apotheken --limit 3` | Pass | Exit 0. |
| 6 | `table fields --table alq_HA2023` | Pass | Exit 0, field list returned. |
| 7 | `table source --table alq_HA2023` | Pass | Exit 0. |
| 8 | `indicator dossier --table alq_HA2023 --limit 2` | Pass | Exit 0. |
| 9 | `query-builder --region Berlin` | Pass | Exit 0. |
| 10 | Unsafe broad sample | Pass | Exit 2, structured safe-limit error. |

Summary: 10/10 passed.
