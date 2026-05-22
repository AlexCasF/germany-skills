# Bundestag Live Test Plan

The same ten checks are run against the Go, Python, and TypeScript / Node.js implementations.

## Test Cases

| # | Test | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Root help | `--help` | exit `0`, text help |
| 2 | Doctor | `doctor` | exit `0`, JSON envelope with endpoint health and no auth requirement |
| 3 | Plenary agenda | `plenum conferences --limit 1 --item-limit 2` | exit `0`, bounded agenda JSON with article next actions |
| 4 | Member search | `members search --name Amthor --limit 1` | exit `0`, compact member result with profile and XML source URLs |
| 5 | Member dossier | `members dossier --id 2022 --grep Tätigkeiten` | exit `0`, biography and disclosure snippets |
| 6 | Committee search | `committees search --term Arbeit --limit 1` | exit `0`, compact committee result with detail next action |
| 7 | Committee dossier | `committees dossier --id a11 --member-limit 2 --news-limit 1 --grep Arbeit` | exit `0`, bounded members, news, task text, and source URLs |
| 8 | Article XML | `article get --article-id 1174778 --grep Meinungsfreiheit` | exit `0`, structured article metadata, snippets, and public page URL |
| 9 | Article page | `article page --url https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778 --grep Meinungsfreiheit` | exit `0`, best-effort public page snippets |
| 10 | Video feed | `video feed --content-id 7529016` | exit `0`, stream groups and media usage warning |

## Notes

- Commands that touch broad XML indexes use small explicit limits.
- `article page` is best-effort HTML extraction; `article get` remains the preferred structured source.
- The current speaker feed was smoke-tested during development, but the ten-case shared plan prioritizes richer source-expansion flows.
