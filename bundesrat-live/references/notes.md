# Bundesrat API Notes

## High-Level Summary

This API exposes Bundesrat public app/XML feeds from `bundesrat.de`. It is useful for current Bundesrat-facing information rather than complete archival research.

The most useful data areas are:

- feed catalog / app startlist
- latest Bundesrat news and press/public text pages
- events and committee dates
- BundesratKOMPAKT plenary summaries and selected agenda TOPs
- current plenary agenda items with Drucksachen and DIP links in embedded detail HTML
- upcoming plenary dates
- current Bundesrat members, roles, states, parties, biographies, and profile URLs
- current vote-distribution/composition context
- presidium and institutional context pages

## Common Pitfalls

- This is not the full parliamentary archive. Use `dipctl` for archive-grade proceedings and printed papers.
- Many fields are embedded HTML inside XML CDATA. Prefer normalized commands before raw XML.
- Public pages and images can include copyright notices; preserve attribution.
- Individual Bundesrat voting behavior by Land is not always recorded centrally by the Bundesrat.
- `robots.txt` publishes `Crawl-delay: 30`; avoid rapid page expansion.

## Output Guidance

When summarizing results:

- include the public source URL when present
- include retrieval time from `retrievedAt`
- distinguish XML feed data from public HTML page extraction
- keep Drucksachen, DIP, and PDF links visible
- cite Bundesrat as the source for official Bundesrat pages
