# Bundeshaushalt Notes

## What The API Provides

Bundeshaushalt Digital exposes structured German federal budget data as JSON. The useful live endpoint is:

```text
https://bundeshaushalt.de/internalapi/budgetData
```

The endpoint returns a hierarchy for a selected:

- `year`
- `account`: `expenses` or `income`
- `quota`: `target` or `actual`
- `unit`: `single`, `function`, or `group`
- optional `id` for a specific node

The response usually contains:

- `meta`: selected year, account, quota, unit, update timestamp, hierarchy entity, and current/max level.
- `detail`: the current root/node.
- `children`: child nodes.
- `parents`: hierarchy context.
- `related`: related agency/function/group links for title-level responses.

## Important Field Meanings

- `value`: nominal euro amount.
- `relativeToParentValue`: share of the immediate parent.
- `relativeValue`: share of the current root context.
- `budgetNumber`: official budget number where present.
- `label`: human-readable budget item label.
- `modifyDate`: upstream modification date in the `meta` object.

## Known Live Behavior

- Target/Soll values were reachable through 2026 during testing.
- Actual/Ist values were reachable through 2024 during testing.
- The old OpenAPI enum stops at 2021 and should not be treated as authoritative for live year availability.
- Missing required query params return HTTP 400.
- Transient HTTP 503 responses can happen under quick repeated node-level requests; the 2.0 CLIs retry briefly.

## Agent-Friendly Workflow

```text
doctor -> fields/years list -> budget tree -> search/title get -> compare -> cited answer
```

Use `search` when you do not know an internal id. Use `title get` once an id is known. Use `compare` to avoid manual repeated calls across years.

## Research Caveats

- This API explains federal budget structure and nominal budget amounts.
- It does not explain economic causes, inflation, unemployment, population, or regional indicators.
- For claims about social conditions or macroeconomic trends, combine budget values with Destatis or other statistical sources.
- Do not mix `single`, `function`, and `group` classifications without explaining the difference.
