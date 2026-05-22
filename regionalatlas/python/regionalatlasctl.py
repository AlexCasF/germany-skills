#!/usr/bin/env python3
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "regionalatlasctl"
MAP_SERVER_URL = "https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer"
QUERY_ENDPOINT = MAP_SERVER_URL + "/dynamicLayer/query"
CATALOG_URL = "https://regionalatlas.statistikportal.de/taskrunner/services.json"
THESAURUS_URL = "https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv"
APP_URL = "https://regionalatlas.statistikportal.de/"
STATISTIKPORTAL_URL = "https://www.statistikportal.de/de/karten/regionalatlas-deutschland"
DESTATIS_URL = "https://www.destatis.de/DE/Service/Statistik-Visualisiert/RegionalatlasAktuell.html"
OPEN_DATA_URL = "https://www.statistikportal.de/de/open-data"
MAPS_GEODATA_URL = "https://www.destatis.de/DE/Service/OpenData/karten-geodaten.html"
OPENAPI_REPO_URL = "https://github.com/bundesAPI/regionalatlas-api"
DEFAULT_LIMIT = 10
SAFE_LIMIT = 100

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")


class CLIError(Exception):
    def __init__(self, exit_code, code, message):
        super().__init__(message)
        self.exit_code = exit_code
        self.code = code
        self.message = message


def main(argv):
    if not argv or is_help(argv[0]):
        print_root_help()
        return 0
    if is_help(argv[-1]):
        print_help(argv[:-1])
        return 0
    try:
        if argv[0] == "doctor":
            run_doctor(argv[1:])
        elif argv[:2] == ["indicators", "list"]:
            run_indicators_list(argv[2:])
        elif argv[:2] == ["indicators", "search"]:
            run_indicators_search(argv[2:])
        elif argv[:2] == ["indicator", "get"]:
            run_indicator_get(argv[2:])
        elif argv[0] == "fields":
            run_fields(argv[1:])
        elif argv[0] == "sample":
            run_sample(argv[1:])
        elif argv[0] == "source":
            run_source(argv[1:])
        elif argv[0] == "dossier":
            run_dossier(argv[1:])
        elif argv[0] == "query-builder":
            run_query_builder(argv[1:])
        elif argv[0] == "explain-field":
            run_explain_field(argv[1:])
        elif argv[0] == "query":
            run_raw_query(argv[1:])
        else:
            raise CLIError(2, "unknown_command", "unknown command; run regionalatlasctl --help")
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""regionalatlasctl -- Regionalatlas Deutschland research CLI

Purpose
  Discover and query official Regionalatlas indicators from the statistical
  offices of the German federation and states.

Fast paths
  regionalatlasctl doctor
  regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5
  regionalatlasctl fields --indicator AI008-1-5
  regionalatlasctl sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 5

Commands
  doctor
  indicators list
  indicators search
  indicator get
  fields
  sample
  source
  dossier
  query-builder
  explain-field
  query

Safety defaults
  resultRecordCount defaults small, geometry is off, and limits above 100
  require --allow-large-output.
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "sample":
        print("""regionalatlasctl sample

Fetch a small bounded sample from a Regionalatlas dynamic-layer query.

Example
  regionalatlasctl sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 5
""")
    elif joined == "dossier":
        print("""regionalatlasctl dossier

Build metadata, fields, source URLs, query URL, warnings, and a tiny sample.
""")
    elif joined == "query-builder":
        print("""regionalatlasctl query-builder

Build the encoded ArcGIS dynamic-layer query URL without fetching data.
""")
    elif joined == "query":
        print("""regionalatlasctl query

Run a raw ArcGIS dynamic-layer query. Prefer sample/query-builder when possible.

Examples
  regionalatlasctl query --layer-file layer.json --param outFields=ags,gen,ai0801
  regionalatlasctl query --layer <json> --param resultRecordCount=5

On Windows shells, prefer --layer-file because raw JSON quoting is fragile.
""")
    else:
        print_root_help()


