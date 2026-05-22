---
name: api-bundestag-lobbyregister
description: Use this skill for official Bundestag Lobbyregister research, including registered interest representatives, lobby finance ranges, funding, donations, regulatory projects, statements, public register pages, and register-wide statistics.
---

# Bundestag Lobbyregister skill

## Purpose

Use this skill to search and cite official public data from the Bundestag
Lobbyregister for interests represented toward the German Bundestag and the
Federal Government.

This skill is especially useful for organizations/persons in the register,
financial expense ranges, funding sources, donations, membership fees, public
allowances, regulatory projects, statements, statement PDFs, and public detail
pages.

## Service facts

- Base URL: `https://api.lobbyregister.bundestag.de/rest/v2`
- Public register: `https://www.lobbyregister.bundestag.de`
- Open Data/API page: `https://www.lobbyregister.bundestag.de/informationen-und-hilfe/open-data-1049716`
- OpenAPI: `https://api.lobbyregister.bundestag.de/rest/v2/R2.21-de.yaml`
- Auth: API key required
- Preferred auth: `LOBBYREGISTER_API_KEY`
- Published exact rate limit: not found in official docs reviewed
- API version note: V2 replaced V1 on 2025-06-23

## Use this when

- the user asks about a registered lobby organization/person
- the user asks who is represented in the official Lobbyregister
- the user asks about lobbying expenditure ranges or funding sources
- the user asks about donations, membership fees, public allowances, contracts, or annual-report links in the register
- the user asks for regulatory projects or statements submitted by a registered interest representative
- the user needs official source URLs for public register pages or PDFs
- the user needs register-wide counts/statistics

## Do not use this when

- the user needs official parliamentary proceedings or plenary speeches; prefer DIP/Bundestag tools
- the user needs law text or court material; prefer Rechtsinformationen des Bundes
- the user needs general news context; prefer a news/source tool and then cross-check official records
- the user treats register disclosure data alone as proof of corruption or illegality

## Preferred tool

Prefer the 2.0 CLI contract.

Use the local executable when available:

```powershell
skills\bundestag-lobbyregister\bin\lobbyregisterctl-2.0.exe doctor
```

Portable fallbacks:

```powershell
python skills\bundestag-lobbyregister\python\lobbyregisterctl.py doctor
node skills\bundestag-lobbyregister\typescript\dist\index.js doctor
```

If the runtime exposes the binary as `lobbyregisterctl`, use that shorter name.

## Auth

Set `LOBBYREGISTER_API_KEY` before live V2 calls.

The CLI also accepts `--apikey`, but prefer the environment variable so keys do
not appear in command previews. Normalized output redacts key material.

## Preferred workflow

1. Run `doctor` if you need auth, docs, endpoint health, or fair-use context.
2. Search narrowly with `search --term "<name>" --limit 3`.
3. Pick the exact `registerNumber`.
4. Run `entry source --register-number <R...>` to inspect citation URLs.
5. Run `entry dossier --register-number <R...> --grep "<term>"` for an evidence bundle.
6. Use `financial summary --register-number <R...>` for finance-focused questions.
7. Use `statements list --register-number <R...> --grep "<term>"` for submitted statement text.
8. Cross-check parliamentary claims with DIP/Bundestag tools.

## Best commands

Health:

```powershell
lobbyregisterctl doctor
```

Statistics:

```powershell
lobbyregisterctl statistics
```

Search:

```powershell
lobbyregisterctl search --term "Bundesverband Soziokultur" --limit 3
lobbyregisterctl search --term "Energie" --limit 5
```

Exact entry:

```powershell
lobbyregisterctl entry get --register-number R001255
lobbyregisterctl entry source --register-number R001255
```

Evidence bundle:

```powershell
lobbyregisterctl entry dossier --register-number R001255 --grep Soziokultur --limit 5
```

Finance:

```powershell
lobbyregisterctl financial summary --register-number R001255
```

Statements:

```powershell
lobbyregisterctl statements list --register-number R001255 --grep Soziokultur --limit 5
```

Legacy V1 wrapper remains available for comparison:

```powershell
lobbyregisterctl v1 search --param "q=Bundesverband"
```

## Output expectations

Research commands return JSON envelopes with:

- `tool`
- `command`
- `status`
- `retrievedAt`
- `request`
- `summary`
- `items` where relevant
- `sources`
- `warnings`
- `nextActions`

Broad commands should keep small limits. Use exact register numbers after
discovery.

## Evidence caveats

- Lobbyregister data is official public register data, but much of it is self-reported disclosure material.
- Financial ranges are disclosed ranges, not exact audited findings by this tool.
- Statement text and PDFs may include copyrighted material; quote only short excerpts.
- Do not infer misconduct from lobbying expenditure, funding, or donations alone.
- When discussing financial benefit, clearly distinguish disclosure data, allegations, investigations, sanctions, and convictions.
- Preserve `sourceDate`, public detail URLs, PDF URLs, and retrieval dates in research outputs.

## References

- `references/openapi-v2.yaml`
- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
- `tests/test-plan.md`
- `MANIFEST.md`
