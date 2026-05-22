# Bundestag Live Notes

## What this API provides

This API wraps public Bundestag XML feeds for:

- current plenary speaker status
- plenary conference and agenda overview
- committee overview and committee details
- member overview and member biographies
- Bundestag article details
- WebTV stream metadata for known content IDs

## Response style

- Upstream responses are XML.
- The CLIs normalize common workflows into compact JSON envelopes.
- Use `--raw` on endpoint-compatible commands when exact upstream XML is needed.

## Common pitfalls

- This is not DIP. Use DIP for archival plenary protocols, Drucksachen, Vorgänge, and complete parliamentary-document research.
- The member index is large. Search first, then fetch one dossier.
- Bundestag profile disclosures can include self-reported fields and publication-rule caveats.
- Public HTML page extraction is best-effort. Prefer article XML for structured metadata.
- Video, image, and website content have usage terms beyond simple API access.

## Output guidance

When citing results:

- include the public Bundestag source URL when available
- include the XML/API URL when relevant to reproducibility
- include `retrievedAt`
- preserve caveats from `warnings[]`
