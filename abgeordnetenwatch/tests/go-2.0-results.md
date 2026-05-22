# Go 2.0 test results

Implementation: `skills/abgeordnetenwatch/go/v2/main.go`

Build command:

```powershell
go build -o ..\..\bin\abgeordnetenwatchctl-2.0.exe .
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help explains the transparency-data purpose, false friends, fast paths, legacy commands, and research commands. |
| 2 | Pass | `politicians dossier --help` explains ID/name/URL inputs, grep, limits, and cross-check caveats. |
| 3 | Pass | `doctor` reports no auth requirement, API version 2.8.2, CC0 license, docs links, and no published exact rate limit. |
| 4 | Pass | `parliaments list --limit 1` returns raw upstream JSON with one returned item and total 18. |
| 5 | Pass | `politicians search --name "Alice Weidel" --limit 2` returns a compact source-rich item. |
| 6 | Pass | `politicians get --id 108379` returns raw upstream politician JSON. |
| 7 | Pass | `politicians page --id 108379 --grep Nebentätigkeiten` extracts page metadata, ID hints, and snippets. |
| 8 | Pass | `politicians dossier --id 108379 --grep Nebentätigkeiten --limit 5` returns one mandate, one side job, profile snippets, and warnings. |
| 9 | Pass | `sidejobs for-politician --id 108379 --limit 3` joins mandates to side jobs and returns income sum 16690.87. |
| 10 | Pass | Invalid ID exits non-zero with structured JSON error for upstream HTTP 500. |

## Notes

- The key improvement is native profile-page extraction, avoiding ad hoc `curl` for pages emitted by the API.
- The side-job path is mandate-oriented: politician ID -> mandates -> sidejobs by mandate ID.
