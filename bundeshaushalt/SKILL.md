---
name: bundeshaushalt
description: Use this skill for German federal budget research through Bundeshaushalt Digital, including revenue/spending hierarchy exploration, budget-line lookup, and year-to-year comparisons.
---

# Bundeshaushalt Skill

## Purpose

Use this skill when a user asks about the German federal budget, ministry-level spending, budget titles, revenue or expenditure categories, planned versus actual budget values, or changes in a federal budget line across years.

The tool reads the Bundeshaushalt Digital internal JSON endpoint. It is best for fiscal structure and nominal euro amounts. It is not a macroeconomic statistics source; use Destatis, Regionalatlas, Deutschlandatlas, or Dashboard Deutschland for inflation, population, labor-market, or regional context.

## First Moves

Start narrow and let the CLI teach you the next command.

```powershell
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe doctor
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe fields
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe years list
```

If you need a Python or TypeScript/Node equivalent:

```powershell
python skills\bundeshaushalt\python\bundeshaushalt.py doctor
node skills\bundeshaushalt\typescript\dist\index.js doctor
```

## Research Workflow

Use this sequence for most questions:

1. Run `doctor` to check live endpoint behavior and fair-use hints.
2. Run `years list` when the year is not obvious.
3. Run `fields` when you need to decide between `account`, `quota`, or `unit`.
4. Use `budget tree` to inspect the hierarchy.
5. Use `search` when you do not know the internal budget id.
6. Use `title get` for the exact node or title.
7. Use `compare` for year-to-year change.
8. Preserve the `sources[]`, `request.url`, `meta.modifyDate`, `year`, `quota`, and `unit` in citations and artifacts.

## Main Commands

- `doctor`: validates live endpoint access, auth status, current target/actual examples, and rate-limit findings.
- `source`: emits canonical application, API, BMF, usage-note, and fair-use URLs.
- `fields`: explains `expenses` versus `income`, `target` versus `actual`, and the `single`/`function`/`group` views.
- `years list`: returns known live years. Current tests found target values through 2026 and actual values through 2024.
- `budget tree --year YEAR --account expenses|income`: returns a compact hierarchy node and children.
- `search --year YEAR --account expenses|income --term TERM`: traverses the hierarchy with request caps.
- `title get --year YEAR --account expenses|income --id ID`: fetches one exact budget node.
- `compare --years 2024,2025 --account expenses --id ID`: compares one node across years.
- `budget-data`: compatibility wrapper for the original endpoint-style command; supports `--param key=value` and `--raw`.

## Common Examples

```powershell
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe budget tree --year 2026 --account expenses --quota target --unit single --limit 8
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe budget tree --year 2025 --account expenses --id 11 --limit 8
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe search --year 2025 --account expenses --term "Bürgergeld" --limit 5
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe title get --year 2025 --account expenses --id 110168112
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe compare --years 2024,2025 --account expenses --id 110168112
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe budget-data --param year=2025 --param account=expenses --param quota=target --param unit=single --raw
```

## Interpretation Rules

- Always say whether values are `target`/Soll or `actual`/Ist.
- Always say whether `account` is `expenses` or `income`.
- Always preserve `unit`: `single` means Einzelplan/ministry/title hierarchy, `function` means policy-function classification, and `group` means economic grouping.
- Treat `value` and `valueEur` as nominal euro amounts.
- Do not compare nominal budget values to inflation-adjusted, per-capita, or macroeconomic claims without bringing in statistical context from another source.
- Use `meta.modifyDate` when recency matters.
- Remember that newer years can have target values but no actual values yet.

## API Caveats

- The old OpenAPI file is stale: its year enum stops at 2021, while live endpoint tests found target data through 2026.
- The live endpoint requires at least `year` and `account`; missing required params return HTTP 400.
- Some node responses include `related` categories such as `agency`, `function`, and `group`; the 2.0 CLIs parse these.
- Broad `search` can traverse many hierarchy nodes. Keep `--limit` low and adjust `--max-requests` only when needed.
- No exact public API quota was found. Respect `robots.txt` crawl-delay guidance for crawling-like workflows and avoid repeated large traversals.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
- `tests/test-plan.md`
