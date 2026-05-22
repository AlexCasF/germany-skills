#!/usr/bin/env python3
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "deutschlandatlas"
PORTAL_SEARCH_BASE = "https://www.karto365.de/portal/sharing/rest/search"
HOSTING_BASE = "https://www.karto365.de/hosting/rest/services"
OFFICIAL_HOME_URL = "https://www.deutschlandatlas.bund.de/DE/Home/home_node.html"
OFFICIAL_DOWNLOADS_URL = "https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html"
GITHUB_SPEC_URL = "https://github.com/bundesAPI/deutschlandatlas-api"
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
        elif argv[:2] == ["tables", "search"]:
            run_tables_search(argv[2:])
        elif argv[:2] == ["table", "query"]:
            run_table_query(argv[2:])
        elif argv[:2] == ["table", "fields"]:
            run_table_fields(argv[2:])
        elif argv[:2] == ["table", "sample"]:
            run_table_sample(argv[2:])
        elif argv[:2] == ["table", "source"]:
            run_table_source(argv[2:])
        elif argv[:2] == ["indicator", "dossier"]:
            run_indicator_dossier(argv[2:])
        elif argv[0] == "query-builder":
            run_query_builder(argv[1:])
        elif argv[0] == "explain-field":
            run_explain_field(argv[1:])
        else:
            raise CLIError(2, "unknown_command", "unknown command; run deutschlandatlas --help")
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""deutschlandatlas -- Deutschlandatlas ArcGIS research CLI

Purpose
  Discover and query public Deutschlandatlas indicator map services for
  regional living-condition indicators in Germany.

Fast paths
  deutschlandatlas doctor
  deutschlandatlas tables search --term "Arbeitslosenquote" --limit 5
  deutschlandatlas table fields --table alq_HA2023
  deutschlandatlas table sample --table alq_HA2023 --limit 5
  deutschlandatlas indicator dossier --table alq_HA2023

Legacy-compatible command
  table query --table <table> [--param key=value] [--layer auto|0|5]

Research commands
  doctor
  tables search
  table fields
  table sample
  table source
  indicator dossier
  query-builder
  explain-field
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "table sample":
        print("""deutschlandatlas table sample

Fetch a small bounded sample from one Deutschlandatlas ArcGIS table.

Examples
  deutschlandatlas table sample --table alq_HA2023 --limit 5
  deutschlandatlas table sample --table alq_HA2023 --fields name,alq --where "alq > 10"
""")
    elif joined == "indicator dossier":
        print("""deutschlandatlas indicator dossier

Bundle metadata, selected layer, fields, source URLs, warnings, and a tiny sample.
""")
    elif joined == "tables search":
        print("""deutschlandatlas tables search

Search the public ArcGIS portal for Deutschlandatlas table services.
""")
    else:
        print_root_help()


def run_doctor(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 1, 10)
    search_url = portal_search_url("", limit, 1)
    payload = envelope("doctor", search_url, None)
    summary = {
        "authRequired": False,
        "publishedRateLimit": "No exact public rate limit found in reviewed Deutschlandatlas/API materials. Use small limits, cache metadata, and avoid parallel broad ArcGIS queries.",
        "fairUseHints": [
            "Prefer tables search, fields, and small samples before broad queries.",
            "Do not request geometry unless map geometry is needed.",
            "Respect ArcGIS transfer limits and back off on 429, 5xx, or slow responses.",
        ],
    }
    warnings = default_warnings()
    try:
        search_data = fetch_json(search_url)
        summary["portalSearchReachable"] = True
        summary["portalTotal"] = int_value(search_data.get("total"))
    except Exception as exc:
        summary["portalSearchReachable"] = False
        warnings.append(f"portalSearch: {exc}")
    try:
        service_data = fetch_json(service_url("alq_HA2023"))
        summary["sampleServiceReachable"] = True
        summary["sampleService"] = service_summary(service_data)
    except Exception as exc:
        summary["sampleServiceReachable"] = False
        warnings.append(f"sampleService: {exc}")
    payload["status"] = "ok" if summary.get("portalSearchReachable") and summary.get("sampleServiceReachable") else "degraded"
    payload["summary"] = summary
    payload["sources"] = default_sources()
    payload["warnings"] = warnings
    payload["nextActions"] = ['deutschlandatlas tables search --term "Arbeitslosenquote" --limit 5', "deutschlandatlas indicator dossier --table alq_HA2023"]
    emit(payload)


