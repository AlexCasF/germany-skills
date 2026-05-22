# Destatis test plan

This plan checks the agent-facing command contract for all three
implementations: Go 2.0, Python, and TypeScript/Node.js.

The test run used no personal `DESTATIS_USERNAME` / `DESTATIS_PASSWORD`, so the
expected auth mode is `GAST/GAST` fallback.

## Test cases

| ID | Purpose | Command shape | Expected result |
| --- | --- | --- | --- |
| 1 | Root discovery | `--help` | Shows purpose, fast paths, auth, legacy commands, and research commands |
| 2 | Focused help | `table dossier --help` | Explains metadata/sample behavior and guest caveat |
| 3 | Doctor | `doctor` | JSON envelope with credential source, live logincheck, find check, docs, and warnings |
| 4 | Backward-compatible raw command | `find search --param term=Arbeitslose --limit 3` | Raw upstream JSON from `find/find` |
| 5 | Safe search alias | `search --term Arbeitslose --limit 3` | Compact normalized results and next actions |
| 6 | Source metadata | `table source --name 12211-0900` | Official source URLs and citation metadata |
| 7 | Table dossier | `table dossier --name 12211-0900` | Summary, sources, warnings, and metadata availability/error |
| 8 | Data sample safety | `table sample --name 12211-0900` | `partial` envelope if guest auth gets 401, no crash |
| 9 | Variable explanation | `variables explain --table 12211-0900` | `partial` envelope if guest auth gets 401, no crash |
| 10 | Error safety | `table source` | Non-zero exit with structured `missing_name` error |

## Fixed test fixture

- Search term: `Arbeitslose`
- Table code: `12211-0900`
- Guest credential fallback: `GAST/GAST`

## Notes

- Current live service behavior requires form `POST` for useful JSON responses.
- Guest credentials worked for discovery but not for metadata/data in this test pass.
- Personal GENESIS credentials should unlock more complete metadata and data tests.
