# Dashboard Deutschland Python CLI

The Python implementation mirrors the Go 2.0 command surface and uses only the Python standard library.

## Run

```powershell
python skills\dashboard-deutschland\python\dashboardctl.py doctor
python skills\dashboard-deutschland\python\dashboardctl.py indicator search --term "Arbeitslosigkeit" --limit 3
python skills\dashboard-deutschland\python\dashboardctl.py indicator data --id tile_1666958835081 --limit 3
```

## Check

```powershell
python -m py_compile skills\dashboard-deutschland\python\dashboardctl.py
```
