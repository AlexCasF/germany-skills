---
name: bundestag-live
description: Use this skill for current Bundestag website/app XML feeds: live plenary agenda, current speaker status, member profiles and disclosures, committee pages, article details, and WebTV feed metadata.
---

# Bundestag Live Skill

## Purpose

Use `bundestag-live` to work with public Bundestag live/site XML surfaces. This tool is best for current Bundestag presentation data: members, biographies, disclosure snippets, committees, agenda article IDs, Bundestag article pages, and media feed metadata.

## Use this skill when

- The user asks for current Bundestag members, profile URLs, biographies, parties, factions, constituencies, or Bundestag-published disclosure fields.
- The user asks about Bundestag committees, their tasks, members, news, or source pages.
- The user needs current or upcoming Bundestag plenary agenda items from the live app feed.
- The user needs to expand a Bundestag agenda article ID into structured article metadata and a public source URL.
- The user needs Bundestag WebTV stream metadata from a known content ID.

## Do not use this skill when

- The user needs complete parliamentary proceedings, legislative dossiers, printed papers, plenary protocols, or archival speech search. Use `dip-bundestag` instead.
- The user needs Bundesrat proceedings. Use `bundesrat-live`.
- The user needs statistical evidence. Use the relevant statistical CLI.
- The user needs legal text. Use `rechtsinformationen-bund`.

## Fast workflow

1. Run `bundestag-live doctor` if endpoint health or usage assumptions matter.
2. Search or list with small limits.
3. Expand one result with `dossier`, `biography`, `committees dossier`, or `article get`.
4. Use `article page` only when the public HTML page itself is needed for citation snippets.
5. Preserve `sources[]`, `retrievedAt`, and `warnings[]` in research notes.

## High-value commands

```text
bundestag-live members search --name "Mustername" --limit 3
bundestag-live members dossier --id 2022 --grep "Suchbegriff"
bundestag-live committees search --term "Arbeit" --limit 5
bundestag-live committees dossier --id a11 --member-limit 5 --news-limit 3
bundestag-live plenum conferences --limit 2 --item-limit 5
bundestag-live article get --article-id 1174778 --grep "Meinungsfreiheit"
bundestag-live article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778" --grep "Meinungsfreiheit"
bundestag-live video feed --content-id 7529016
```

## Output shape

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

## Agent habits

- Prefer `members search` before `members dossier` unless you already have a Bundestag `mdbID`.
- Prefer `committees search` before `committees dossier` unless you already have an ID such as `a11`.
- Use `plenum conferences` to discover agenda article IDs, then expand with `article get`.
- Use `--grep` for biography, disclosure, committee, and article snippets instead of ad hoc shell filtering.
- Keep limits small. The member index is broad, and committee detail records can include many members and news items.
- Treat WebTV, images, and public page content as subject to Bundestag usage terms. Do not imply unrestricted reuse.

## References

- `references/openapi.yaml`
- `references/notes.md`
- `references/research.md`
- `references/rate-limits-and-terms.md`
