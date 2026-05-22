---
name: api-dip-bundestag
description: Use this skill for Bundestag parliamentary materials and DIP research, including proceedings, printed papers, plenary protocols, activities, and person master data.
---

# DIP Bundestag skill

## Purpose

This skill helps use the DIP Bundestag API for parliamentary process and document research.

## Use this skill when

- the user asks about Bundestag proceedings or legislative dossiers
- the user needs printed papers or plenary protocol metadata
- the user needs full text for parliamentary material
- the user needs person master data from DIP

## CLI organization

The CLI binary is `dipctl`.

Typical workflow:

1. Inspect `dipctl --help`.
2. Run `dipctl doctor` if auth or endpoint status is uncertain.
3. Choose the relevant entity family such as `vorgang`, `drucksache`, or `plenarprotokoll`.
4. Prefer list/search endpoints before detail endpoints when exploring the corpus.
5. Use `source`, `text`, or `dossier` commands before making evidence claims.
6. Preserve identifiers so the user can trace the result back to the original parliamentary material.

## Important notes

- The API requires an API key.
- Prefer `DIP_API_KEY` from the environment.
- The API is JSON-first and strongly suited to metadata and text retrieval.
- DIP is the best fit in this repo for Bundestag process research, not live session presentation data.
- Official plenary-session claims should be checked against `plenarprotokoll-text`, not news or outside quotes.

## Useful v2 commands

- `dipctl doctor`
- `dipctl person search --name "Name"`
- `dipctl person dossier --id <id>`
- `dipctl vorgang dossier --id <id>`
- `dipctl source --type plenarprotokoll --document-number "20/139"`
- `dipctl plenarprotokoll text --document-number "20/139" --grep "Suchbegriff"`

## References

- `references/openapi.yaml`
- `references/notes.md`
