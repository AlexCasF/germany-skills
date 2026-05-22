# Python Test Results

Runtime:

```text
python skills\bundesrat-live\python\bundesrat-live.py
```

## Result Summary

| # | Test | Exit | Result |
| --- | --- | --- | --- |
| 1 | Root help | 0 | Pass |
| 2 | Doctor | 0 | Pass |
| 3 | News search | 0 | Pass |
| 4 | News page | 0 | Pass |
| 5 | Dates | 0 | Pass |
| 6 | Members search | 0 | Pass |
| 7 | Member dossier | 0 | Pass |
| 8 | Plenum compact | 0 | Pass |
| 9 | Plenum current | 0 | Pass |
| 10 | Plenum next | 0 | Pass |

## Extra Smoke Checks

| Test | Exit | Result |
| --- | --- | --- |
| `votes summary` | 0 | Pass |
| `presidium --limit 2` | 0 | Pass |

## Notes

- All JSON cases parsed successfully with `ConvertFrom-Json`.
- The Python version uses only the standard library.
- Output shape matches the Go implementation closely enough for shared agent usage.
