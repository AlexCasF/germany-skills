# DIP Go implementations

## v1

`v1/main.go` is the preserved legacy thin endpoint wrapper.

## v2

`v2/main.go` is the agent-friendly implementation. It keeps the legacy
endpoint commands and adds:

- `DIP_API_KEY` fallback
- `doctor`
- compact person search
- source extraction
- text/snippet helpers
- person and proceeding dossiers

