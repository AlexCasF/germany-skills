# Abgeordnetenwatch notes

## What this API provides

The abgeordnetenwatch API exposes:

- parliaments
- parliament periods
- politicians
- candidacies and mandates
- polls

## Response style

- JSON
- no auth visible in the public documentation

## Supported interaction patterns from the docs

- list and detail retrieval
- filtering with operators
- nested filtering via referenced entities
- pagination and sorting
- `related_data` expansions

## Common pitfalls

- nested filters can become complex quickly
- list endpoints are easier to reason about before requesting detail records
