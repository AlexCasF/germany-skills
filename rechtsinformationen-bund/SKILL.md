---
name: rechtsinformationen-bund
description: Use this skill for official German federal legislation, federal case law, legal literature, administrative directives, ELI/ECLI citation work, and source-backed legal research through the Rechtsinformationen des Bundes trial API.
---

# Rechtsinformationen des Bundes skill

## Purpose

Use this skill to search, inspect, cite, and quote official legal information from the German federal legal information trial service.

The service currently covers:

- federal legislation
- federal case law
- legal literature metadata
- administrative directives
- HTML, XML, and ZIP source renditions where available

## Service facts

- Base URL: `https://testphase.rechtsinformationen.bund.de/v1`
- Documentation: `https://docs.rechtsinformationen.bund.de/`
- OpenAPI: `https://testphase.rechtsinformationen.bund.de/openapi.json`
- Auth: no API key required
- Rate limit: 600 requests per minute per client IP
- Status: official trial service; dataset and endpoint behavior may change

## Preferred tool

Prefer the CLI contract.

Use the local executable when available:

```powershell
skills\rechtsinformationen-bund\bin\rechtsinformationen-bund.exe doctor
```

Portable fallbacks:

```powershell
python skills\rechtsinformationen-bund\python\rechtsinformationen-bund.py doctor
node skills\rechtsinformationen-bund\typescript\dist\index.js doctor
```

If the runtime exposes the binary as `rechtsinformationen-bund`, use that shorter name.

## When to use

Use this skill when the user asks about:

- German federal laws, decrees, or legal provisions
- federal court decisions
- ELI or ECLI identifiers
- official legal source texts
- legal-document HTML/XML renditions
- legal citations that need stable source links
- searches across legislation and case law

Do not use this skill as legal advice. Use it to retrieve and cite official materials.

## Agent workflow

1. Start with `doctor` if you need service health, auth, rate-limit, or collection counts.
2. Search narrowly first when you do not have an identifier.
3. Use `documents search --search-term "<term>" --limit 3` for cross-collection discovery.
4. Use `source` to expand a known document into API, HTML, XML, and ZIP source URLs.
5. Use `documents text` when you need source text or grep-style snippets.
6. Use `documents dossier` when preparing a source-backed answer.
7. Preserve document numbers, ECLI, ELI, court names, decision dates, and source URLs.
8. Mention that this is a trial service when the answer depends on completeness or production certainty.

## High-value commands

Health:

```powershell
rechtsinformationen-bund doctor
```

Search:

```powershell
rechtsinformationen-bund documents search --search-term "Suchbegriff" --limit 3
rechtsinformationen-bund documents search-case-law --search-term "Revision" --limit 3
rechtsinformationen-bund legislation list --search-term "Suchbegriff" --limit 3
```

Source expansion:

```powershell
rechtsinformationen-bund source --type case-law --document-number KORE600422026
rechtsinformationen-bund documents source --type legislation --eli "eli/bund/bgbl-1/2007/s2942/2024-01-01/1/deu"
```

Text and snippets:

```powershell
rechtsinformationen-bund documents text --type case-law --document-number KORE600422026 --grep Revision
```

Evidence bundle:

```powershell
rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision
rechtsinformationen-bund documents dossier --search-term "Suchbegriff" --grep Suchbegriff
```

Raw endpoint access remains available:

```powershell
rechtsinformationen-bund statistics
rechtsinformationen-bund case-law get --document-number KORE600422026
rechtsinformationen-bund case-law html --document-number KORE600422026
rechtsinformationen-bund case-law xml --document-number KORE600422026
rechtsinformationen-bund legislation get --jurisdiction bund --agent bgbl-1 --year 2007 --natural-identifier s2942 --point-in-time 2024-01-01 --version 1 --language deu
```

## Output expectations

Research commands return JSON envelopes with:

- `tool`
- `command`
- `status`
- `retrievedAt`
- `request`
- `summary`
- `sources`
- `warnings`
- `nextActions`

Compact search also returns normalized top-level `items` with identifiers, text-match hints, and source links.

Raw raw endpoint commands return the upstream API response directly unless the command is an HTML/XML source command.

## Good habits

- Keep `--limit` or `size` small during discovery.
- Prefer official source URLs emitted by the CLI over ad hoc web search.
- Use `documents dossier` before drafting legal research answers.
- Use `documents text --grep` to collect short evidence snippets instead of dumping full documents.
- Cite every legal claim with at least one API or HTML/XML source URL.
- Distinguish search-result snippets from full source text.
- Do not assume the trial dataset is complete.

## Known documentation mismatches

- Some prose docs mention `/v1/search`; the live OpenAPI currently uses `/v1/document/lucene-search`.
- Some examples use `limit`; live endpoints use `size`, while the CLI accepts `--limit` and maps it safely.
- Detail lookup uses document-number paths for case law in the current OpenAPI.
- Always trust live OpenAPI and tested CLI behavior over older prose examples when they disagree.

## References

- `references/openapi.json`
- `references/docs.md`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
