# Deutschlandatlas manifest

## Folder contents

- `SKILL.md`: final operational guide for skill-using agents.
- `README.md`: human-facing quick start.
- `references/openapi.yaml`: original OpenAPI wrapper reference.
- `references/notes.md`: local usage notes.
- `references/research.md`: API purpose and data coverage research.
- `references/rate-limits-and-terms.md`: auth, rate-limit, and fair-use notes.
- `go/v1/main.go`: preserved legacy Go source.
- `go/v2/main.go`: refactored Go 2.0 source.
- `bin/deutschlandatlasctl-legacy.exe`: preserved legacy executable.
- `bin/deutschlandatlasctl-2.0.exe`: built Go 2.0 executable.
- `python/deutschlandatlasctl.py`: Python parity CLI.
- `typescript/src/index.ts`: TypeScript/Node.js parity CLI source.
- `typescript/dist/index.js`: compiled TypeScript/Node.js CLI.
- `tests/*.md`: test plan and result notes.

## Inspect first

Start with:

1. `SKILL.md`
2. `tests/test-plan.md`
3. `tests/go-2.0-results.md`
4. `go/v2/main.go`

## Known caveat

No exact official API rate limit was found in reviewed public documentation.
The tool therefore enforces cautious defaults and records fair-use warnings.
