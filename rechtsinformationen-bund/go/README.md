# Rechtsinformationen Go implementations

## v1

`v1/main.go` is the preserved legacy thin endpoint wrapper.

## v2

`v2/main.go` is the agent-friendly implementation. It keeps the legacy
endpoint commands and adds:

- `doctor`
- compact search via `--search-term` and `--limit`
- source extraction
- HTML/XML text helpers
- document, case-law, and legislation dossiers
- citation helper output

