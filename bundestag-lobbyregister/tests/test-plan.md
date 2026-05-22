# Bundestag Lobbyregister test plan

This plan checks the agent-facing command contract for all three
implementations: Go 2.0, Python, and TypeScript/Node.js.

Load `LOBBYREGISTER_API_KEY` before live V2 tests. Do not print the key in test
logs.

## Test cases

| ID | Purpose | Command shape | Expected result |
| --- | --- | --- | --- |
| 1 | Root discovery | `--help` | Shows purpose, fast paths, auth, research commands, and legacy note |
| 2 | Focused help | `entry dossier --help` | Shows dossier inputs and examples |
| 3 | Health and policy summary | `doctor` | JSON envelope with auth status, docs, fair-use, and live health if configured |
| 4 | Statistics | `statistics` | Compact register-wide metrics with source date |
| 5 | Safe search | `search --term "Bundesverband Soziokultur" --limit 2` | Compact result with register number, source URLs, finance/project counts, and next actions |
| 6 | Detail/get | `entry get --register-number R001255` | Normalized exact entry summary |
| 7 | Source expansion | `entry source --register-number R001255` | API/public/PDF/statement source URLs |
| 8 | Dossier | `entry dossier --register-number R001255 --grep Soziokultur --limit 3` | Evidence bundle with summary, finance, projects, statements, warnings, and next actions |
| 9 | Financial summary | `financial summary --register-number R001255` | Normalized finance ranges, funding, donations, fees, allowances, and annual-report link |
| 10 | Error safety | `entry get --register-number BAD` | Non-zero exit with structured JSON error |

## Fixed test fixture

- Register number `R001255` is `Bundesverband Soziokultur` during testing.
- The entry has public detail/PDF URLs, financial ranges, regulatory projects,
  and at least one statement, making it a useful integration fixture.

## Extra smoke check

`statements list --register-number R001255 --grep Soziokultur --limit 2`
returned one matching statement and did not leak key material.
