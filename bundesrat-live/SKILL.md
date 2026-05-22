---
name: bundesrat-live
description: Use this skill for current Bundesrat public app/XML data: news, dates, plenary summaries and agenda items, upcoming sessions, members, vote-distribution context, presidium pages, and official bundesrat.de source URLs.
---

# Bundesrat Live Skill

## Purpose

Use `bundesrat-live` to work with public Bundesrat XML feeds from `bundesrat.de`. The tool is optimized for current Bundesrat research: news, events, BundesratKOMPAKT plenary summaries, agenda TOPs, Drucksachen/DIP links embedded in plenary records, member roles and profiles, upcoming sessions, vote-distribution context, and presidium pages.

## Use This Skill When

- The user asks about current Bundesrat activity, news, dates, sittings, or BundesratKOMPAKT summaries.
- The user needs current Bundesrat members, party, state, role, biography, contact, or official profile URL.
- The user needs Bundesrat agenda items, Drucksachen links, DIP links, or public source pages for citation.
- The user needs upcoming plenary-session dates.
- The user needs the current Bundesrat composition/vote-distribution graphic or presidium context.

## Do Not Use This Skill When

- The user needs complete parliamentary archives, Bundestag proceedings, printed papers, or historical plenary protocol search. Use `dip-bundestag`.
- The user needs federal law or court text. Use `rechtsinformationen-bund`.
- The user needs statistical evidence. Use the relevant statistical CLI.
- The user needs state-by-state voting behavior as a guaranteed structured dataset. Bundesrat sources note that individual Land voting behavior is not always recorded by the Bundesrat itself.

## Fast Workflow

1. Run `bundesrat-live doctor` if endpoint health, auth, or fair-use assumptions matter.
2. Use small-limit list/search commands first.
3. Expand one promising result with `page`, `news page`, `dates page`, `members dossier`, or `plenum compact/current`.
4. Use `--grep` for source snippets instead of ad hoc shell filtering.
5. Preserve `sources[]`, `retrievedAt`, `warnings[]`, and public URLs in research notes.

## High-Value Commands

```text
bundesrat-live doctor
bundesrat-live news --limit 5
bundesrat-live news search --term "Suchbegriff" --limit 3
bundesrat-live news page --url "https://www.bundesrat.de/SharedDocs/pm/2026/example.html" --grep "Suchbegriff"
bundesrat-live dates --limit 5
bundesrat-live members search --name "Mustername" --limit 3
bundesrat-live members dossier --name "Mustername" --grep "Bundesrat"
bundesrat-live plenum compact --limit 1 --top-limit 3
bundesrat-live plenum current --limit 1 --top-limit 5
bundesrat-live plenum next
bundesrat-live votes summary
bundesrat-live presidium --limit 3
bundesrat-live page --url "https://www.bundesrat.de/SharedDocs/personen/DE/laender/bw/oezdemir-cem.html" --grep "Ministerpräsident"
```

## Output Shape

Research commands return JSON envelopes with:

- `status`
- `tool`
- `command`
- `retrievedAt`
- `request`
- `summary`
- `items`
- `sources`
- `warnings`
- `nextActions`

Endpoint-compatible commands also support `--raw` when the original XML is needed.

## Agent Habits

- Prefer `news search`, `dates`, and `members search` before expanding public pages.
- Prefer `plenum compact` for BundesratKOMPAKT summaries and `plenum current` for current agenda TOPs and Drucksachen/DIP links.
- Use `plenum next` for upcoming sitting dates.
- Use `members dossier` when a person is the research subject.
- Use `page --url` only for public `bundesrat.de` URLs returned by the tool.
- Keep limits small. Treat the site as public government infrastructure, not a bulk scraping target.
- Respect the documented `robots.txt` crawl delay for crawling-like workflows.
- Treat image/media fields as copyright-sensitive and preserve attribution.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
