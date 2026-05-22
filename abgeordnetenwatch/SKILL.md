---
name: api-abgeordnetenwatch
description: Use this skill for public German politician profiles, mandates, candidacies, side jobs, voting/context data, and profile-page evidence from abgeordnetenwatch.de.
---

# Abgeordnetenwatch skill

## Purpose

Use this skill to search and cite public transparency data from abgeordnetenwatch.de, especially politician profiles, mandates, candidacies, side jobs, voting/context data, and public profile-page text.

## Service facts

- Base URL: `https://www.abgeordnetenwatch.de/api/v2`
- Documentation: `https://www.abgeordnetenwatch.de/api`
- Auth: no API key required
- License: CC0 1.0 in API metadata
- Published exact rate limit: not found in official docs
- Result limits: default 100; `range_end` and `pager_limit` up to 1,000 in official docs

## Use this when

- the user asks about a German politician's public abgeordnetenwatch profile
- the user needs mandates or candidacies for a politician
- the user needs disclosed side-job or outside-income evidence
- the user needs a profile-page URL, page text, or bounded snippets
- the user needs public transparency context before checking official records

## Do not use this when

- the user needs an official parliamentary archive; prefer DIP or Bundestag tools
- the user needs official legislative document metadata; prefer DIP
- the user needs legal text; prefer Rechtsinformationen des Bundes
- side-job data alone is being treated as proof of misconduct

## Preferred tool

Prefer the 2.0 CLI contract.

Use the local executable when available:

```powershell
skills\abgeordnetenwatch\bin\abgeordnetenwatchctl-2.0.exe doctor
```

Portable fallbacks:

```powershell
python skills\abgeordnetenwatch\python\abgeordnetenwatchctl.py doctor
node skills\abgeordnetenwatch\typescript\dist\index.js doctor
```

If the runtime exposes the binary as `abgeordnetenwatchctl`, use that shorter name.

## Preferred workflow

1. Run `doctor` if you need auth, license, or fair-use context.
2. Search with `politicians search --name "<name>" --limit 3`.
3. Use `politicians source --id <id>` to inspect API and public profile URLs.
4. Use `politicians page --id <id> --grep "<term>"` for profile-page snippets.
5. Use `politicians dossier --id <id> --grep "<term>"` for an evidence bundle.
6. Cross-check official parliamentary claims with DIP or Bundestag tools.

## Best commands

Health:

```powershell
abgeordnetenwatchctl doctor
```

Search:

```powershell
abgeordnetenwatchctl politicians search --name "Alice Weidel" --limit 3
abgeordnetenwatchctl politicians search --name "Gauweiler" --limit 3
```

Source/page:

```powershell
abgeordnetenwatchctl politicians source --id 108379
abgeordnetenwatchctl politicians page --id 108379 --grep Nebentätigkeiten
abgeordnetenwatchctl politicians page --url https://www.abgeordnetenwatch.de/profile/alice-weidel --grep Nebentätigkeiten
```

Evidence bundle:

```powershell
abgeordnetenwatchctl politicians dossier --id 108379 --grep Nebentätigkeiten --limit 5
abgeordnetenwatchctl sidejobs for-politician --id 108379 --limit 5
abgeordnetenwatchctl mandates for-politician --id 108379 --limit 5
```

Raw endpoint access remains available:

```powershell
abgeordnetenwatchctl parliaments list --limit 5
abgeordnetenwatchctl politicians get --id 108379
abgeordnetenwatchctl sidejobs list --limit 5
abgeordnetenwatchctl sidejobs get --id 20846
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

Legacy endpoint commands return upstream API JSON directly.

## Evidence caveats

- abgeordnetenwatch is a transparency platform, not the official Bundestag archive.
- Public profile pages are useful citation sources, but official parliamentary claims should be checked against DIP or Bundestag records.
- Side-job data is disclosure evidence; do not infer corruption or illegality from income fields alone.
- Side jobs are connected through mandates, so the reliable path is politician -> mandates -> sidejobs.
- Invalid entity IDs can return upstream HTTP 500; handle this as a structured failed lookup.

## Safety rules

- Keep `--limit` small during discovery.
- Do not crawl profile pages broadly.
- Prefer exact IDs after search.
- Preserve API URLs, profile URLs, retrieval dates, and caveats in answers.
- If discussing financial benefit, clearly distinguish disclosed side-job income, allegations, investigations, sanctions, and convictions.

## References

- `references/docs.md`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
- `tests/test-plan.md`
- `MANIFEST.md`