def run_doctor(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 1, 10)
    payload = envelope("doctor", MAP_SERVER_URL + "?f=json", None)
    warnings = default_warnings()
    summary = {
        "authRequired": False,
        "catalogUrl": CATALOG_URL,
        "mapServerUrl": MAP_SERVER_URL,
        "publishedRateLimit": "No exact public API rate limit found in reviewed Regionalatlas/API materials. Use small limits, cache catalog metadata, and avoid parallel broad ArcGIS queries.",
        "fairUseHints": [
            "Use indicators search/list and fields before sample/query.",
            "Do not request geometry unless map shapes are required.",
            "Avoid municipality-level full pulls unless explicitly exporting with a plan.",
            "Back off on 429, 5xx, or slow responses.",
        ],
    }
    try:
        summary["mapServerReachable"] = True
        summary["mapServer"] = map_server_summary(fetch_json(MAP_SERVER_URL + "?f=json"))
    except Exception as exc:
        summary["mapServerReachable"] = False
        warnings.append(f"mapServer: {exc}")
    try:
        catalog = fetch_catalog()
        flat = flatten_catalog(catalog)
        summary["catalogReachable"] = True
        summary["topics"] = len(catalog)
        summary["indicators"] = len(flat)
        summary["sampleIndicators"] = compact_indicators(flat, limit)
    except Exception as exc:
        summary["catalogReachable"] = False
        warnings.append(f"catalog: {exc}")
    payload["status"] = "ok" if summary.get("mapServerReachable") and summary.get("catalogReachable") else "degraded"
    payload["summary"] = summary
    payload["sources"] = default_sources()
    payload["warnings"] = warnings
    payload["nextActions"] = ['regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5', "regionalatlasctl fields --indicator AI008-1-5"]
    emit(payload)


def run_indicators_list(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, DEFAULT_LIMIT, 50)
    topic = first_non_empty(parsed["flags"].get("topic"), parsed["flags"].get("thema")).lower()
    flat = flatten_catalog(fetch_catalog())
    if topic:
        flat = [item for item in flat if topic in item["topic"].lower()]
    payload = envelope("indicators list", CATALOG_URL, {"limit": limit, "topic": topic})
    payload["summary"] = {"returned": min(limit, len(flat)), "available": len(flat), "topicFilter": topic}
    payload["items"] = compact_indicators(flat, limit)
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5']
    emit(payload)


def run_indicators_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "indicators search requires --term")
    limit = limit_flag(parsed, 5, 50)
    matches = search_catalog(flatten_catalog(fetch_catalog()), term)
    payload = envelope("indicators search", CATALOG_URL, {"term": term, "limit": limit})
    payload["summary"] = {"term": term, "matches": len(matches), "returned": min(limit, len(matches))}
    payload["items"] = compact_indicators(matches, limit)
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_for_indicators(matches)
    emit(payload)


def run_indicator_get(argv):
    parsed = parse_args(argv)
    item = find_indicator(required_indicator(parsed))
    field = first_attribute_code(item["node"])
    year = latest_year(item["node"])
    payload = envelope("indicator get", CATALOG_URL, {"indicator": item["node"].get("code")})
    payload["summary"] = indicator_summary(item)
    payload["items"] = compact_attributes(item["node"].get("attributes") or [], 50)
    payload["sources"] = sources_for_indicator(item["node"], field, year)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"regionalatlasctl fields --indicator {item['node'].get('code')}", f"regionalatlasctl sample --indicator {item['node'].get('code')} --field {field} --year {year} --region-level 1 --limit 5"]
    emit(payload)


