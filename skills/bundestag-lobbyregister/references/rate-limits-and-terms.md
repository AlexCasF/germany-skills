# Rate limits, auth, and terms

## Auth

The official Open Data/API page says API usage requires an API key. It also
publishes a currently valid key and says individual durable API keys can be
requested by email from the register office.

Do not copy the public key into repo files or logs. Use:

```powershell
$env:LOBBYREGISTER_API_KEY = "<key>"
```

The current OpenAPI document supports:

- `Authorization: ApiKey <key>`
- `apikey=<key>` query parameter

The CLI prefers the header form and redacts key material from output.

## Rate limits

No exact request-per-minute or daily quota was found in the official docs
reviewed.

Fair-use guidance for agents:

- keep broad search limits small
- prefer exact register numbers after discovery
- avoid repeated full-register downloads
- use `statistics` for aggregate facts instead of broad searches
- retry politely and preserve source timestamps

## Terms and data-use notes

The official Open Data/API page states that public register contents can be
queried through the API and references JSON/OpenAPI documentation. It also says
The current API replaced the earlier API on 2025-06-23.

Statement texts and PDFs may include copyrighted material. Quote only short
excerpts and cite the official source URL.

## Sources

- `https://www.lobbyregister.bundestag.de/informationen-und-hilfe/open-data-1049716`
- `https://api.lobbyregister.bundestag.de/rest/v2/R2.21-de.yaml`
