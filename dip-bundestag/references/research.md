# DIP API research notes

Retrieved: 2026-05-18

## What the API provides

The DIP API is the official Bundestag API for the Documentation and
Information System for Parliamentary Material. It provides read access to:

- `vorgang`: parliamentary proceedings and legislative process metadata
- `vorgangsposition`: proceeding positions and parliamentary process steps
- `drucksache`: printed-paper metadata
- `drucksache-text`: printed-paper metadata plus full text where available
- `plenarprotokoll`: plenary protocol metadata
- `plenarprotokoll-text`: plenary protocol metadata plus full text where available
- `aktivitaet`: parliamentary activities
- `person`: person master data

The API is the best repo source for official parliamentary records. It is not
the same thing as the live Bundestag feed, news coverage, party communication,
or speeches outside official parliamentary proceedings.

## Technical shape

Base URL:

```text
https://search.dip.bundestag.de/api/v1
```

The official short documentation says the API allows read requests through
`GET`, `HEAD`, and `OPTIONS`. Responses are JSON by default, with XML available
through the `format` parameter.

The OpenAPI file in this folder is treated as the authoritative endpoint and
parameter catalog for implementation.

## Authentication

Every request requires a valid API key. The key can be sent either as:

```text
Authorization: ApiKey <key>
```

or as:

```text
?apikey=<key>
```

The CLI should prefer `DIP_API_KEY` from the environment. Passing `--apikey`
remains supported for backward compatibility.

## Good uses

- Find official records for parliamentary proceedings.
- Check whether a statement came from an official plenary protocol.
- Retrieve printed papers and their metadata.
- Retrieve plenary protocols and full text.
- Link activities to people and parliamentary records.
- Build source-backed evidence bundles for research artifacts.

## Poor uses

- General news context.
- Live session presentation data.
- Non-parliamentary campaign statements or social media posts.
- Statistical data about society or the economy.
- Lobby register financial records.

## Source links

- API help page: https://dip.bundestag.de/%C3%BCber-dip/hilfe/api
- API short documentation PDF: https://dip.bundestag.de/documents/informationsblatt_zur_dip_api.pdf
- Terms PDF: https://dip.bundestag.de/documents/nutzungsbedingungen_dip.pdf
- OpenAPI YAML: https://search.dip.bundestag.de/api/v1/openapi.yaml

