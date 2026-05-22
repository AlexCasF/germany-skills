# Tagesschau Notes

## What The API Provides

The Tagesschau API exposes current news and media feed data in JSON. It is useful for discovering current-news context and article URLs.

Main endpoints:

- `https://www.tagesschau.de/api2u/homepage`
- `https://www.tagesschau.de/api2u/news`
- `https://www.tagesschau.de/api2u/channels`
- `https://www.tagesschau.de/api2u/search`

## Common Fields

- `title`: article or item title.
- `topline`: short topical heading.
- `date`: publication/update timestamp.
- `details`: JSON detail URL.
- `detailsweb`: public article URL.
- `shareURL`: sometimes points to a regional broadcaster article.
- `firstSentence`: short teaser where available.
- `content`: article body blocks in detail JSON.
- `tags`: topical tags.
- `type`: story, video, audio, channel, or related item type.

## Agent Workflow

```text
doctor -> search/homepage/news -> article source -> article get --grep -> primary-source verification
```

Use `article get` only for bounded snippets. Do not use it to dump or republish complete articles.

## Important Caveats

- The API is open and did not require auth during testing.
- Published documentation says more than 60 requests per hour are not allowed.
- Use is private/non-commercial; publication is restricted except where content is explicitly Creative Commons licensed.
- Treat Tagesschau as a context/news source, not a primary official dataset.
- Article text snippets should be short and cited with the public article URL.
