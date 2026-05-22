# Go 2.0 test results

Implementation: `skills/rechtsinformationen-bund/go/v2/main.go`

Build command:

```powershell
go build -o ..\..\bin\rechtsinformationenctl-2.0.exe .
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help shows purpose, fast paths, legacy endpoint commands, and research commands. |
| 2 | Pass | `documents dossier --help` explains evidence bundles and accepted input styles. |
| 3 | Pass | `doctor` reports no auth requirement, base URL, 600 requests/min/IP limit, and live statistics. |
| 4 | Pass | `statistics` returns live raw counts: legislation 2423, case-law 82473, literature 0, administrative-directive 0. |
| 5 | Pass | Compact search returns normalized items with document number, ECLI, first text match, and source URLs. |
| 6 | Pass | `case-law get` returns raw case-law JSON including `documentNumber`, `ecli`, `caseFacts`, `decisionGrounds`, and `tenor`. |
| 7 | Pass | `source` returns API, HTML, XML, and ZIP source URLs for `KORE600422026`. |
| 8 | Pass | `documents dossier` returns citation, source count 4, text length 20681, and 10 snippets for `Revision`. |
| 9 | Pass | `documents text` extracts source text length 20681 and 10 snippets for `Revision`. |
| 10 | Pass | Invalid document number exits non-zero with structured JSON error for HTTP 404. |

## Bug found and fixed

The first Go run exposed a panic in HTML stripping. The regex used a backreference, which Go's regexp engine does not support. It was replaced with a Go-safe expression:

```go
regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
```

After rebuilding, dossier and text extraction passed.
