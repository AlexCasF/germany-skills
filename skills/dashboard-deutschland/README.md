# Dashboard Deutschland Tool

This folder contains the refactored Dashboard Deutschland skill and CLI implementations.

## Contents

- `SKILL.md`: agent-facing operating instructions.
- `references/openapi.yaml`: original OpenAPI wrapper spec.
- `references/notes.md`: operational notes.
- `references/research.md`: API research summary.
- `references/rate-limits-and-terms.md`: auth, rate-limit, and fair-use findings.
- `go/main.go`: Go research-oriented CLI.
- `bin/dashboard-deutschland.exe`: built Go executable.
- `python/dashboard-deutschland.py`: Python implementation.
- `typescript/src/index.ts`: TypeScript / Node.js implementation.

## Fast Start

```powershell
skills\dashboard-deutschland\bin\dashboard-deutschland.exe doctor
skills\dashboard-deutschland\bin\dashboard-deutschland.exe indicator search --term "Indikator" --limit 3
skills\dashboard-deutschland\bin\dashboard-deutschland.exe indicator data --id <indicator-id> --limit 3
```

## Main Improvement

The CLI exposes raw endpoint wrappers and adds dashboard discovery, indicator search, parsed tile metadata, chart-ready series extraction, source reporting, dossiers, endpoint diagnostics, and three runtime implementations.
