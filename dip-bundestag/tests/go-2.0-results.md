# Go 2.0 test results

Implementation: `skills/dip-bundestag/bin/dip-bundestag-2.0.exe`

Run date: 2026-05-18

Auth: `DIP_API_KEY` loaded from local environment. The key was not printed.

Build:

```text
cd skills/dip-bundestag/go
go build -o ..\bin\dip-bundestag-2.0.exe .
```

Result: passed.

## Results

| Test | Command | Result | Notes |
| --- | --- | --- | --- |
| 1 | `dip-bundestag-2.0.exe --help` | Pass | Root help contains purpose, use cases, fast paths, legacy endpoint commands, research commands, auth notes, and output notes. |
| 2 | `dip-bundestag-2.0.exe person dossier --help` | Pass | Command help explains dossier behavior, inputs, examples, and source orientation. |
| 3 | `dip-bundestag-2.0.exe doctor` | Pass | Returned `status=ok`, endpoint health, auth source, docs links, and fair-use warnings. |
| 4 | `dip-bundestag-2.0.exe person list --param "f.person=Gauweiler" --limit 1` | Pass | Legacy path still works and returned one upstream-style document. |
| 5 | `dip-bundestag-2.0.exe person search --name "Gauweiler" --limit 3` | Pass | Returned normalized envelope with one compact person item and next actions. |
| 6 | `dip-bundestag-2.0.exe person get --id 760` | Pass | Legacy exact lookup still returns upstream-style JSON. |
| 7 | `dip-bundestag-2.0.exe source --type plenarprotokoll --document-number "20/139"` | Pass | Returned normalized source envelope with API/PDF/XML source links. |
| 8 | `dip-bundestag-2.0.exe person dossier --id 760 --limit 3` | Pass | Returned person record, related activities, source data, warnings, and next actions. |
| 9 | `dip-bundestag-2.0.exe plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"` | Pass | Returned normalized envelope with 10 bounded snippets and source links. |
| 10 | `dip-bundestag-2.0.exe person get --id 760 --apikey BAD_KEY` | Pass | Returned nonzero exit with structured JSON error and did not echo the supplied key. |

## Notes

PowerShell turns native stderr from failing `.exe` commands into an error
record when using `2>&1`. The bad-key test was therefore verified through
`cmd /c "... 2>&1"` to inspect the raw JSON error payload.

The Go 2.0 CLI preserves the legacy endpoint paths and adds research-oriented
helpers without changing the default raw output for legacy commands.