def run_fields(argv):
    parsed = parse_args(argv)
    item = find_indicator(required_indicator(parsed))
    node = item["node"]
    field = first_attribute_code(node)
    payload = envelope("fields", CATALOG_URL, {"indicator": node.get("code")})
    payload["summary"] = {
        "indicator": node.get("code"),
        "title": node.get("title_short"),
        "topic": item["topic"],
        "availableYears": available_years(node),
        "latestYear": latest_year(node),
        "regionLevels": region_level_availability(node),
        "attributeCount": len(node.get("attributes") or []),
        "regionalDbTable": node.get("code"),
    }
    payload["items"] = compact_attributes(node.get("attributes") or [], 100)
    payload["sources"] = sources_for_indicator(node, field, latest_year(node))
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"regionalatlasctl explain-field --indicator {node.get('code')} --field {field}", f"regionalatlasctl sample --indicator {node.get('code')} --field {field} --year {latest_year(node)} --region-level 1 --limit 5"]
    emit(payload)


def run_sample(argv):
    parsed = parse_args(argv)
    item, field, year, region_level = resolve_query_inputs(parsed)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    params = build_query_params(item["node"], field, year, region_level, limit, parsed)
    request_url = QUERY_ENDPOINT + "?" + urllib.parse.urlencode(params)
    data = fetch_json(request_url)
    warnings = default_warnings()
    if data.get("exceededTransferLimit"):
        warnings.append("ArcGIS reported exceededTransferLimit=true; the returned sample is not a complete extract.")
    if flag_bool(parsed, "geometry"):
        warnings.append("Geometry was requested intentionally; municipality-level geometry can be very large.")
    items = compact_features(data, flag_bool(parsed, "geometry"))
    node = item["node"]
    payload = envelope("sample", request_url, {"indicator": node.get("code"), "field": field.upper(), "year": year, "regionLevel": region_level, "limit": limit})
    payload["summary"] = {"indicator": node.get("code"), "field": field.upper(), "fieldTitle": attribute_title(node, field), "unit": attribute_unit(node, field), "year": year, "regionLevel": region_level, "regionLevelLabel": region_level_label(region_level), "returned": len(items), "limitApplied": limit, "returnGeometry": flag_bool(parsed, "geometry"), "exceededTransferLimit": bool(data.get("exceededTransferLimit"))}
    payload["items"] = items
    payload["sources"] = sources_for_indicator(node, field, year)
    payload["warnings"] = warnings
    payload["nextActions"] = [f"regionalatlasctl query-builder --indicator {node.get('code')} --field {field.upper()} --year {year} --region-level {region_level} --limit {limit}", f"regionalatlasctl explain-field --indicator {node.get('code')} --field {field.upper()}"]
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_source(argv):
    parsed = parse_args(argv)
    item = find_indicator(required_indicator(parsed))
    node = item["node"]
    field = first_non_empty(parsed["flags"].get("field"), first_attribute_code(node))
    year = int_flag(parsed, "year", latest_year(node))
    payload = envelope("source", CATALOG_URL, {"indicator": node.get("code"), "field": field, "year": year})
    payload["summary"] = {"indicator": node.get("code"), "title": node.get("title_short"), "field": field.upper(), "year": year, "regionalDbTable": node.get("code"), "authRequired": False}
    payload["sources"] = sources_for_indicator(node, field, year)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"regionalatlasctl dossier --indicator {node.get('code')} --field {field.upper()} --year {year}"]
    emit(payload)


