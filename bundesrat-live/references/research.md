# Bundesrat Live API Research

## What The API Provides

The Bundesrat live/app API is a set of public XML feeds served from `https://www.bundesrat.de`. The local OpenAPI file describes ten documented GET endpoints under the iOS/app path family.

Main data surfaces:

| Area | Endpoint / command | Data provided |
| --- | --- | --- |
| Feed catalog | `startlist` | App feed names, feed URLs, dates, layout dates, and cache/hash metadata. |
| News | `news` | Press releases and public text items with titles, dates, summaries, embedded detail HTML, image metadata, and public source URLs. |
| Dates | `dates` | Event and committee dates with start/stop dates, titles, descriptions, detail HTML, and public source URLs. |
| Plenary summary | `plenum compact` | BundesratKOMPAKT session summary, selected TOPs, decision text, document links, video links, and public page URLs. |
| Current plenary | `plenum current` | Current agenda/TOP records with Drucksachen, committee involvement, decision tenor, DIP links, and PDF links embedded in detail HTML. |
| Chronological plenary | `plenum chronological` | Chronological plenary/TOP-oriented data from the app feed. |
| Upcoming sessions | `plenum next` | Upcoming plenary sitting dates, usually as a table embedded in a page-like item. |
| Members | `members` | Current Bundesrat members and representatives with name, party, state, role details, biography, contact information, profile URL, and image metadata. |
| Votes/composition | `votes summary` | Current Bundesrat composition/vote-distribution context and image metadata. |
| Presidium | `presidium` | Presidium, presidency, standing advisory board, secretariat/directorate, and historical presidency context pages. |

## What It Should Be Used For

Use this tool for current Bundesrat research where the source trail matters:

- finding official Bundesrat news and event source pages
- retrieving BundesratKOMPAKT summaries for recent sittings
- inspecting current agenda items and linked Drucksachen/DIP records
- finding current members and official profile pages
- listing upcoming plenary-session dates
- collecting source URLs for citation-ready research artifacts

## What It Should Not Be Used For

- Full legislative proceeding history: use `dipctl`.
- Full legal text: use `rechtsinformationenctl`.
- Bundestag member/parliamentary data: use `bundestagctl` or `dipctl`.
- Statistical evidence: use Destatis, Regionalatlas, Deutschlandatlas, Dashboard Deutschland, or related statistical tools.

## Research Sources

- Bundesrat OpenAPI wrapper: https://github.com/bundesAPI/bundesrat-api
- Bundesrat website: https://www.bundesrat.de/DE/homepage/homepage-node.html
- Bundesrat service.bund.de profile: https://www.service.bund.de/Content/DE/DEBehoerden/B/BR/Bundesrat.html
- Bundesrat impressum: https://www.bundesrat.de/DE/service-navi/impressum/impressum-node.html
- Bundesrat privacy policy: https://www.bundesrat.de/DE/service-navi/datenschutz/datenschutz-node.html
- Bundesrat robots.txt: https://www.bundesrat.de/robots.txt
