# Tagesschau CLI Test Plan

Run the same planned cases against Go 2.0, Python, and TypeScript/Node.

| # | Case | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Root help | `--help` | exit `0`, non-empty text help |
| 2 | Article help | `article get --help` | exit `0`, article URL/snippet flags visible |
| 3 | Source metadata | `source` | exit `0`, JSON envelope with API and reuse references |
| 4 | Fields | `fields` | exit `0`, JSON envelope with feed, ressort, region, and article-field notes |
| 5 | Doctor | `doctor` | exit `0`, endpoint health and usage warnings |
| 6 | Homepage | `homepage --limit 1` | exit `0`, compact feed item |
| 7 | News filter | `news --ressort inland --limit 1` | exit `0`, compact filtered item |
| 8 | Channels | `channels --limit 1` | exit `0`, compact channel item |
| 9 | Legacy search params | `search --param searchText=Bundestag --param pageSize=1` | exit `0`, compact search result |
| 10 | Article grep | `article get --url <detailsweb> --grep Bundestag --limit 1` | exit `0`, bounded snippet |

## Extra Smoke Check

`article source --url <detailsweb>` was also run for each runtime to verify URL conversion and citation metadata without fetching article text.

## Network Discipline

Because published documentation says not to exceed 60 requests per hour, tests should use small limits and avoid repeated broad article expansion.
