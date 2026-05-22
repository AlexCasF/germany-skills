# Regionalatlas Tool

This folder contains the refactored Regionalatlas skill and CLI implementations.

## Contents

- `SKILL.md`: agent-facing operating instructions.
- `references/openapi.yaml`: original OpenAPI wrapper spec.
- `references/notes.md`: operational notes.
- `references/research.md`: API research summary.
- `references/rate-limits-and-terms.md`: auth, rate-limit, and fair-use findings.
- `go/main.go`: Go research-oriented CLI.
- `bin/regionalatlas.exe`: built Go executable.
- `python/regionalatlas.py`: Python implementation.
- `typescript/src/index.ts`: TypeScript / Node.js implementation.

## Fast Start

```powershell
skills\regionalatlas\bin\regionalatlas.exe doctor
skills\regionalatlas\bin\regionalatlas.exe indicators search --term "Indikator" --limit 3
skills\regionalatlas\bin\regionalatlas.exe sample --indicator <indicator-code> --field <field-code> --year 2024 --region-level 1 --limit 3
```

## Main Improvement

The old tool exposed only a raw dynamic-layer query wrapper. The CLI preserves that raw access and adds catalog discovery, field explanations, source metadata, safe samples, query building, dossiers, hard output caps, and three runtime implementations.
