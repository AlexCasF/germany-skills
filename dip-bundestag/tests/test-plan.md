# DIP CLI 10-case test plan

Run these tests against each implementation:

- Go 2.0: `skills/dip-bundestag/bin/dip-bundestag-2.0.exe`
- Python: `python skills/dip-bundestag/python/dip-bundestag.py`
- TypeScript/Node.js: `node skills/dip-bundestag/typescript/dist/index.js`

Load `DIP_API_KEY` from the local environment before running authenticated
tests. Do not print the key in test logs.

## Tests

| Test | Purpose | Command shape |
| --- | --- | --- |
| 1 | Root help is a research guide. | `<cli> --help` |
| 2 | Command help explains dossier behavior. | `<cli> person dossier --help` |
| 3 | Doctor checks auth and endpoint health without printing the key. | `<cli> doctor` |
| 4 | Legacy list command still works with `--param`. | `<cli> person list --param "f.person=Gauweiler" --limit 1` |
| 5 | Compact person search works. | `<cli> person search --name "Gauweiler" --limit 3` |
| 6 | Legacy exact person lookup works. | `<cli> person get --id 760` |
| 7 | Source expansion works from a document number. | `<cli> source --type plenarprotokoll --document-number "20/139"` |
| 8 | Person dossier bundles official record and activities. | `<cli> person dossier --id 760 --limit 3` |
| 9 | Plenary text grep returns bounded snippets. | `<cli> plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"` |
| 10 | Error handling is structured and redacted. | `<cli> person get --id 760 --apikey BAD_KEY` |

## Pass criteria

| Area | Criteria |
| --- | --- |
| Help | Includes purpose, use cases, examples, auth notes, and next-action style guidance. |
| Auth | Uses `DIP_API_KEY` when no `--apikey` flag is passed. |
| Legacy compatibility | Existing v1 command paths still exist and return upstream-style JSON. |
| Research helpers | New commands return normalized envelopes with `status`, `request`, `retrievedAt`, `sources`, `warnings`, and `nextActions` where relevant. |
| Safety | API keys are not printed. Errors are JSON and nonzero exit codes. |
