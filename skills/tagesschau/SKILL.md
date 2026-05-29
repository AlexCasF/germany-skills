---
name: tagesschau
description: Use this skill for current Tagesschau news context, homepage/news/channel/search feeds, and bounded article expansion with strong copyright and source-use caveats.
---

# Tagesschau Skill

## Purpose

Use this skill when a user needs current-news context from Tagesschau, wants to search recent Tagesschau coverage, or needs a bounded article/source expansion for a URL returned by the Tagesschau JSON feeds.

Treat Tagesschau as a context layer. Do not use it as the sole evidence for parliamentary, legal, fiscal, statistical, or register claims when primary official sources are available.

## First Moves

Start with health and source/usage context:

```powershell
& .\skills\tagesschau\bin\tagesschau.exe doctor
& .\skills\tagesschau\bin\tagesschau.exe source
& .\skills\tagesschau\bin\tagesschau.exe fields
```

Alternative runtimes:

```powershell
python skills\tagesschau\python\tagesschau.py doctor
node skills\tagesschau\typescript\dist\index.js doctor
```

## Main Workflow

Use this sequence:

1. Run `doctor` to check live endpoint health and usage limits.
2. Use `search`, `homepage`, `news`, or `channels` to find relevant items.
3. Follow a result's `nextActions`.
4. Use `article source` for citation metadata.
5. Use `article get --grep` for short, bounded snippets.
6. Use `article dossier` only when an article should become a context artifact.
7. Verify official claims against primary tools such as DIP, Bundestag, Destatis, Bundeshaushalt, Lobbyregister, or Rechtsinformationen.

## Commands

- `doctor`: checks homepage, news, channels, search endpoints and reports auth, rate-limit, and reuse warnings.
- `source`: returns canonical API, OpenAPI, public service, usage, and Creative Commons references.
- `fields`: explains feed commands, ressort values, region codes, and core article fields.
- `homepage --limit N`: compact homepage feed.
- `news --ressort inland --limit N`: compact news feed. Also supports `--regions`.
- `channels --limit N`: compact channel feed.
- `search --text TERM --limit N`: compact search feed.
- `search --param searchText=TERM --param pageSize=N`: raw-compatible parameter path.
- `article source --url URL`: cite/source metadata without fetching article content.
- `article get --url URL --grep TERM --limit N`: bounded snippets from one article JSON/detail URL.
- `article dossier --url URL --limit N`: metadata, snippets, sources, caveats, and next actions.

## Examples

```powershell
& .\skills\tagesschau\bin\tagesschau.exe homepage --limit 5
& .\skills\tagesschau\bin\tagesschau.exe news --ressort inland --limit 5
& .\skills\tagesschau\bin\tagesschau.exe search --text "Suchbegriff" --limit 5
& .\skills\tagesschau\bin\tagesschau.exe search --param searchText=Suchbegriff --param pageSize=5
& .\skills\tagesschau\bin\tagesschau.exe article source --url "https://www.tagesschau.de/inland/example-100.html"
& .\skills\tagesschau\bin\tagesschau.exe article get --url "https://www.tagesschau.de/inland/example-100.html" --grep "Suchbegriff" --limit 3
```

## Interpretation And Safety Rules

- Always preserve `detailsweb` or `sourceUrl` for citation.
- Use `date`, `title`, `topline`, and public URL in citations.
- Do not reproduce long article text.
- Use snippets sparingly and only when they are necessary for analysis.
- Respect the documented 60 requests/hour ceiling.
- Publication/reuse is restricted except for content explicitly released under Creative Commons.
- Do not treat a Tagesschau article as a primary official record when a government, parliament, court, register, or statistical source exists.

## URL Handling

`article get`, `article source`, and `article dossier` accept either:

- public URLs such as `https://www.tagesschau.de/...-100.html`
- API URLs such as `https://www.tagesschau.de/api2u/...-100.json`

The CLI converts between public and API URLs and returns both.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
