# Dashboard Deutschland Tool

This folder contains the refactored Dashboard Deutschland skill and CLI implementations.

## Contents

- `SKILL.md`: agent-facing operating instructions.
- `references/openapi.yaml`: original OpenAPI wrapper spec.
- `references/notes.md`: operational notes.
- `references/research.md`: API research summary.
- `references/rate-limits-and-terms.md`: auth, rate-limit, and fair-use findings.
- `go/v1/main.go`: preserved legacy Go wrapper.
- `go/v2/main.go`: Go 2.0 research-oriented CLI.
- `bin/dashboardctl-legacy.exe`: preserved legacy executable.
- `bin/dashboardctl-2.0.exe`: built Go 2.0 executable.
- `python/dashboardctl.py`: Python implementation.
- `typescript/src/index.ts`: TypeScript / Node.js implementation.
- `tests/`: test plan and results.

## Fast Start

```powershell
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe doctor
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe indicator search --term "Arbeitslosigkeit" --limit 3
skills\dashboard-deutschland\bin\dashboardctl-2.0.exe indicator data --id tile_1666958835081 --limit 3
```

## Main Improvement

The old tool exposed raw endpoint wrappers. Version 2.0 preserves those wrappers and adds dashboard discovery, indicator search, parsed tile metadata, chart-ready series extraction, source reporting, dossiers, endpoint diagnostics, and three runtime implementations.
