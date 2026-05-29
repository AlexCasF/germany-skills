# Bundestag Lobbyregister research notes

## What the API provides

The Bundestag Lobbyregister API provides machine-readable public data from the
official register for interests represented toward the German Bundestag and the
Federal Government.

The useful research objects are:

- register entries for organizations, persons, associations, companies, and networks
- identity and contact metadata
- activity descriptions and fields of interest
- people involved in lobbying work
- financial expense ranges by fiscal year
- main funding sources
- public allowances
- donations and membership fees where disclosed
- annual-report URLs
- regulatory projects
- statements and statement PDFs
- contracts and code-of-conduct metadata
- register-wide statistics

## Current endpoint surface

The current OpenAPI document exposes:

- `GET /registerentries`
- `GET /registerentries/{registerNumber}`
- `GET /registerentries/{registerNumber}/{version}`
- `GET /statistics/registerentries`

The free-text search endpoint returns rich full-detail records and can be large.
The CLI therefore defaults to compact summaries and small limits.

## Common research workflow

1. Run `doctor` to confirm auth and endpoint health.
2. Run `search --term "<name>" --limit 3`.
3. Pick an exact `registerNumber`.
4. Run `entry source --register-number <R...>` to inspect citation URLs.
5. Run `entry dossier --register-number <R...> --grep "<term>"`.
6. Use `financial summary` for finance-focused questions.
7. Use `statements list --grep` when the question concerns submitted positions or project statements.

## Interpretation caveats

Lobbyregister data is official published register data, but much of it is
self-reported disclosure material. It is strong evidence for what is recorded
in the register, not standalone proof of wrongdoing.

For contentious claims, distinguish:

- disclosed financial ranges
- donations or membership fees
- public allowances
- contracts
- published statements
- regulatory projects
- sanctions or code-of-conduct flags
- allegations or media reporting from outside the register

## Sources

- `https://www.lobbyregister.bundestag.de/informationen-und-hilfe/open-data-1049716`
- `https://api.lobbyregister.bundestag.de/rest/v2/swagger-ui/`
- `https://api.lobbyregister.bundestag.de/rest/v2/R2.21-de.yaml`
- `https://github.com/bundesAPI/bundestag-lobbyregister-api`
