# Python test results

Implementation: `python skills/dip-bundestag/python/dip-bundestag.py`

Run date: 2026-05-18

Auth: `DIP_API_KEY` loaded from local environment. The key was not printed.

Result: 10/10 passed.

## Results

| Test | Command | Result | Notes |
| --- | --- | --- | --- |
| 1 | `python dip-bundestag.py --help` | Pass | Root help contains purpose, use cases, fast paths, legacy endpoint commands, research commands, and auth notes. |
| 2 | `python dip-bundestag.py person dossier --help` | Pass | Command help explains dossier behavior, inputs, and examples. |
| 3 | `python dip-bundestag.py doctor` | Pass | Returned `status=ok`, endpoint health, auth source, docs links, and fair-use warnings. |
| 4 | `python dip-bundestag.py person list --param "f.person=Gauweiler" --limit 1` | Pass | Legacy path still works and returned one upstream-style document. |
| 5 | `python dip-bundestag.py person search --name "Gauweiler" --limit 3` | Pass | Returned normalized envelope with one compact person item and next actions. |
| 6 | `python dip-bundestag.py person get --id 760` | Pass | Legacy exact lookup still returns upstream-style JSON. |
| 7 | `python dip-bundestag.py source --type plenarprotokoll --document-number "20/139"` | Pass | Returned normalized source envelope with API/PDF/XML source links. |
| 8 | `python dip-bundestag.py person dossier --id 760 --limit 3` | Pass | Returned person record, related activities, source data, warnings, and next actions. |
| 9 | `python dip-bundestag.py plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"` | Pass | Returned normalized envelope with 10 bounded snippets and source links. |
| 10 | `python dip-bundestag.py person get --id 760 --apikey BAD_KEY` | Pass | Returned nonzero exit with structured JSON error and did not echo the supplied key. |

## Notes

The Python version uses only the standard library.

As with the Go executable, PowerShell may wrap native stderr from failing
commands. The bad-key payload was inspected through `cmd /c "... 2>&1"` and was
valid JSON.

