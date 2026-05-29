# TypeScript / Node Tagesschau CLI

Build:

```powershell
cd skills\tagesschau\typescript
npm install
npm run build
```

Run from the repository root:

```powershell
node skills\tagesschau\typescript\dist\index.js doctor
node skills\tagesschau\typescript\dist\index.js search --text Suchbegriff --limit 5
```

The TypeScript/Node version mirrors the Go command contract and uses native Node `fetch`.
