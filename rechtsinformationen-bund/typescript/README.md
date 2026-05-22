# Rechtsinformationen CLI TypeScript/Node.js

Node.js implementation of the `rechtsinformationen-bund` command surface.

## Build

```powershell
cd skills\rechtsinformationen-bund\typescript
npm install
npm run build
```

## Run

```powershell
node dist\index.js doctor
node dist\index.js documents search --search-term "Suchbegriff" --limit 2
node dist\index.js documents dossier --type case-law --document-number KORE600422026 --grep Revision
```

## Notes

- Uses Node's built-in `https` module, not browser/global `fetch`.
- No API key is required.
- The upstream trial API rate limit is 600 requests per minute per client IP.
