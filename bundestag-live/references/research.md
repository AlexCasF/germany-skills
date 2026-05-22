# Bundestag Live API Research

## Overall description

The Bundestag live/site API is a small public XML surface used by Bundestag website and app-style views. The OpenAPI wrapper identifies it as `Bundestag: Live Informationen` and lists endpoints for article details, current speaker status, plenary conference overview, committees, members, biographies, and video feeds.

Primary data exposed by the current tool:

- `speaker.xml`: current speaker/live status for plenary context.
- `conferences.xml`: current agenda-style sitting days with agenda items and article IDs.
- `ausschuesse/index.xml`: committee index with IDs, names, source images, and detail XML URLs.
- `ausschuesse/{id}.xml`: committee detail data, task text, contact data, members, news, and public source URL.
- `mdb/index.xml`: current member index with names, Bundestag IDs, fractions, states, constituencies, profile URLs, and biography XML URLs.
- `mdb/biografien/{id}.xml`: member biography data, official profile source URL, party/faction/state, biography text, disclosure fields, media URLs, and speeches RSS where available.
- `asAppV2NewsarticleXml`: Bundestag article metadata, public article URL, date, title, policy field, image source, and article text.
- `feed_vod.xml`: WebTV stream metadata for known content IDs.

## What it should be used for

Use it as a current Bundestag source layer:

- identifying Bundestag members and official profile URLs
- checking current Bundestag-published biography and disclosure text
- finding committee IDs and committee membership
- connecting current plenary agenda items to Bundestag article IDs
- expanding article IDs into official Bundestag article metadata
- getting WebTV stream metadata when an article or agenda item exposes a content ID

## What it should not be used for

Do not use it as a replacement for DIP. It is not optimized for full legislative process research, archival plenary protocols, Drucksachen, Vorgänge, or complete speech search.

Use `dip-bundestag` for:

- parliamentary proceedings
- printed papers
- plenary protocol records
- person/activity/proceeding relationships
- official plenary-session statement research

## Sources reviewed

- Bundestag live OpenAPI wrapper: https://github.com/bundesAPI/bundestag-api
- Bundestag Open Data page: https://www.bundestag.de/services/opendata/
- Bundestag website legal notice / terms section: https://www.bundestag.de/services/impressum
- Bundestag audio/video terms: https://www.bundestag.de/mediathek/nutzungsbedingungen-247892
- Bundestag image database terms: https://www.bundestag.de/bildnutz

## Implementation implications

- XML normalization is the main usability improvement.
- Broad list commands need safe default limits.
- Every normalized result should expose public source URLs as first-class fields.
- `members dossier`, `committees dossier`, and `article get` are the most agent-friendly commands.
- `article page` is useful for citation snippets, but should be described as best-effort.
- `video feed` should always include media usage warnings.
