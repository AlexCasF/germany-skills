# Rechtsinformationen des Bundes API research notes

Retrieved: 2026-05-18

## What the API provides

The Rechtsinformationen des Bundes preview API provides read access to legal
information from the German federal legal information portal.

It currently exposes:

- federal legislation and legislation expressions
- federal case law and court decisions
- legal literature metadata
- administrative-directive metadata
- cross-collection document listing and search
- HTML, XML, and ZIP encodings for many records
- service statistics

The portal is a trial service. The public website says the data stock is not
yet complete and the service is still under development. Existing websites such
as Gesetze-im-Internet and Rechtsprechung-im-Internet remain relevant for
production-grade legal research.

## Technical shape

Base URL:

```text
https://testphase.rechtsinformationen.bund.de/v1
```

OpenAPI:

```text
https://testphase.rechtsinformationen.bund.de/openapi.json
```

The API uses JSON by default. Many entity endpoints also provide `.html`,
`.xml`, and sometimes `.zip` encodings.

List/search endpoints use Hydra-style collection envelopes with `member` and
`view` fields.

## Authentication

The current documentation says the API is open and does not require an API key.

## Good uses

- Find and cite German federal court decisions.
- Search federal legislation by term or ELI.
- Retrieve official HTML/XML text for legal records.
- Build legal evidence bundles with metadata, source URLs, and snippets.
- Inspect standards such as ELI, ECLI, LegalDocML, JSON-LD, and schema.org.

## Poor uses

- Claims requiring legal advice or interpretation beyond sourced text.
- State-level or municipal law unless represented in the federal portal.
- Final production legal research without checking current official sources.
- High-volume scraping beyond the documented rate limits.

## Source links

- Portal: https://testphase.rechtsinformationen.bund.de/
- API documentation: https://docs.rechtsinformationen.bund.de/
- Getting started: https://docs.rechtsinformationen.bund.de/get-started
- Endpoints: https://docs.rechtsinformationen.bund.de/endpoints
- Formats: https://docs.rechtsinformationen.bund.de/guides/formats
- Pagination: https://docs.rechtsinformationen.bund.de/guides/pagination
- Rate limiting: https://docs.rechtsinformationen.bund.de/guides/rate-limiting
- OpenAPI JSON: https://testphase.rechtsinformationen.bund.de/openapi.json
- DigitalService project context: https://digitalservice.bund.de/projekte/neues-rechtsinformationssystem

