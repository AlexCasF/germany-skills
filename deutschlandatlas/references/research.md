# Deutschlandatlas API research

## Overall description

The Deutschlandatlas is an official German government atlas for regional living
conditions. It presents indicators in map form and provides data downloads for
the underlying indicators. The public API surface used here is an ArcGIS REST
service collection hosted under `www.karto365.de`.

The data is useful for questions about regional differences in Germany:

- labour market and employment indicators
- housing and rents
- accessibility of services such as pharmacies, hospitals, and public transport
- mobility and commuting
- broadband and infrastructure
- demography and social indicators
- education and public services

The `bundesAPI/deutschlandatlas-api` OpenAPI wrapper documents a generic query
pattern where `{table}` is the indicator service name. Portal search discovers
available services. In live testing on 2026-05-18, portal search reported 233
Deutschlandatlas-related items for the broad `deutschlandatlas` query.

## Live endpoint findings

The old wrapper assumed `/MapServer/0/query`. Live testing showed that this is
not always correct. The sample unemployment table `alq_HA2023` exposes its
feature layer as id `5`:

- service metadata: `https://www.karto365.de/hosting/rest/services/alq_HA2023/MapServer?f=json`
- layer metadata: `https://www.karto365.de/hosting/rest/services/alq_HA2023/MapServer/5?f=json`
- sample query: `https://www.karto365.de/hosting/rest/services/alq_HA2023/MapServer/5/query?f=json&where=1%3D1&outFields=*&returnGeometry=false&resultRecordCount=3`

The sample service reported:

- one feature layer
- layer id `5`
- `maxRecordCount` 2000
- supported query formats: `JSON, geoJSON, PBF`
- fields including `name`, `Gebietskennziffer`, and `alq`

## Source links

- Official home: https://www.deutschlandatlas.bund.de/DE/Home/home_node.html
- Official downloads: https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html
- Indicator/download notes page: https://www.deutschlandatlas.bund.de/DE/Service/Downloads/Indikatoren_Deutschlandatlas.html
- OpenAPI wrapper repository: https://github.com/bundesAPI/deutschlandatlas-api
- Broad portal search: https://www.karto365.de/portal/sharing/rest/search?q=deutschlandatlas&f=json&num=100&start=1
