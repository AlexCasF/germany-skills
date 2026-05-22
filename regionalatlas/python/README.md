# Regionalatlas Python CLI

The Python implementation mirrors the Go 2.0 command surface and uses only the Python standard library.

## Run

```powershell
python skills\regionalatlas\python\regionalatlasctl.py doctor
python skills\regionalatlas\python\regionalatlasctl.py indicators search --term "Arbeitslosenquote" --limit 3
python skills\regionalatlas\python\regionalatlasctl.py sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 3
```

## Check

```powershell
python -m py_compile skills\regionalatlas\python\regionalatlasctl.py
```
