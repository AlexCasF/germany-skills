# Deutschlandatlas test plan

Reference table: `alq_HA2023`

This table was chosen because portal search finds it reliably and live service
metadata exposes a feature layer at id `5`, which validates layer discovery.

## Test cases

1. Root help: `--help`
2. Command help: `table sample --help`
3. Endpoint/auth health: `doctor`
4. Legacy raw query: `table query --table alq_HA2023 --layer 5 --param where=1=1 --param outFields=* --limit 2`
5. Table discovery: `tables search --term Apotheken --limit 3`
6. Field discovery: `table fields --table alq_HA2023`
7. Source URLs: `table source --table alq_HA2023`
8. Evidence bundle: `indicator dossier --table alq_HA2023 --limit 2`
9. Query builder: `query-builder --table alq_HA2023 --region Berlin --fields name,alq --limit 3`
10. Safety guard: `table sample --table alq_HA2023 --limit 5000`

## Expected behavior

- Tests 1-9 exit with code 0.
- Test 10 exits with code 2 and a structured `limit_exceeds_safe_max` error.
- JSON research commands include `sources[]`, `warnings[]`, and `nextActions[]`.
- Broad samples do not request geometry by default.
