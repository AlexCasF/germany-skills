# Abgeordnetenwatch CLI Python 2.0

Stdlib Python implementation of the `abgeordnetenwatchctl` 2.0 command surface.

## Run

```powershell
python skills\abgeordnetenwatch\python\abgeordnetenwatchctl.py doctor
python skills\abgeordnetenwatch\python\abgeordnetenwatchctl.py politicians search --name "Alice Weidel" --limit 2
python skills\abgeordnetenwatch\python\abgeordnetenwatchctl.py politicians dossier --id 108379 --grep Nebentätigkeiten
```

## Notes

- No API key is required.
- The official docs do not publish an exact request-per-minute rate limit.
- JSON is emitted by default for machine-friendly agent use.
