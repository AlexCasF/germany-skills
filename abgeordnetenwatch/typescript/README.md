# Abgeordnetenwatch CLI TypeScript/Node.js 2.0

Node.js implementation of the `abgeordnetenwatchctl` 2.0 command surface.

## Build

```powershell
cd skills\abgeordnetenwatch\typescript
npm install
npm run build
```

## Run

```powershell
node dist\index.js doctor
node dist\index.js politicians search --name "Alice Weidel" --limit 2
node dist\index.js politicians dossier --id 108379 --grep Nebentätigkeiten
```

## Notes

- Uses Node's built-in `https` module, not browser/global `fetch`.
- No API key is required.
- The official docs do not publish an exact request-per-minute rate limit.
