# Bundeshaushalt CLI Test Plan

Run the same cases against Go 2.0, Python, and TypeScript/Node.

| # | Case | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Help | `--help` | exit `0`, non-empty text help |
| 2 | Doctor | `doctor` | exit `0`, JSON envelope, endpoint checks pass |
| 3 | Fields | `fields` | exit `0`, JSON envelope with account/quota/unit meanings |
| 4 | Years | `years list` | exit `0`, known years include 2012-2026 |
| 5 | Root tree | `budget tree --year 2026 --account expenses --quota target --unit single --limit 3` | exit `0`, compact child rows |
| 6 | Node tree | `budget tree --year 2025 --account expenses --id 11 --limit 3` | exit `0`, ministry node children |
| 7 | Search | `search --year 2026 --account expenses --term Bundesministerium --limit 2` | exit `0`, bounded traversal |
| 8 | Title lookup | `title get --year 2025 --account expenses --id 110168112` | exit `0`, exact title-level detail |
| 9 | Compare | `compare --years 2024,2025 --account expenses --id 110168112` | exit `0`, two year rows |
| 10 | Raw endpoint compatibility | `budget-data --param year=2025 --param account=expenses --param quota=target --param unit=single --raw` | exit `0`, upstream JSON |

## Notes

- The search test intentionally uses a first-level term to avoid unnecessary traversal load.
- The title and compare tests exercise the `related` response shape found in live title-level responses.
- The raw endpoint test verifies that the legacy endpoint-wrapper use case still exists.
