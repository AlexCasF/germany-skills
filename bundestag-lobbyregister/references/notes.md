# Bundestag Lobbyregister notes

## High-level summary

The Bundestag Lobbyregister API exposes official public register data for
interests represented toward the German Bundestag and the Federal Government.

The current useful surface is API V2 under:

```text
https://api.lobbyregister.bundestag.de/rest/v2
```

The old `sucheDetailJson` wrapper is preserved for comparison, but the
official Open Data/API page says API V2 replaced API V1 on 2025-06-23.

## Common workflows

- Check endpoint/auth health with `doctor`.
- Use `statistics` for aggregate register counts.
- Use `search --term "<name>" --limit 3` for discovery.
- Use `entry get --register-number <R...>` for exact detail.
- Use `entry source --register-number <R...>` for citation URLs.
- Use `entry dossier --register-number <R...>` for source-rich research.
- Use `financial summary --register-number <R...>` for finance-focused questions.
- Use `statements list --register-number <R...> --grep "<term>"` for embedded statement text.

## Common pitfalls

- V2 requires an API key; use `LOBBYREGISTER_API_KEY`.
- Search responses can be large because they include rich entry details.
- Register disclosure data is not proof of illegality or corruption.
- Financial expense fields are ranges, not exact amounts.
- Some statement texts and PDFs may be copyrighted; quote sparingly.
- For parliamentary proceedings or speeches, switch to DIP/Bundestag tools.

## Output guidance

When summarizing results:

- include `registerNumber`
- include `name`
- include `sourceDate`
- cite the public detail page and PDF URL when available
- preserve financial ranges and fiscal years
- distinguish donations, membership fees, public allowances, and expenses
- include caveats for self-reported disclosure data
