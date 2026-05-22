# regionalatlasctl TypeScript / Node.js 2.0

TypeScript / Node.js implementation of the Regionalatlas research CLI.

## Build

```powershell
npm install
npm run build
```

## Run

```powershell
node dist/index.js doctor
node dist/index.js indicators search --term Arbeitslosenquote --limit 3
node dist/index.js sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 3
```

The command surface mirrors the Go and Python versions. JSON is the default output format.

For raw dynamic-layer JSON on Windows, prefer `query --layer-file layer.json` over `query --layer <json>`.
