# DIP Bundestag notes

## What this API provides

The DIP API exposes Bundestag parliamentary materials and process data, including:

- proceedings (`vorgang`)
- proceeding positions (`vorgangsposition`)
- printed papers (`drucksache`)
- printed paper full texts (`drucksache-text`)
- plenary protocols and their full texts
- activities
- person master data

## Response style

- JSON
- API key required via `Authorization` header or `apikey` query parameter
- normal list responses return up to 100 entities
- full-text list resources usually return up to 10 entities
- cursor-based pagination is used for additional results

## Good first workflows

- search proceedings first, then fetch a single proceeding by ID
- use printed paper and plenary protocol detail endpoints once you have identifiers
- use `plenarprotokoll text --document-number ... --grep ...` for official transcript snippets
- use `source` to extract PDF/XML/API source URLs before citation
- use `person dossier` or `vorgang dossier` when building an evidence bundle
- keep IDs in trace output because they are the best handles for follow-up research

## Common pitfalls

- this is not the live Bundestag site feed
- some questions fit DIP better than the Bundestag live XML API
- auth is mandatory, so the CLI should fail clearly when no key is provided
- official plenary statements are a different source category from media interviews or campaign statements
- avoid bulk cursor traversal unless explicitly requested
