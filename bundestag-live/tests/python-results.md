# Python Test Results

Run from repository root with `python skills\bundestag-live\python\bundestag-live.py`.

| # | Test | Result | Notes |
| --- | --- | --- | --- |
| 1 | Root help | Pass | Exit `0`, text help returned. |
| 2 | Doctor | Pass | Exit `0`, JSON envelope, `status: ok`, endpoint health returned. |
| 3 | Plenary agenda | Pass | Exit `0`, bounded agenda JSON with article next action. |
| 4 | Member search | Pass | Exit `0`, found Philipp Amthor with profile and XML source URLs. |
| 5 | Member dossier | Pass | Exit `0`, returned biography and disclosure snippets for `Tätigkeiten`. |
| 6 | Committee search | Pass | Exit `0`, found `a11` Arbeit und Soziales. |
| 7 | Committee dossier | Pass | Exit `0`, returned bounded members, news, task text, and committee source page. |
| 8 | Article XML | Pass | Exit `0`, returned article metadata and `Meinungsfreiheit` snippet. |
| 9 | Article page | Pass | Exit `0`, public page extraction returned snippets. |
| 10 | Video feed | Pass | Exit `0`, returned three stream groups and media warnings. |

## Observations

- The Python implementation has no third-party dependencies.
- All JSON test cases parsed successfully.
- Output shape matches the Go 2.0 implementation closely enough for agent-side substitution.