def run_dossier(argv):
    parsed = parse_args(argv)
    item, field, year, region_level = resolve_query_inputs(parsed)
    limit = limit_flag(parsed, 5, 25)
    params = build_query_params(item["node"], field, year, region_level, limit, parsed)
    request_url = QUERY_ENDPOINT + "?" + urllib.parse.urlencode(params)
    warnings = default_warnings()
    sample_data = None
    try:
        sample_data = fetch_json(request_url)
    except Exception as exc:
        warnings.append(f"sampleQuery: {exc}")
    node = item["node"]
    payload = envelope("dossier", request_url, {"indicator": node.get("code"), "field": field.upper(), "year": year, "regionLevel": region_level, "limit": limit})
    payload["summary"] = {"indicator": node.get("code"), "title": node.get("title_short"), "topic": item["topic"], "field": field.upper(), "fieldTitle": attribute_title(node, field), "unit": attribute_unit(node, field), "year": year, "regionLevel": region_level, "regionLevelLabel": region_level_label(region_level), "availableYears": available_years(node), "regionLevels": region_level_availability(node)}
    payload["fields"] = compact_attributes(node.get("attributes") or [], 100)
    payload["metadata"] = {"indicatorTitleLong": node.get("title_long"), "fieldMetaSnippets": meta_snippets(attribute_meta(node, field), "", 6)}
    if sample_data:
        payload["sample"] = {"items": compact_features(sample_data, flag_bool(parsed, "geometry")), "exceededTransferLimit": bool(sample_data.get("exceededTransferLimit"))}
        if sample_data.get("exceededTransferLimit"):
            warnings.append("Sample query reports exceededTransferLimit=true; use pagination/filtering for complete extraction.")
    payload["sources"] = sources_for_indicator(node, field, year)
    payload["warnings"] = warnings
    payload["nextActions"] = [f"regionalatlasctl explain-field --indicator {node.get('code')} --field {field.upper()} --grep Quelle", f"regionalatlasctl query-builder --indicator {node.get('code')} --field {field.upper()} --year {year} --region-level {region_level}"]
    emit(payload)


def run_query_builder(argv):
    parsed = parse_args(argv)
    item, field, year, region_level = resolve_query_inputs(parsed)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    params = build_query_params(item["node"], field, year, region_level, limit, parsed)
    request_url = QUERY_ENDPOINT + "?" + urllib.parse.urlencode(params)
    payload = envelope("query-builder", request_url, {"indicator": item["node"].get("code"), "field": field.upper(), "year": year, "regionLevel": region_level, "limit": limit})
    payload["summary"] = {"indicator": item["node"].get("code"), "field": field.upper(), "year": year, "regionLevel": region_level, "regionLevelLabel": region_level_label(region_level), "requestUrl": request_url, "layerJson": params["layer"], "doesNotFetch": True}
    payload["sources"] = sources_for_indicator(item["node"], field, year)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"regionalatlasctl sample --indicator {item['node'].get('code')} --field {field.upper()} --year {year} --region-level {region_level} --limit {limit}"]
    emit(payload)


def run_explain_field(argv):
    parsed = parse_args(argv)
    item = find_indicator(required_indicator(parsed))
    node = item["node"]
    field = first_non_empty(parsed["flags"].get("field"), parsed["flags"].get("name"), first_position(parsed), first_attribute_code(node))
    attr = find_attribute(node, field)
    if not attr:
        raise CLIError(2, "field_not_found", "field not found in indicator attributes")
    grep = first_non_empty(parsed["flags"].get("grep"))
    payload = envelope("explain-field", CATALOG_URL, {"indicator": node.get("code"), "field": field.upper(), "grep": grep})
    payload["summary"] = {"indicator": node.get("code"), "field": attr.get("code", "").upper(), "title": attr.get("title_short"), "titleLong": attr.get("title_long"), "unit": attr.get("unit")}
    payload["items"] = meta_snippets(attr.get("meta", ""), grep, 10)
    payload["sources"] = sources_for_indicator(node, attr.get("code", ""), latest_year(node))
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"regionalatlasctl sample --indicator {node.get('code')} --field {attr.get('code', '').upper()} --year {latest_year(node)} --region-level 1 --limit 5"]
    emit(payload)


