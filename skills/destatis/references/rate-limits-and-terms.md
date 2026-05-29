# Rate limits, auth, and terms

## Terms and license

The official Destatis Open Data page says GENESIS-Online is generally free and
usable without registration under:

```text
Datenlizenz Deutschland - Namensnennung - Version 2.0
```

It also says all users access the same data stock.

## API access

The official Destatis Open Data page describes the GENESIS webservice as an
automated interface for integrating GENESIS-Online data into automated
processes. It says Destatis currently offers a RESTful/JSON interface with many
methods. The English page also mentions SOAP/XML.

The API is credential-shaped. The CLI supports:

- `DESTATIS_USERNAME`
- `DESTATIS_PASSWORD`
- `--username`
- `--password`
- fallback `GAST/GAST` for public discovery

Do not store personal credentials in repo files or logs.

## Rate limits

No exact request-per-minute or daily quota was found in the official Destatis
docs reviewed.

Live `logincheck` output said that when there are more than 3 parallel requests,
requests running longer than 15 minutes are terminated. Treat this as a strong
fair-use hint:

- avoid parallel broad calls
- use small `pagelength`
- inspect metadata before data
- prefer `search` and `source` before table downloads
- request only the table/time range/dimensions needed

## Support

The official GENESIS contact page directs technical questions about
registration, handling, or API use to the GENESIS-Online user service.

## Sources

- `https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html`
- `https://www.destatis.de/EN/Service/OpenData/api-webservice.html`
- `https://www.destatis.de/DE/Service/Kontakt/Genesis/Servicekontakt-GENESIS.html`
