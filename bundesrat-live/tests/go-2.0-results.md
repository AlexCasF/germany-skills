# Go 2.0 Test Results

Runtime:

```text
skills\bundesrat-live\bin\bundesratctl-2.0.exe
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
- `doctor` returned `status: ok`.
- The tool found the expected current member match for `Özdemir`.
- Public-page extraction for the sample Bundesrat news URL returned matching `Merkel` snippets.
