# TypeScript / Node.js test results

Implementation: `node skills/dip-bundestag/typescript/dist/index.js`

Run date: 2026-05-18

Auth: `DIP_API_KEY` loaded from local environment. The key was not printed.

Build:

```text
cd skills/dip-bundestag/typescript
npm install
npm run build
```

Result: 10/10 passed.

## Results

| Test | Command | Result | Notes |
| --- | --- | --- | --- |
| 1 | `node dist/index.js --help` | Pass | Root help contains purpose, use cases, fast paths, legacy endpoint commands, research commands, and auth notes. |
| 2 | `node dist/index.js person dossier --help` | Pass | Command help explains dossier behavior, inputs, and examples. |
| 3 | `node dist/index.js doctor` | Pass | Returned `status=ok`, endpoint health, auth source, docs links, and fair-use warnings. |
| 4 | `node dist/index.js person list --param "f.person=Gauweiler" --limit 1` | Pass | Legacy path still works and returned one upstream-style document. |
| 5 | `node dist/index.js person search --name "Gauweiler" --limit 3` | Pass | Returned normalized envelope with one compact person item and next actions. |
| 6 | `node dist/index.js person get --id 760` | Pass | Legacy exact lookup still returns upstream-style JSON. |
| 7 | `node dist/index.js source --type plenarprotokoll --document-number "20/139"` | Pass | Returned normalized source envelope with API/PDF/XML source links. |
| 8 | `node dist/index.js person dossier --id 760 --limit 3` | Pass | Returned person record, related activities, source data, warnings, and next actions. |
| 9 | `node dist/index.js plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"` | Pass | Returned normalized envelope with 10 bounded snippets and source links. |
| 10 | `node dist/index.js person get --id 760 --apikey BAD_KEY` | Pass | Returned nonzero exit with structured JSON error and did not echo the supplied key. |

## Notes

The first TypeScript implementation used Node 24 global `fetch`, which produced
correct JSON but triggered a Windows/libuv assertion after some requests. The
implementation now uses Node's built-in `https` module instead. That keeps the
same behavior and avoids the runtime assertion.

PowerShell may wrap native stderr from failing commands. The bad-key payload
was inspected through `cmd /c "... 2>&1"` and was valid JSON.

