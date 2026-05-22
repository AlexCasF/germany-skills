# Go 2.0 Test Results

Runtime:

```text
skills\bundeshaushalt\bin\bundeshaushalt-2.0.exe
```

| # | Case | Exit | Result | Note |
| --- | --- | --- | --- | --- |
| 1 | Help | 0 | Pass | Printed text help. |
| 2 | Doctor | 0 | Pass | Checked 2026 target expenses and 2024 actual expenses. |
| 3 | Fields | 0 | Pass | Returned account, quota, unit, and known-year metadata. |
| 4 | Years | 0 | Pass | Returned 15 known years, 2012-2026. |
| 5 | Root tree | 0 | Pass | Returned 2026 spending root with 26 children, 3 shown. |
| 6 | Node tree | 0 | Pass | Returned 2025 Einzelplan 11 with 15 children, 3 shown. |
| 7 | Search | 0 | Pass | Returned 2 bounded first-level matches using 1 request. |
| 8 | Title lookup | 0 | Pass | Returned `1101 681 12 Bürgergeld`, 29.6 billion EUR for 2025 target. |
| 9 | Compare | 0 | Pass | Returned 2024 and 2025 rows for `110168112`. |
| 10 | Raw endpoint compatibility | 0 | Pass | Returned upstream JSON from `budget-data --param ... --raw`. |

## Additional Fix Verified

Title-level responses include `related` as an object keyed by category. The Go parser was adjusted and retested successfully.