def run_raw_query(argv):
    parsed = parse_args(argv)
    params = dict(parsed["params"])
    if not params.get("layer") and not parsed["flags"].get("layer") and not parsed["flags"].get("layer-file"):
        raise CLIError(2, "missing_layer", "raw query requires --param layer=<json>, --layer-file <path>, or use query-builder/sample")
    if parsed["flags"].get("layer-file"):
        try:
            with open(parsed["flags"]["layer-file"], "r", encoding="utf-8") as handle:
                params["layer"] = handle.read().lstrip("\ufeff").strip()
        except OSError as exc:
            raise CLIError(2, "layer_file_read_failed", str(exc))
    if parsed["flags"].get("layer"):
        params["layer"] = parsed["flags"]["layer"]
    params.setdefault("f", "json")
    params.setdefault("returnGeometry", "false")
    params.setdefault("where", "1=1")
    params.setdefault("spatialRel", "esriSpatialRelIntersects")
    if not params.get("resultRecordCount") and not params.get("resultrecordcount"):
        params["resultRecordCount"] = str(limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT))
    if not flag_bool(parsed, "allow-large-output") and int_value(first_non_empty(params.get("resultRecordCount"), params.get("resultrecordcount"))) > SAFE_LIMIT:
        raise CLIError(2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override")
    emit(fetch_json(QUERY_ENDPOINT + "?" + urllib.parse.urlencode(params)))


def parse_args(args):
    parsed = {"flags": {}, "params": {}, "positionals": []}
    i = 0
    while i < len(args):
        arg = args[i]
        if not arg.startswith("--"):
            parsed["positionals"].append(arg)
            i += 1
            continue
        key_value = arg[2:]
        if "=" in key_value:
            key, value = key_value.split("=", 1)
        elif i + 1 < len(args) and not args[i + 1].startswith("--"):
            key, value = key_value, args[i + 1]
            i += 1
        else:
            key, value = key_value, "true"
        key = key.lower().strip()
        if key == "param" and "=" in value:
            pkey, pvalue = value.split("=", 1)
            parsed["params"][pkey] = pvalue
        else:
            parsed["flags"][key] = value
        i += 1
    return parsed


def fetch_catalog():
    return fetch_json(CATALOG_URL)


def fetch_json(request_url):
    req = urllib.request.Request(request_url, headers={"User-Agent": "democracy-researcher/regionalatlasctl-python-2.0"})
    try:
        with urllib.request.urlopen(req, timeout=45) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"HTTP {exc.code} from {request_url}: {body[:300]}")
    data = json.loads(body)
    if isinstance(data, dict) and data.get("error"):
        raise RuntimeError(f"upstream error {data['error']}")
    return data


def required_indicator(parsed):
    code = first_non_empty(parsed["flags"].get("indicator"), parsed["flags"].get("code"), parsed["flags"].get("table"), first_position(parsed))
    if not code:
        raise CLIError(2, "missing_indicator", "command requires --indicator")
    return code.upper()


def find_indicator(code):
    norm = normalize_code(code)
    for item in flatten_catalog(fetch_catalog()):
        if normalize_code(item["node"].get("code", "")) == norm:
            return item
    raise CLIError(2, "indicator_not_found", "indicator not found in Regionalatlas catalog")


def resolve_query_inputs(parsed):
    item = find_indicator(required_indicator(parsed))
    node = item["node"]
    field = first_non_empty(parsed["flags"].get("field"), parsed["flags"].get("icode"), first_attribute_code(node)).lower()
    if not find_attribute(node, field):
        raise CLIError(2, "field_not_found", "field not found for indicator")
    year = int_flag(parsed, "year", latest_year(node))
    if not year:
        raise CLIError(2, "missing_year", "year could not be inferred; pass --year")
    region_level = int_flag(parsed, "region-level", int_flag(parsed, "typ", 1))
    if region_level not in {1, 2, 3, 5}:
        raise CLIError(2, "invalid_region_level", "region-level must be one of 1, 2, 3, or 5")
    return item, field, year, region_level


