# Rate Limits, Auth, And Terms

## Authentication

No authentication was required during live testing of:

- `https://www.tagesschau.de/api2u/homepage`
- `https://www.tagesschau.de/api2u/news`
- `https://www.tagesschau.de/api2u/channels`
- `https://www.tagesschau.de/api2u/search`

## Published Rate Limit

The published API documentation states that it is not permitted to make more than 60 requests per hour.

Source:

- https://github.com/bundesAPI/tagesschau-api

## Usage Restrictions

The same published API documentation says use is allowed for private, non-commercial purposes and publication is not allowed except for offers explicitly released under a Creative Commons license.

Tagesschau also has a Creative Commons page for selected videos. That page explains that selected videos are available under CC BY-SA 4.0, with attribution, documentation of changes, share-alike requirements, and no misleading implication of Tagesschau endorsement.

Sources:

- https://github.com/bundesAPI/tagesschau-api
- https://www.tagesschau.de/multimedia/video/creative-commons-index-100.html
- https://www.tagesschau.de/infoservices/rssfeeds

## Agent Fair-Use Rules

- Keep `--limit` low.
- Prefer search/feed summaries before article expansion.
- Do not crawl large topic histories automatically.
- Do not fetch full articles unless needed for a specific cited snippet.
- Preserve public article URLs in final citations.
- Do not republish long article text in artifacts.
- Use primary official sources for final factual verification whenever possible.
