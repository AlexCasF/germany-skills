# Rechtsinformationen CLI Python 2.0

Stdlib Python implementation of the `rechtsinformationenctl` 2.0 command surface.

## Run

```powershell
python skills\rechtsinformationen-bund\python\rechtsinformationenctl.py doctor
python skills\rechtsinformationen-bund\python\rechtsinformationenctl.py documents search --search-term "Bürgergeld" --limit 2
python skills\rechtsinformationen-bund\python\rechtsinformationenctl.py documents dossier --type case-law --document-number KORE600422026 --grep Revision
```

## Notes

- No API key is required.
- The upstream trial API rate limit is 600 requests per minute per client IP.
- JSON is emitted by default for machine-friendly agent use.