def build_query_params(node, field, year, region_level, limit, parsed):
    table = table_name(node.get("code", ""))
    geo_year = int_flag(parsed, "geo-year", year)
    sql = f"SELECT * FROM verwaltungsgrenzen_gesamt LEFT OUTER JOIN {table} ON ags = ags2 and jahr = jahr2 WHERE typ = {region_level} AND jahr = {geo_year} AND (jahr2 = {year} OR jahr2 IS NULL)"
    layer = {"source": {"dataSource": {"geometryType": "esriGeometryPolygon", "workspaceId": "gdb", "query": sql, "oidFields": "id", "spatialReference": {"wkid": 25832}, "type": "queryTable"}, "type": "dataLayer"}}
    out_fields = first_non_empty(parsed["flags"].get("fields"), parsed["params"].get("outFields"), f"ags,gen,typ,jahr,jahr2,ags2,gen2,{field.lower()}")
    where = first_non_empty(parsed["flags"].get("where"), parsed["params"].get("where"), "1=1")
    if parsed["flags"].get("ags") and where == "1=1":
        where = "ags = '%s'" % parsed["flags"]["ags"].replace("'", "''")
    params = {"layer": json.dumps(layer, separators=(",", ":")), "f": "json", "outFields": out_fields, "returnGeometry": bool_string(flag_bool(parsed, "geometry")), "spatialRel": "esriSpatialRelIntersects", "where": where, "resultRecordCount": str(limit)}
    for key, value in parsed["params"].items():
        if key != "layer":
            params[key] = value
    return params


def flatten_catalog(catalog):
    flat = []

    def walk(nodes, topic=""):
        for node in nodes:
            next_topic = topic
            if node.get("title") and not node.get("code"):
                next_topic = node["title"]
            if node.get("code") and node.get("attributes"):
                flat.append({"topic": next_topic, "node": node})
            if node.get("children"):
                walk(node["children"], next_topic)

    walk(catalog)
    return sorted(flat, key=lambda item: item["node"].get("code", ""))


def search_catalog(flat, term):
    needle = term.lower()
    matches = []
    for item in flat:
        node = item["node"]
        hay = " ".join([item["topic"], node.get("code", ""), node.get("title_short", ""), node.get("title_long", "")]).lower()
        for attr in node.get("attributes") or []:
            hay += " " + " ".join([attr.get("code", ""), attr.get("title_short", ""), attr.get("title_long", ""), attr.get("unit", ""), strip_wiki(attr.get("meta", ""))]).lower()
        if needle in hay:
            matches.append(item)
    return matches


def compact_indicators(items, limit):
    output = []
    for item in items[:limit]:
        node = item["node"]
        field = first_attribute_code(node)
        year = latest_year(node)
        output.append({"code": node.get("code"), "table": table_name(node.get("code", "")), "topic": item["topic"], "title": node.get("title_short"), "titleLong": node.get("title_long"), "latestYear": year, "availableYears": available_years(node), "attributes": compact_attributes(node.get("attributes") or [], 8), "nextActions": [f"regionalatlasctl fields --indicator {node.get('code')}", f"regionalatlasctl sample --indicator {node.get('code')} --field {field} --year {year} --region-level 1 --limit 5"]})
    return output


def compact_attributes(attrs, limit):
    return [{"code": attr.get("code", "").upper(), "field": attr.get("code", "").lower(), "title": attr.get("title_short"), "titleLong": attr.get("title_long"), "unit": attr.get("unit"), "metaPreview": truncate(strip_wiki(attr.get("meta", "")), 500)} for attr in attrs[:limit]]


def compact_features(data, include_geometry):
    items = []
    for feature in data.get("features") or []:
        item = {"attributes": normalize_attributes(feature.get("attributes") or {})}
        if include_geometry:
            item["geometry"] = feature.get("geometry")
        items.append(item)
    return items


def normalize_attributes(attrs):
    clean = {}
    for key, value in attrs.items():
        if key.lower().endswith(".shape"):
            continue
        clean[key] = value.strip() if isinstance(value, str) else value
    return clean


