# Python test results

Implementation: `skills/abgeordnetenwatch/python/abgeordnetenwatch.py`

Validation command:

```powershell
python -m py_compile skills\abgeordnetenwatch\python\abgeordnetenwatch.py
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help lists the same research and legacy command groups as Go 2.0. |
| 2 | Pass | `politicians dossier --help` shows dossier purpose and examples. |
| 3 | Pass | `doctor` reports no auth requirement, API version 2.8.2, CC0 license, docs links, and no published exact rate limit. |
| 4 | Pass | `parliaments list --limit 1` returns raw upstream JSON with one returned item and total 18. |
| 5 | Pass | Search returns compact politician data with source URLs and next actions. |
| 6 | Pass | Exact politician lookup by ID returns raw upstream JSON. |
| 7 | Pass | Profile-page extraction returns metadata, profile ID, text length, and snippets. |
| 8 | Pass | Dossier returns mandate count 1, side-job count 1, income sum 16690.87, and page snippets. |
| 9 | Pass | Side-job join returns disclosed side-job evidence through mandate ID. |
| 10 | Pass | Invalid ID exits non-zero with structured JSON error for upstream HTTP 500. |

## Patch note

The first dossier test exposed a Windows console encoding issue. The Python CLI now explicitly configures UTF-8 stdout/stderr.
