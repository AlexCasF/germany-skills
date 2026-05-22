# Bundesrat Live Test Plan

The same ten checks are run against the Go, Python, and TypeScript / Node.js implementations.

## Test Cases

| # | Test | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Root help | `--help` | exit `0`, text help |
| 2 | Doctor | `doctor` | exit `0`, JSON envelope with endpoint health, no auth requirement, and fair-use notes |
| 3 | News search | `news search --term Bovenschulte --limit 2` | exit `0`, compact news results with source URLs and next actions |
| 4 | News page | `news page --url <bundesrat.de news URL> --grep Merkel` | exit `0`, public-page snippets and source metadata |
| 5 | Dates | `dates --limit 2` | exit `0`, compact event/date records |
| 6 | Members search | `members search --name Özdemir --limit 1` | exit `0`, compact member record with official profile URL |
| 7 | Member dossier | `members dossier --name Özdemir --grep Bundesrat` | exit `0`, role/biography/contact snippets and profile source |
| 8 | Plenum compact | `plenum compact --limit 1 --top-limit 2` | exit `0`, BundesratKOMPAKT summary/TOP data |
| 9 | Plenum current | `plenum current --limit 1 --top-limit 2` | exit `0`, current agenda/TOP data with Drucksachen/DIP links where present |
| 10 | Plenum next | `plenum next` | exit `0`, upcoming sitting date table and source metadata |

## Extra Smoke Checks

These are not part of the shared ten-case plan but were run during implementation:

- `votes summary`
- `presidium --limit 2`

## Notes

- Broad commands use explicit small limits.
- `news page` and `page` only accept `https://www.bundesrat.de` URLs.
- Public HTML extraction is best-effort; XML feed fields remain preferred for structured metadata.
