# Regionalatlas Tool

This folder contains the refactored Regionalatlas skill and CLI implementations.

## Contents

- `SKILL.md`: agent-facing operating instructions.
- `references/openapi.yaml`: original OpenAPI wrapper spec.
- `references/notes.md`: operational notes.
- `references/research.md`: API research summary.
- `references/rate-limits-and-terms.md`: auth, rate-limit, and fair-use findings.
- `go/v1/main.go`: preserved legacy Go wrapper.
- `go/v2/main.go`: Go 2.0 research-oriented CLI.
- `bin/regionalatlasctl-legacy.exe`: preserved legacy executable.
- `bin/regionalatlasctl-2.0.exe`: built Go 2.0 executable.
- `python/regionalatlasctl.py`: Python implementation.
- `typescript/src/index.ts`: TypeScript / Node.js implementation.
- `tests/`: test plan and results.

## Fast Start

```powershell
skills\regionalatlas\bin\regionalatlasctl-2.0.exe doctor
skills\regionalatlas\bin\regionalatlasctl-2.0.exe indicators search --term "Arbeitslosenquote" --limit 3
skills\regionalatlas\bin\regionalatlasctl-2.0.exe sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 3
```

## Main Improvement

The old tool exposed only a raw dynamic-layer query wrapper. Version 2.0 preserves that raw access and adds catalog discovery, field explanations, source metadata, safe samples, query building, dossiers, hard output caps, and three runtime implementations.
