# Python Test Results

Runtime:

```powershell
python skills\tagesschau\python\tagesschau.py
```

| # | Case | Exit | Result | Note |
| --- | --- | --- | --- | --- |
| 1 | Root help | 0 | Pass | Printed root help. |
| 2 | Article help | 0 | Pass | Printed article flags. |
| 3 | Source metadata | 0 | Pass | Returned API, OpenAPI, public service, usage, and CC URLs. |
| 4 | Fields | 0 | Pass | Returned feed, ressort, region, and article-field guidance. |
| 5 | Doctor | 0 | Pass | Homepage, news, channels, and search endpoints were healthy. |
| 6 | Homepage | 0 | Pass | Returned 1 compact item from homepage. |
| 7 | News filter | 0 | Pass | Returned 1 compact `inland` item. |
| 8 | Channels | 0 | Pass | Returned 1 compact channel entry. |
| 9 | Legacy search params | 0 | Pass | `--param searchText=Bundestag --param pageSize=1` worked. |
| 10 | Article grep | 0 | Pass | Returned a bounded snippet for one returned article URL. |

Extra smoke check: `article source --url <detailsweb>` passed.
