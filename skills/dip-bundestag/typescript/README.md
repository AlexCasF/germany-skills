# DIP TypeScript / Node.js CLI

This is the TypeScript source for the Node.js DIP CLI.

Install local dev dependencies and build:

```powershell
cd skills\dip-bundestag\typescript
npm install
npm run build
```

Run from the repo root after building:

```powershell
node skills\dip-bundestag\typescript\dist\index.js --help
```

It uses `DIP_API_KEY` unless `--apikey` is passed.

