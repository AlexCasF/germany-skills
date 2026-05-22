# Abgeordnetenwatch API research

Retrieved: 2026-05-18

## What the API provides

The public API exposes machine-readable JSON data from abgeordnetenwatch.de.
The official documentation describes data such as:

- parliament periods and elections
- politicians and candidates
- mandates and candidacies
- voting behavior and named votes
- side jobs and side-job organizations
- related-data expansion paths

The raw CLI covered:

- `parliaments`
- `parliament-periods`
- `politicians`
- `candidacies-mandates`
- `polls`

The docs and live API also expose useful research entities such as:

- `sidejobs`
- `sidejob-organizations`
- `votes`
- `parties`
- `committees`
- `committee-memberships`
- `fractions`
- `topics`
- `cities`
- `countries`

## Response style

Responses contain:

- `meta`
- `data`

The `meta.abgeordnetenwatch_api` block includes API version, changelog,
license, license link, and documentation link.

The live API reported version `2.8.2` during testing.

## Useful behavior tested

Search by name:

```text
https://www.abgeordnetenwatch.de/api/v2/politicians?first_name[cn]=Muster&last_name[cn]=Name&pager_limit=3
```

Exact politician:

```text
https://www.abgeordnetenwatch.de/api/v2/politicians/<politician-id>
```

Mandates for a politician:

```text
https://www.abgeordnetenwatch.de/api/v2/candidacies-mandates?politician=<politician-id>&range_end=3
```

Side jobs for a mandate:

```text
https://www.abgeordnetenwatch.de/api/v2/sidejobs?mandates=<mandate-id>&range_end=5
```

Public profile page:

```text
https://www.abgeordnetenwatch.de/profile/example
```

## Interpretation caveats

- abgeordnetenwatch is a transparency platform and not an official parliamentary archive like DIP.
- Profile pages are useful public-source context, but final official claims should be cross-checked when possible.
- Side-job data can show disclosed outside income or roles, but it does not by itself prove corruption.
- Side-job filters are mandate-oriented; to find side jobs for a politician, first fetch mandates and then query side jobs by mandate ID.
- Broad list endpoints can return large responses; use small limits during discovery.

## Sources

- https://www.abgeordnetenwatch.de/api
- https://www.abgeordnetenwatch.de/api/response
- https://www.abgeordnetenwatch.de/api/version-changelog/aktuell
- https://www.abgeordnetenwatch.de/api/entitaeten/sidejob
- https://www.abgeordnetenwatch.de/api/entitaeten/sidejob-organization
