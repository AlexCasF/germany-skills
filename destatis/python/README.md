# Python implementation

Run with:

```powershell
python skills\destatis\python\destatisctl.py doctor
```

The Python version mirrors the Go 2.0 command contract: `doctor`, `search`,
legacy endpoint commands, `table source`, `table dossier`, `table sample`,
`timeseries dossier`, and `variables explain`.

Set `DESTATIS_USERNAME` and `DESTATIS_PASSWORD` for full metadata/data access.
