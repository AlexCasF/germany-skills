# TypeScript/Node.js test results

Implementation: `skills/rechtsinformationen-bund/typescript/src/index.ts`

Build commands:

```powershell
cd skills\rechtsinformationen-bund\typescript
npm install
npm run build
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help shows purpose, fast paths, legacy commands, and research commands. |
| 2 | Pass | `documents dossier --help` shows evidence-bundle behavior and examples. |
| 3 | Pass | `doctor` reports no auth requirement, base URL, 600 requests/min/IP limit, and live statistics. |
| 4 | Pass | `statistics` returns live raw counts: legislation 2423, case-law 82473, literature 0, administrative-directive 0. |
| 5 | Pass | Compact search returns normalized items with document number, ECLI, first text match, and source URLs. |
| 6 | Pass | `case-law get` returns raw case-law JSON with expected top-level fields. |
| 7 | Pass | `source` returns API, HTML, XML, and ZIP source URLs. |
| 8 | Pass | `documents dossier` returns source count 4, citation, text length 20681, and 10 snippets for `Revision`. |
| 9 | Pass | `documents text` returns text length 20681 and 10 snippets for `Revision`. |
| 10 | Pass | Invalid document number exits non-zero with structured JSON error for HTTP 404. |

## Implementation note

This version uses Node's built-in `https` module instead of global `fetch`, avoiding the fetch instability observed during earlier Windows CLI testing.