def run_tables_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "tables search requires --term")
    limit = limit_flag(parsed, 5, 25)
    start = int_flag(parsed, "start", 1)
    request_url = portal_search_url(term, limit, start)
    data = fetch_json(request_url)
    items = compact_portal_results(data, limit)
    payload = envelope("tables search", request_url, {"term": term, "limit": limit, "start": start})
    payload["summary"] = {"term": term, "total": int_value(data.get("total")), "returned": len(items), "limitApplied": limit, "nextStart": int_value(data.get("nextStart"))}
    payload["items"] = items
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_for_tables(items)
    emit(payload)


def run_table_query(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    layer, _ = resolve_layer(table, parsed)
    params = {
        "f": first_non_empty(parsed["params"].get("f"), "json"),
        "where": first_non_empty(parsed["params"].get("where"), parsed["flags"].get("where"), "1=1"),
        "outFields": first_non_empty(parsed["params"].get("outFields"), parsed["params"].get("outfields"), parsed["flags"].get("fields"), "*"),
        "returnGeometry": first_non_empty(parsed["params"].get("returnGeometry"), parsed["params"].get("returngeometry"), bool_string(flag_bool(parsed, "geometry"))),
    }
    params.update(parsed["params"])
    if not params.get("resultRecordCount") and not params.get("resultrecordcount"):
        params["resultRecordCount"] = str(limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT))
    if not flag_bool(parsed, "allow-large-output"):
        count = int_value(first_non_empty(params.get("resultRecordCount"), params.get("resultrecordcount")))
        if count > SAFE_LIMIT:
            raise CLIError(2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override")
    emit(fetch_json(query_url(table, layer, params)))


def run_table_fields(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    layer, layer_source = resolve_layer(table, parsed)
    request_url = layer_url(table, layer)
    layer_data = fetch_json(request_url)
    fields = compact_fields(layer_data)
    payload = envelope("table fields", request_url, {"table": table, "layer": layer})
    payload["summary"] = {
        "table": table,
        "layer": layer,
        "layerSource": layer_source,
        "fieldCount": len(fields),
        "displayField": layer_data.get("displayField"),
        "objectIdField": layer_data.get("objectIdField"),
        "geometryType": layer_data.get("geometryType"),
        "maxRecordCount": int_value(layer_data.get("maxRecordCount")),
        "likelyIndicatorFields": likely_indicator_fields(fields),
    }
    payload["items"] = fields
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"deutschlandatlas table sample --table {table} --fields name,{first_likely_indicator(fields)} --limit 5", f"deutschlandatlas indicator dossier --table {table}"]
    emit(payload)


def run_table_sample(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    layer, layer_source = resolve_layer(table, parsed)
    params = {
        "f": "json",
        "where": first_non_empty(parsed["params"].get("where"), parsed["flags"].get("where"), "1=1"),
        "outFields": first_non_empty(parsed["params"].get("outFields"), parsed["params"].get("outfields"), parsed["flags"].get("fields"), "*"),
        "returnGeometry": bool_string(flag_bool(parsed, "geometry")),
        "resultRecordCount": str(limit),
    }
    params.update(parsed["params"])
    request_url = query_url(table, layer, params)
    data = fetch_json(request_url)
    items = compact_features(data, flag_bool(parsed, "geometry"))
    warnings = default_warnings()
    if data.get("exceededTransferLimit"):
        warnings.append("ArcGIS reported exceededTransferLimit=true; narrow the where clause or paginate deliberately.")
    if flag_bool(parsed, "geometry"):
        warnings.append("Geometry was requested intentionally; outputs can grow quickly.")
    payload = envelope("table sample", request_url, {"table": table, "layer": layer, "limit": limit})
    payload["summary"] = {"table": table, "layer": layer, "layerSource": layer_source, "returned": len(items), "limitApplied": limit, "returnGeometry": flag_bool(parsed, "geometry"), "exceededTransferLimit": bool(data.get("exceededTransferLimit")), "displayField": data.get("displayFieldName")}
    payload["items"] = items
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = warnings
    payload["nextActions"] = [f"deutschlandatlas table fields --table {table}", f"deutschlandatlas query-builder --table {table} --where \"name LIKE '%Berlin%'\" --fields name,* --limit 10"]
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_table_source(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    layer, layer_source = resolve_layer(table, parsed) if not flag_bool(parsed, "skip-layer-discovery") else (0, "legacy_default")
    payload = envelope("table source", service_url(table), {"table": table, "layer": layer})
    payload["summary"] = {"table": table, "selectedLayer": layer, "layerSource": layer_source, "authRequired": False, "apiStyle": "ArcGIS REST MapServer query endpoint", "rateLimitFound": False}
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"deutschlandatlas table fields --table {table}", f"deutschlandatlas table sample --table {table} --limit 5"]
    emit(payload)


def run_indicator_dossier(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    limit = limit_flag(parsed, 5, 25)
    layer, layer_source = resolve_layer(table, parsed)
    payload = envelope("indicator dossier", service_url(table), {"table": table, "layer": layer, "limit": limit})
    payload["summary"] = {"table": table, "selectedLayer": layer, "layerSource": layer_source, "limitApplied": limit, "authRequired": False}
    warnings = default_warnings()
    try:
        payload["service"] = service_summary(fetch_json(service_url(table)))
    except Exception as exc:
        warnings.append(f"serviceMetadata: {exc}")
    fields = []
    try:
        fields = compact_fields(fetch_json(layer_url(table, layer)))
        payload["fields"] = fields
        payload["summary"]["likelyIndicatorFields"] = likely_indicator_fields(fields)
    except Exception as exc:
        warnings.append(f"layerMetadata: {exc}")
    try:
        params = {"f": "json", "where": "1=1", "outFields": "*", "returnGeometry": "false", "resultRecordCount": str(limit)}
        sample = fetch_json(query_url(table, layer, params))
        payload["sample"] = {"items": compact_features(sample, False), "exceededTransferLimit": bool(sample.get("exceededTransferLimit"))}
        if sample.get("exceededTransferLimit"):
            warnings.append("Sample query reports exceededTransferLimit=true; use pagination/filtering for full extraction.")
    except Exception as exc:
        warnings.append(f"sampleQuery: {exc}")
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = warnings
    payload["nextActions"] = [f"deutschlandatlas table fields --table {table}", f"deutschlandatlas table sample --table {table} --fields name,* --where \"1=1\" --limit 10", f"deutschlandatlas explain-field --table {table} --field {first_likely_indicator(fields)}"]
    emit(payload)


def run_query_builder(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    layer, layer_source = resolve_layer(table, parsed)
    where = first_non_empty(parsed["flags"].get("where"), "1=1")
    if parsed["flags"].get("region") and where == "1=1":
        region = parsed["flags"]["region"].replace("'", "''")
        where = f"name LIKE '%{region}%'"
    params = {"f": "json", "where": where, "outFields": first_non_empty(parsed["flags"].get("fields"), "*"), "returnGeometry": bool_string(flag_bool(parsed, "geometry")), "resultRecordCount": str(limit)}
    params.update(parsed["params"])
    built_url = query_url(table, layer, params)
    payload = envelope("query-builder", built_url, {"table": table, "layer": layer, "params": params})
    payload["summary"] = {"table": table, "layer": layer, "layerSource": layer_source, "requestUrl": built_url, "doesNotFetch": True, "limitApplied": limit, "returnGeometry": flag_bool(parsed, "geometry")}
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = default_warnings()
    if parsed["flags"].get("year"):
        payload["warnings"].append("Generic Deutschlandatlas services do not expose one standard year parameter; choose a year-specific table.")
    payload["nextActions"] = [f"deutschlandatlas table query --table {table} --layer {layer} --param where={where!r} --param outFields={params['outFields']!r} --limit {limit}"]
    emit(payload)


def run_explain_field(argv):
    parsed = parse_args(argv)
    table = required_table(parsed)
    field_name = first_non_empty(parsed["flags"].get("field"), parsed["flags"].get("name"), first_position(parsed))
    if not field_name:
        raise CLIError(2, "missing_field", "explain-field requires --field")
    layer, _ = resolve_layer(table, parsed)
    fields = compact_fields(fetch_json(layer_url(table, layer)))
    match = next((field for field in fields if field.get("name", "").lower() == field_name.lower()), None)
    if not match:
        raise CLIError(2, "field_not_found", "field not found in layer metadata")
    payload = envelope("explain-field", layer_url(table, layer), {"table": table, "field": field_name, "layer": layer})
    payload["summary"] = {"table": table, "layer": layer, "field": match, "interpretationHint": "Use the alias, table title/snippet from tables search, and official downloads/method notes for statistical meaning and units."}
    payload["sources"] = sources_for_table(table, layer)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"deutschlandatlas table sample --table {table} --fields name,{field_name} --limit 10"]
    emit(payload)


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
        if key == "param":
            if "=" in value:
                pkey, pvalue = value.split("=", 1)
                parsed["params"][pkey] = pvalue
        else:
            parsed["flags"][key] = value
        i += 1
    return parsed


def required_table(parsed):
    table = first_non_empty(parsed["flags"].get("table"), parsed["flags"].get("name"), first_position(parsed))
    if not table:
        raise CLIError(2, "missing_table", "command requires --table")
    return table


def resolve_layer(table, parsed):
    if flag_bool(parsed, "legacy-layer-zero"):
        return 0, "legacy_layer_zero"
    layer_flag = first_non_empty(parsed["flags"].get("layer"), "auto")
    if layer_flag and layer_flag != "auto":
        try:
            return int(layer_flag), "explicit_flag"
        except ValueError:
            raise CLIError(2, "invalid_layer", "--layer must be auto or an integer")
    data = fetch_json(service_url(table))
    for layer in data.get("layers") or []:
        if "feature" in str(layer.get("type", "")).lower():
            return int_value(layer.get("id")), "service_metadata"
    layers = data.get("layers") or []
    if layers:
        return int_value(layers[0].get("id")), "service_metadata"
    raise CLIError(1, "no_feature_layer", "service metadata did not expose a feature layer")


def fetch_json(request_url):
    req = urllib.request.Request(request_url, headers={"User-Agent": "germany-skills/deutschlandatlas-python-2.0"})
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"HTTP {exc.code} from {request_url}: {body[:300]}")
    data = json.loads(body)
    if isinstance(data, dict) and data.get("error"):
        raise RuntimeError(f"upstream error {data['error']}")
    return data


def portal_search_url(term, limit, start):
    query = "deutschlandatlas" + (f" {term.strip()}" if str(term).strip() else "")
    return PORTAL_SEARCH_BASE + "?" + urllib.parse.urlencode({"q": query, "f": "json", "num": str(limit), "start": str(start)})


def service_url(table):
    return f"{HOSTING_BASE}/{urllib.parse.quote(table)}/MapServer?f=json"


def layer_url(table, layer):
    return f"{HOSTING_BASE}/{urllib.parse.quote(table)}/MapServer/{layer}?f=json"


def query_url(table, layer, params):
    return f"{HOSTING_BASE}/{urllib.parse.quote(table)}/MapServer/{layer}/query?" + urllib.parse.urlencode(params)


def compact_portal_results(data, limit):
    items = []
    for item in (data.get("results") or [])[:limit]:
        service = item.get("url") or ""
        table = first_non_empty(item.get("title"), table_from_url(service))
        items.append({
            "table": table,
            "title": item.get("title"),
            "snippet": item.get("snippet"),
            "type": item.get("type"),
            "serviceUrl": service,
            "access": item.get("access"),
            "tags": item.get("tags"),
            "modifiedUtc": millis_to_utc(item.get("modified")),
            "nextActions": [f"deutschlandatlas table fields --table {table}", f"deutschlandatlas indicator dossier --table {table}"],
        })
    return items


def compact_fields(layer_data):
    return [{"name": f.get("name"), "alias": f.get("alias"), "type": f.get("type"), "length": int_value(f.get("length")), "domain": f.get("domain")} for f in layer_data.get("fields") or []]


def compact_features(data, include_geometry):
    items = []
    for feature in data.get("features") or []:
        item = {"attributes": feature.get("attributes")}
        if include_geometry:
            item["geometry"] = feature.get("geometry")
        items.append(item)
    return items


def likely_indicator_fields(fields):
    skip = {"objectid", "shape", "gf", "gen", "bez", "gebietskennziffer", "name", "shape_length", "shape_area"}
    return [f.get("name") for f in fields if f.get("name") and f.get("name").lower() not in skip and not f.get("name").lower().startswith("shape")]


def first_likely_indicator(fields):
    likely = likely_indicator_fields(fields)
    return likely[0] if likely else "*"


def service_summary(data):
    return {
        "serviceDescription": data.get("serviceDescription"),
        "mapName": data.get("mapName"),
        "supportedQueryFormats": data.get("supportedQueryFormats"),
        "maxRecordCount": int_value(data.get("maxRecordCount")),
        "layers": [{"id": int_value(layer.get("id")), "name": layer.get("name"), "type": layer.get("type")} for layer in data.get("layers") or []],
    }


def sources_for_table(table, layer):
    return [
        {"title": "Deutschlandatlas start page", "url": OFFICIAL_HOME_URL, "kind": "official_context"},
        {"title": "Deutschlandatlas data downloads and method notes", "url": OFFICIAL_DOWNLOADS_URL, "kind": "official_downloads"},
        {"title": "bundesAPI Deutschlandatlas OpenAPI wrapper", "url": GITHUB_SPEC_URL, "kind": "openapi_reference"},
        {"title": "ArcGIS service metadata", "url": service_url(table), "kind": "api_service"},
        {"title": "ArcGIS layer metadata", "url": layer_url(table, layer), "kind": "api_layer"},
        {"title": "ArcGIS portal search", "url": portal_search_url(table, 10, 1), "kind": "api_discovery"},
    ]


def default_sources():
    return [
        {"title": "Deutschlandatlas start page", "url": OFFICIAL_HOME_URL, "kind": "official_context"},
        {"title": "Deutschlandatlas data downloads and method notes", "url": OFFICIAL_DOWNLOADS_URL, "kind": "official_downloads"},
        {"title": "bundesAPI Deutschlandatlas OpenAPI wrapper", "url": GITHUB_SPEC_URL, "kind": "openapi_reference"},
        {"title": "ArcGIS portal Deutschlandatlas search", "url": portal_search_url("", 100, 1), "kind": "api_discovery"},
    ]


def default_warnings():
    return [
        "No exact published API rate limit was found in reviewed materials; keep requests small and cache stable metadata.",
        "Official download notes state that missing values in tabular downloads are represented as -9999; check field notes before statistical interpretation.",
        "ArcGIS services can enforce maxRecordCount/transfer limits; use filters, fields, and pagination rather than broad full-table pulls.",
    ]


def next_actions_for_tables(items):
    return [f"deutschlandatlas indicator dossier --table {item['table']}" for item in items[:3] if item.get("table")] or ['deutschlandatlas tables search --term "Apotheken" --limit 5']


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
    try:
        value = int(raw) if raw else fallback
    except ValueError:
        value = fallback
    if value < 1:
        value = fallback
    if value > max_value and not flag_bool(parsed, "allow-large-output"):
        raise CLIError(2, "limit_exceeds_safe_max", f"limit {value} exceeds safe max {max_value}; pass --allow-large-output to override")
    return value


def int_flag(parsed, key, fallback):
    try:
        return int(parsed["flags"].get(key, fallback))
    except ValueError:
        return fallback


def int_value(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def table_from_url(raw):
    parts = str(raw).strip("/").split("/")
    for i, part in enumerate(parts):
        if part == "services" and i + 1 < len(parts):
            return parts[i + 1]
    return ""


def millis_to_utc(value):
    try:
        return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(int(value) / 1000))
    except (TypeError, ValueError):
        return ""


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
