# TypeScript / Node Bundeshaushalt CLI

Build:

```powershell
cd skills\bundeshaushalt\typescript
npm install
npm run build
```

Run from the repository root:

```powershell
node skills\bundeshaushalt\typescript\dist\index.js doctor
node skills\bundeshaushalt\typescript\dist\index.js budget tree --year 2026 --account expenses --limit 5
```

The TypeScript/Node version mirrors the Go command contract and uses native Node `fetch`.
