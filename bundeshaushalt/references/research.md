# Bundeshaushalt API Research

## Overall Description

Bundeshaushalt Digital is the official web application for exploring the German federal budget. It lets users inspect federal revenue and expenditure by budget year and by several classifications: Einzelplan/ministry structure, functional area, and economic group. The live application exposes JSON budget hierarchy data through `/internalapi/budgetData`.

The API is useful for:

- ministry-level expenditure and revenue structure
- individual budget chapters and titles
- planned budget values (`target`/Soll)
- actual accounting values (`actual`/Ist), where available
- comparisons of one budget node across years
- finding a budget line by label and then citing the exact API request

The API is not a replacement for statistical APIs. Budget values are nominal euro amounts and need statistical context for inflation-adjusted, per-capita, labor-market, or macroeconomic interpretation.

## Source Links

- Official application: https://www.bundeshaushalt.de/DE/Bundeshaushalt-digital/bundeshaushalt-digital.html
- Live endpoint: https://bundeshaushalt.de/internalapi/budgetData
- Download portal: https://www.bundeshaushalt.de/DE/Download-Portal/download-portal.html
- User notes: https://www.bundeshaushalt.de/DE/Service/Benutzerhinweise/benutzerhinweise.html
- Imprint: https://www.bundeshaushalt.de/DE/Service/Impressum/impressum.html
- Privacy page: https://www.bundeshaushalt.de/DE/Service/Datenschutz/datenschutz.html
- robots.txt: https://www.bundeshaushalt.de/robots.txt
- BMF federal budget overview: https://www.bundesfinanzministerium.de/Web/DE/Themen/Oeffentliche_Finanzen/Bundeshaushalt/bundeshaushalt.html
- BMF data portal usage notes: https://www.bundesfinanzministerium.de/Datenportal/Nutzungshinweise/nutzungshinweise.html
- OpenAPI wrapper source: https://github.com/anetz89/bundeshaushalt-api

## OpenAPI Versus Live Endpoint

The bundled `openapi.yaml` documents a small `GET /internalapi/budgetData` surface and lists years only through 2021. Live probes showed newer target years are available. The CLIs therefore keep the OpenAPI file as a reference, but use live endpoint behavior for `years list`, `doctor`, and examples.

## Tested Request Shapes

Root spending tree:

```text
GET https://bundeshaushalt.de/internalapi/budgetData?year=2026&account=expenses&quota=target&unit=single
```

Ministry node:

```text
GET https://bundeshaushalt.de/internalapi/budgetData?year=2025&account=expenses&quota=target&unit=single&id=11
```

Budget title:

```text
GET https://bundeshaushalt.de/internalapi/budgetData?year=2025&account=expenses&quota=target&unit=single&id=110168112
```

The title-level response can include `related` keys such as `agency`, `function`, and `group`.
