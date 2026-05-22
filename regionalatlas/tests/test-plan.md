# Regionalatlas Test Plan

The same ten checks are run against the Go, Python, and TypeScript / Node.js implementations.

## Test Cases

| # | Test | Command shape | Expected |
| --- | --- | --- | --- |
| 1 | Root help | `--help` | exit `0`, text help |
| 2 | Sample help | `sample --help` | exit `0`, text help |
| 3 | Doctor | `doctor` | exit `0`, JSON envelope |
| 4 | Raw query compatibility | `query --layer-file tmp\regionalatlas-layer.json --param outFields=... --param resultRecordCount=2` | exit `0`, upstream ArcGIS JSON |
| 5 | Indicator search | `indicators search --term Arbeitslosenquote --limit 3` | exit `0`, JSON envelope |
| 6 | Indicator get | `indicator get --indicator AI008-1-5` | exit `0`, JSON envelope |
| 7 | Source metadata | `source --indicator AI008-1-5 --field AI0801 --year 2024` | exit `0`, JSON envelope |
| 8 | Dossier | `dossier --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 2` | exit `0`, JSON envelope |
| 9 | Explain field grep | `explain-field --indicator AI008-1-5 --field AI0801 --grep Quelle` | exit `0`, JSON envelope |
| 10 | Large output guard | `sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 5 --limit 5000` | exit `2`, structured error |

## Notes

- Test 4 uses `--layer-file` because Windows native-command JSON quoting corrupts raw layer JSON.
- The CLIs strip UTF-8 BOMs from `--layer-file` content.
- `exceededTransferLimit=true` is expected for small ArcGIS samples and is treated as a warning, not a failure.
