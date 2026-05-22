# TypeScript/Node.js test results

Implementation: `skills/abgeordnetenwatch/typescript/src/index.ts`

Build commands:

```powershell
cd skills\abgeordnetenwatch\typescript
npm install
npm run build
```

Result: pass.

## Test summary

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help shows purpose, fast paths, legacy commands, and research commands. |
| 2 | Pass | `politicians dossier --help` shows evidence-bundle behavior and examples. |
| 3 | Pass | `doctor` reports no auth requirement, API version 2.8.2, CC0 license, docs links, and no published exact rate limit. |
| 4 | Pass | `parliaments list --limit 1` returns raw upstream JSON with one returned item and total 18. |
| 5 | Pass | Search returns a compact source-rich politician item. |
| 6 | Pass | Exact politician lookup by ID returns raw upstream JSON. |
| 7 | Pass | Profile-page extraction returns metadata, shortlink, text length, and snippets. |
| 8 | Pass | Dossier returns mandate count 1, side-job count 1, income sum 16690.87, and page snippets. |
| 9 | Pass | Side-job join returns disclosed side-job evidence through mandate ID. |
| 10 | Pass | Invalid ID exits non-zero with structured JSON error for upstream HTTP 500. |

## Implementation note

This version uses Node's built-in `https` module instead of global `fetch`, matching the safer pattern from the previous tool refactor.