def indicator_summary(item):
    node = item["node"]
    return {"code": node.get("code"), "table": table_name(node.get("code", "")), "topic": item["topic"], "title": node.get("title_short"), "titleLong": node.get("title_long"), "timestamp": node.get("timestamp"), "latestYear": latest_year(node), "availableYears": available_years(node), "regionLevels": region_level_availability(node), "attributeCount": len(node.get("attributes") or [])}


def available_years(node):
    years = []
    for key in (node.get("years") or {}).keys():
        try:
            years.append(int(key))
        except ValueError:
            pass
    return sorted(years)


def latest_year(node):
    years = available_years(node)
    return years[-1] if years else 0


def region_level_availability(node):
    return {str(level): {"label": region_level_label(level), "appearsAvailableLatestYear": True} for level in [1, 2, 3, 5]}


def find_attribute(node, code):
    code = code.lower()
    for attr in node.get("attributes") or []:
        if attr.get("code", "").lower() == code:
            return attr
    return None


def first_attribute_code(node):
    attrs = node.get("attributes") or []
    for attr in attrs:
        code = attr.get("code", "")
        if code and not code.lower().endswith("v"):
            return code.upper()
    return attrs[0].get("code", "").upper() if attrs else ""


def attribute_title(node, code):
    attr = find_attribute(node, code)
    return attr.get("title_short", "") if attr else ""


def attribute_unit(node, code):
    attr = find_attribute(node, code)
    return attr.get("unit", "") if attr else ""


def attribute_meta(node, code):
    attr = find_attribute(node, code)
    return attr.get("meta", "") if attr else ""


def meta_snippets(meta, grep, limit):
    clean = strip_wiki(meta)
    lines = [line.strip() for line in clean.splitlines() if len(line.strip()) > 10]
    needle = grep.lower()
    snippets = []
    for line in lines:
        if not needle or needle in line.lower():
            snippets.append({"text": truncate(line, 700)})
            if len(snippets) >= limit:
                break
    return snippets


def strip_wiki(value):
    value = value.removeprefix("wiki")
    value = value.replace("===", "").replace("==", "").replace("'''", "").replace("''", "").replace("*", "")
    value = re.sub(r"\s+\|", " |", value)
    return value.strip()


def sources_for_indicator(node, field, year):
    app_deep_link = APP_URL + "?" + urllib.parse.urlencode({"BL": "DE", "TCode": node.get("code"), "ICode": field.upper(), "Jhr": str(year)})
    return [
        {"title": "Regionalatlas app", "url": app_deep_link, "kind": "interactive_atlas"},
        {"title": "Regionalatlas Statistikportal page", "url": STATISTIKPORTAL_URL, "kind": "official_context"},
        {"title": "Destatis Regionalatlas page", "url": DESTATIS_URL, "kind": "official_context"},
        {"title": "Statistikportal Open Data", "url": OPEN_DATA_URL, "kind": "terms_and_downloads"},
        {"title": "Destatis maps and geodata", "url": MAPS_GEODATA_URL, "kind": "terms_and_downloads"},
        {"title": "Regionalatlas catalog JSON", "url": CATALOG_URL, "kind": "catalog"},
        {"title": "Regionaldatenbank table", "url": "https://www.regionalstatistik.de/genesis/online/data?operation=table&code=" + node.get("code", ""), "kind": "official_table"},
        {"title": "ArcGIS dynamic-layer query endpoint", "url": QUERY_ENDPOINT, "kind": "api_endpoint"},
        {"title": "bundesAPI Regionalatlas OpenAPI wrapper", "url": OPENAPI_REPO_URL, "kind": "openapi_reference"},
    ]


