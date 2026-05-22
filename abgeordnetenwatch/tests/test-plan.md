# Abgeordnetenwatch test plan

This plan checks the agent-facing command contract for all three implementations:
Go 2.0, Python, and TypeScript/Node.js.

## Test cases

| ID | Purpose | Command shape | Expected result |
| --- | --- | --- |
| 1 | Root discovery | `--help` | Shows purpose, fast paths, legacy commands, and research commands |
| 2 | Focused help | `politicians dossier --help` | Shows dossier inputs, examples, and evidence caveats |
| 3 | Health and policy summary | `doctor` | JSON envelope with no-auth status, license, docs, result-limit guidance, and rate-limit caveat |
| 4 | Backward-compatible raw command | `parliaments list --limit 1` | Raw upstream JSON with `meta.result.count = 1` |
| 5 | Search/list safe default | `politicians search --name "Alice Weidel" --limit 2` | Compact JSON result with API URL, profile URL, party, and next actions |
| 6 | Detail/get | `politicians get --id 108379` | Raw upstream politician JSON for Alice Weidel |
| 7 | Source/page expansion | `politicians page --id 108379 --grep Nebentätigkeiten` | Extracted profile title, canonical URL, shortlink, text length, and snippets |
| 8 | Dossier | `politicians dossier --id 108379 --grep Nebentätigkeiten --limit 5` | Evidence bundle with politician record, mandates, side jobs, profile snippets, warnings |
| 9 | Related evidence join | `sidejobs for-politician --id 108379 --limit 3` | Mandate-to-sidejob join with disclosed side-job income and source URLs |
| 10 | Error safety | `politicians get --id 999999999` | Non-zero exit with structured JSON error for upstream HTTP 500 |

## Fixed test fixtures

- Politician ID `108379` is Alice Weidel.
- The public profile URL is `https://www.abgeordnetenwatch.de/profile/alice-weidel`.
- During testing, mandate ID `68967` led to side-job ID `20846`.

## Notes

- The API returns HTTP 500 for some invalid entity IDs instead of a clean 404.
- PowerShell can display mojibake for non-ASCII text in some Python output views, but the CLI writes UTF-8 JSON.
- Search ranking and side-job data can change as the live dataset updates.
