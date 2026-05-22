# Rechtsinformationen test plan

This plan checks the agent-facing command contract for all three implementations:
Go 2.0, Python, and TypeScript/Node.js.

## Test cases

| ID | Purpose | Command shape | Expected result |
| --- | --- | --- | --- |
| 1 | Root discovery | `--help` | Shows purpose, fast paths, legacy commands, research commands |
| 2 | Focused help | `documents dossier --help` | Shows evidence-bundle inputs and examples |
| 3 | Health and policy summary | `doctor` | JSON envelope with `authRequired: false`, base URL, rate limit, statistics |
| 4 | Legacy raw endpoint | `statistics` | Raw JSON counts for legislation, case law, literature, administrative directives |
| 5 | Compact search | `documents search --search-term "Bürgergeld" --limit 2` | JSON envelope with normalized items, identifiers, text matches, source URLs |
| 6 | Legacy detail endpoint | `case-law get --document-number KORE600422026` | Raw case-law JSON with document metadata and decision text fields |
| 7 | Source expansion | `source --type case-law --document-number KORE600422026` | JSON envelope with API, HTML, XML, and ZIP source URLs |
| 8 | Evidence bundle | `documents dossier --type case-law --document-number KORE600422026 --grep Revision` | JSON envelope with citation, source count, text length, and snippets |
| 9 | Text extraction | `documents text --type case-law --document-number KORE600422026 --grep Revision` | JSON envelope with extracted text preview and grep snippets |
| 10 | Structured error | `case-law get --document-number DOES_NOT_EXIST` | Non-zero exit with structured JSON error and HTTP 404 detail |

## Fixed test fixtures

- `KORE600422026` is a live case-law document from the preview API.
- `Bürgergeld` is used as a search term because it currently returns both search metadata and full source links.

## Notes

- The API is a trial service; live counts and search ranking can change.
- Tests intentionally use a small `--limit`/`size` to respect fair-use expectations.
- PowerShell can display mojibake for non-ASCII text in summarized output, but the JSON responses are UTF-8.