def default_sources():
    return [
        {"title": "Regionalatlas Statistikportal page", "url": STATISTIKPORTAL_URL, "kind": "official_context"},
        {"title": "Destatis Regionalatlas page", "url": DESTATIS_URL, "kind": "official_context"},
        {"title": "Statistikportal Open Data", "url": OPEN_DATA_URL, "kind": "terms_and_downloads"},
        {"title": "Regionalatlas catalog JSON", "url": CATALOG_URL, "kind": "catalog"},
        {"title": "Regionalatlas thesaurus CSV", "url": THESAURUS_URL, "kind": "catalog"},
        {"title": "ArcGIS MapServer metadata", "url": MAP_SERVER_URL + "?f=json", "kind": "api_metadata"},
        {"title": "bundesAPI Regionalatlas OpenAPI wrapper", "url": OPENAPI_REPO_URL, "kind": "openapi_reference"},
    ]


def default_warnings():
    return [
        "No exact published API rate limit was found in reviewed materials; keep requests small and cache catalog metadata.",
        "The ArcGIS service advertises a very high maxRecordCount; never run broad municipality-level pulls accidentally.",
        "Use field metadata for units, definitions, source statistics, and regional caveats before interpreting values.",
        "Statistikportal Open Data notes point to Datenlizenz Deutschland 2.0 for statistical data and atlas/imprint license hints for geodata.",
    ]


def next_actions_for_indicators(items):
    actions = []
    for item in items[:3]:
        node = item["node"]
        actions.append(f"regionalatlasctl dossier --indicator {node.get('code')} --field {first_attribute_code(node)} --year {latest_year(node)} --region-level 1")
    return actions or ['regionalatlasctl indicators search --term "Bevoelkerung" --limit 5']


def map_server_summary(data):
    return {"mapName": data.get("mapName"), "supportsDynamicLayers": bool(data.get("supportsDynamicLayers")), "supportedQueryFormats": data.get("supportedQueryFormats"), "maxRecordCount": int_value(data.get("maxRecordCount")), "capabilities": data.get("capabilities"), "featureLayerCount": len(data.get("layers") or []), "spatialReferenceLatest": int_value((data.get("spatialReference") or {}).get("latestWkid"))}


def envelope(command, request_url, request):
    return {"status": "ok", "tool": APP_NAME, "command": command, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "request": {"method": "GET", "url": request_url, "params": request}, "summary": {}, "items": [], "sources": [], "warnings": [], "nextActions": []}


def emit(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})
    sys.exit(exit_code)


def is_help(value):
    return value in {"--help", "-h", "help"}


def first_non_empty(*values):
    for value in values:
        if value is not None and str(value).strip():
            return str(value).strip()
    return ""


def first_position(parsed):
    return parsed["positionals"][0] if parsed["positionals"] else ""


def flag_bool(parsed, key):
    return str(parsed["flags"].get(key, "")).lower() in {"true", "1", "yes", "y"}


def bool_string(value):
    return "true" if value else "false"


def limit_flag(parsed, fallback, max_value):
    raw = first_non_empty(parsed["flags"].get("limit"), parsed["flags"].get("resultrecordcount"), parsed["params"].get("resultRecordCount"), parsed["params"].get("resultrecordcount"))
    value = int_value(raw) if raw else fallback
    if value < 1:
        value = fallback
    if value > max_value and not flag_bool(parsed, "allow-large-output"):
        raise CLIError(2, "limit_exceeds_safe_max", f"limit {value} exceeds safe max {max_value}; pass --allow-large-output to override")
    return value


def int_flag(parsed, key, fallback):
    return int_value(parsed["flags"].get(key)) or fallback


def int_value(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def table_name(code):
    return code.replace("-", "_").lower()


def normalize_code(code):
    return code.strip().replace("_", "-").upper()


def region_level_label(level):
    return {1: "Laender", 2: "Regierungsbezirke/statistical regions", 3: "Kreise and kreisfreie Staedte", 5: "Gemeinden/Gemeindeverbaende"}.get(level, "unknown")


def truncate(value, max_len):
    return value if len(value) <= max_len else value[:max_len] + "..."


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
