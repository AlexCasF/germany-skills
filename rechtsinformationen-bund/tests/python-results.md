# Python test results

Implementation: `skills/rechtsinformationen-bund/python/rechtsinformationenctl.py`

Validation command:

```powershell
python -m py_compile skills\rechtsinformationen-bund\python\rechtsinformationenctl.py
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help is available and lists the same major command families as Go 2.0. |
| 2 | Pass | `documents dossier --help` explains evidence-bundle behavior and examples. |
| 3 | Pass | `doctor` reports no auth requirement, base URL, 600 requests/min/IP limit, and live statistics. |
| 4 | Pass | `statistics` returns live raw counts: legislation 2423, case-law 82473, literature 0, administrative-directive 0. |
| 5 | Pass | Compact search returns normalized record fields, text-match count, first match, and source URLs after patching SearchResult handling. |
| 6 | Pass | `case-law get` returns raw case-law JSON with the expected document metadata fields. |
| 7 | Pass | `source` returns API, HTML, XML, and ZIP source URLs. |
| 8 | Pass | `documents dossier` returns source count 4, citation, text length 20681, and 10 snippets for `Revision`. |
| 9 | Pass | `documents text` returns text length 20681 and 10 snippets for `Revision`. |
| 10 | Pass | Invalid document number exits non-zero with structured JSON error for HTTP 404. |

## Patch note

The first compact search summary exposed only the `SearchResult` wrapper. The Python implementation now unwraps `item`, preserves `textMatches`, and attaches source links so agents can act on search results directly.
