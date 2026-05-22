# Destatis notes

## High-level summary

Destatis GENESIS-Online provides official German statistical tables, time
series, metadata, variables, values, cubes, charts, maps, and result files.

The 2.0 CLI is designed for progressive discovery:

```text
doctor -> search -> table source -> table dossier -> variables/sample -> cited answer
```

## Current live behavior

- Use form `POST` for GENESIS REST calls.
- `GAST/GAST` worked for `helloworld/logincheck` and `find/find` in tests.
- `GAST/GAST` returned `401 Unauthorized` for several catalogue, metadata, and data endpoints in tests.
- Use personal `DESTATIS_USERNAME` and `DESTATIS_PASSWORD` for full access.

## Common workflows

- Use `doctor` before data work.
- Use `search --term "<concept>" --limit 5` for discovery.
- Use `table source --name <code>` for source/citation URLs.
- Use `table dossier --name <code>` to inspect metadata availability.
- Use `variables explain --table <code>` before interpreting dimensions.
- Use `table sample --name <code>` only after a precise table code is known.

## Common pitfalls

- Statistical values are unsafe without metadata.
- Broad table downloads can be large; keep limits small.
- Credentials must not appear in traces or logs.
- Guest discovery success does not guarantee metadata/data access.
- Table titles alone are not enough for final claims; preserve codes and dimensions.

## Output guidance

When summarizing results:

- include table/statistic/time-series code
- include source URL and retrieval date
- include unit, region, time period, and dimensions when available
- distinguish missing metadata from missing data
- state whether guest or configured credentials were used
