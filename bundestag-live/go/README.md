# Go Implementation

## Versions

- `v1/main.go`: original endpoint-wrapper implementation
- `v2/main.go`: agent-friendly 2.0 implementation

## Build

From `skills/bundestag-live/go/v2`:

```text
go build -o ..\..\bin\bundestagctl-2.0.exe .
```

## Notes

The Go 2.0 implementation is the primary executable. It keeps raw XML access through `--raw` and adds normalized JSON research commands.
