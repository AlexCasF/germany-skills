# Deutschlandatlas TypeScript / Node.js CLI

TypeScript/Node.js parity implementation for `deutschlandatlas`.

## Build

```powershell
cd skills\deutschlandatlas\typescript
npm install
npm run build
```

## Run

```powershell
node skills\deutschlandatlas\typescript\dist\index.js doctor
node skills\deutschlandatlas\typescript\dist\index.js table sample --table alq_HA2023 --limit 2
```

The implementation uses Node's built-in `fetch`.
