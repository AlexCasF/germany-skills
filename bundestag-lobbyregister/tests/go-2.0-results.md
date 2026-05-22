# Go 2.0 test results

Implementation:

```powershell
skills\bundestag-lobbyregister\bin\lobbyregisterctl-2.0.exe
```

Environment:

- Go 1.26.2
- `LOBBYREGISTER_API_KEY` loaded from local environment during tests
- Test date: 2026-05-18

## Results

| ID | Result | Notes |
| --- | --- | --- |
| 1 | Pass | Root help printed research guidance. |
| 2 | Pass | `entry dossier --help` printed inputs and examples. |
| 3 | Pass | `doctor` returned `status=ok`, live health, and no key leakage. |
| 4 | Pass | `statistics` returned `totalLobbyists=6846`, `activeLobbyists=6205`, `peopleInvolved=29485`. |
| 5 | Pass | Safe search returned one result for `Bundesverband Soziokultur` with `R001255`. |
| 6 | Pass | `entry get` returned normalized summary for `R001255`. |
| 7 | Pass | `entry source` returned public detail/PDF/source URLs. |
| 8 | Pass | `entry dossier` returned summary, finance, projects, statements, warnings, and next actions. |
| 9 | Pass | `financial summary` returned finance fields and caveats. |
| 10 | Pass | Invalid register number returned structured `invalid_register_number` error with exit 2. |

Key redaction check: pass for all cases.
