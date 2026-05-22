# Dashboard Deutschland Test Plan

The same ten checks are run against the Go, Python, and TypeScript / Node.js implementations.

## Test Cases

| # | Test | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Root help | `--help` | exit `0`, text help |
| 2 | Indicator data help | `indicator data --help` | exit `0`, text help |
| 3 | Doctor | `doctor` | exit `0`, JSON envelope with `status: degraded` because GeoJSON returns 403 |
| 4 | Legacy dashboard get | `dashboard get --param ids=arbeitsmarkt` | exit `0`, upstream JSON |
| 5 | Legacy indicators | `indicators --param ids=tile_1666958835081` | exit `0`, upstream JSON |
| 6 | Dashboards list | `dashboards list --limit 5` | exit `0`, JSON envelope |
| 7 | Indicator get | `indicator get --id tile_1666958835081` | exit `0`, parsed tile metadata |
| 8 | Indicator data series | `indicator data --id tile_1666958835081 --series Arbeitslose --limit 3` | exit `0`, bounded chart points |
| 9 | Dashboard dossier | `dashboard dossier --id arbeitsmarkt --indicator-limit 2` | exit `0`, dashboard + indicator summaries |
| 10 | Geo failure diagnostic | `geo` | exit `1`, structured `geo_endpoint_failed` error |

## Notes

- The GeoJSON failure is expected as of the test date.
- Legacy commands are intentionally raw compatibility wrappers.
- Research commands return the standard envelope with sources, warnings, and next actions.
