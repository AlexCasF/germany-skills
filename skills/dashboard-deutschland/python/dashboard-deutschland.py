#!/usr/bin/env python3
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "dashboard-deutschland"
BASE_URL = "https://www.dashboard-deutschland.de"
DASHBOARDS_URL = BASE_URL + "/api/dashboard/get"
INDICATORS_URL = BASE_URL + "/api/tile/indicators"
GEO_URL = BASE_URL + "/geojson/de-all.geo.json"
DESTATIS_URL = "https://www.destatis.de/DE/Ueber-uns/Aufgaben/dashboards.html"
BMWE_URL = "https://www.bundeswirtschaftsministerium.de/Redaktion/DE/Dossier/WirtschaftlicheEntwicklung/dashboard-deutschland.html"
PYPI_URL = "https://pypi.org/project/de-dashboarddeutschland/"
OPENAPI_REPO_URL = "https://github.com/bundesAPI/dashboard-deutschland-api"
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
        elif argv[:2] == ["dashboard", "get"]:
            emit(fetch_json(with_params(DASHBOARDS_URL, parse_args(argv[2:])["params"])))
        elif argv[:2] == ["dashboards", "list"]:
            run_dashboards_list(argv[2:])
        elif argv[:2] == ["dashboard", "dossier"]:
            run_dashboard_dossier(argv[2:])
        elif argv[0] == "indicators":
            run_indicators_raw(argv[1:])
        elif argv[:2] == ["indicator", "search"]:
            run_indicator_search(argv[2:])
        elif argv[:2] == ["indicator", "get"]:
            run_indicator_get(argv[2:])
        elif argv[:2] == ["indicator", "data"]:
            run_indicator_data(argv[2:])
        elif argv[:2] == ["indicator", "source"]:
            run_indicator_source(argv[2:])
        elif argv[0] == "source":
            run_indicator_source(argv[1:])
        elif argv[0] == "geo":
            run_geo(argv[1:])
        else:
            raise CLIError(2, "unknown_command", "unknown command; run dashboard-deutschland --help")
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""dashboard-deutschland -- Dashboard Deutschland research CLI

Purpose
  Discover and normalize curated Dashboard Deutschland indicators.

Fast paths
  dashboard-deutschland doctor
  dashboard-deutschland dashboards list --limit 5
  dashboard-deutschland indicator search --term "Indikator" --limit 5
  dashboard-deutschland indicator get --id <indicator-id>
  dashboard-deutschland indicator data --id <indicator-id> --limit 5
  dashboard-deutschland dashboard dossier --id arbeitsmarkt --indicator-limit 3

Raw endpoint commands
  dashboard get [--param key=value]
  indicators --param ids=<indicator-id>
  geo
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "indicator data":
        print("""dashboard-deutschland indicator data --id <indicator-id> [--limit n]

Extract chart-ready series from an indicator tile. Use --series to filter
series names and --from-start for earliest points.
""")
    elif joined == "dashboard dossier":
        print("""dashboard-deutschland dashboard dossier --id <dashboard-id> [--indicator-limit n]

Bundle dashboard metadata and a small set of normalized indicator summaries.
""")
    elif joined == "geo":
        print("""dashboard-deutschland geo

Raw GeoJSON endpoint wrapper. The endpoint returned 403 AccessDenied in
live tests; doctor reports this as degraded.
""")
    else:
        print_root_help()


def run_doctor(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 2, 10)
    payload = envelope("doctor", DASHBOARDS_URL, None)
    warnings = default_warnings()
    summary = {
        "authRequired": False,
        "publishedRateLimit": "No exact public rate limit was found in reviewed materials. Use small batches and avoid repeated all-indicator pulls.",
        "fairUseHints": [
            "Use dashboards list or indicator search before fetching indicator data.",
            "Fetch indicator data by explicit ID.",
            "Use small --limit values for chart points.",
            "Back off on 429, 5xx, or gateway/object-storage errors.",
        ],
    }
    try:
        dashboards = fetch_dashboards()
        ids = unique_indicator_ids(dashboards)
        summary["dashboardEndpoint"] = {"ok": True, "dashboards": len(dashboards), "uniqueIndicatorIds": len(ids), "sampleDashboards": compact_dashboards(dashboards, limit)}
        indicators = fetch_indicators(ids[:1]) if ids else []
        summary["indicatorEndpoint"] = {"ok": True, "sample": compact_indicators(indicators, 1)}
    except Exception as exc:
        summary["dashboardEndpoint"] = {"ok": False, "error": str(exc)}
        payload["status"] = "degraded"
    status, content_type, body = fetch_raw(GEO_URL)
    geo_ok = 200 <= status < 300
    summary["geoEndpoint"] = {"url": GEO_URL, "statusCode": status, "ok": geo_ok, "contentType": content_type, "bodyPreview": truncate(strip_space(body), 180)}
    if not geo_ok:
        payload["status"] = "degraded"
        warnings.append("The documented GeoJSON endpoint currently returns 403 AccessDenied; use geo as a diagnostic command.")
    payload.setdefault("status", "ok")
    payload["summary"] = summary
    payload["sources"] = default_sources()
    payload["warnings"] = warnings
    payload["nextActions"] = ['dashboard-deutschland indicator search --term "Indikator" --limit 5', "dashboard-deutschland dashboards list --limit 5"]
    emit(payload)


def run_indicators_raw(argv):
    parsed = parse_args(argv)
    params = dict(parsed["params"])
    if parsed["flags"].get("id"):
        params["ids"] = parsed["flags"]["id"]
    if parsed["flags"].get("ids"):
        params["ids"] = parsed["flags"]["ids"]
    emit(fetch_json(with_params(INDICATORS_URL, params)))


def run_geo(argv):
    status, content_type, body = fetch_raw(GEO_URL)
    if not (200 <= status < 300):
        raise CLIError(1, "geo_endpoint_failed", f"geo endpoint status {status} content-type {content_type} body: {truncate(strip_space(body), 220)}")
    try:
        emit(json.loads(body))
    except json.JSONDecodeError:
        print(body)


def run_dashboards_list(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, DEFAULT_LIMIT, 50)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"])).lower()
    dashboards = fetch_dashboards()
    filtered = [item for item in dashboards if not term or term in dashboard_search_text(item).lower()]
    payload = envelope("dashboards list", DASHBOARDS_URL, {"term": term, "limit": limit})
    payload["summary"] = {"available": len(filtered), "returned": min(limit, len(filtered)), "totalDashboards": len(dashboards), "uniqueIndicatorIds": len(unique_indicator_ids(dashboards))}
    payload["items"] = compact_dashboards(filtered, limit)
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["dashboard-deutschland dashboard dossier --id arbeitsmarkt --indicator-limit 3"]
    emit(payload)


def run_indicator_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "indicator search requires --term")
    limit = limit_flag(parsed, 5, 50)
    dashboards = fetch_dashboards()
    ids = unique_indicator_ids(dashboards)
    indicators = fetch_indicators(ids)
    needle = term.lower()
    matches = [item for item in indicators if needle in indicator_search_text(item).lower()]
    payload = envelope("indicator search", INDICATORS_URL, {"term": term, "limit": limit})
    payload["summary"] = {"term": term, "matches": len(matches), "searchedIndicatorIds": len(ids), "returned": min(limit, len(matches))}
    payload["items"] = compact_indicators(matches, limit)
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_for_indicators(matches)
    emit(payload)


def run_indicator_get(argv):
    parsed = parse_args(argv)
    item_id = required_id(parsed)
    indicator = fetch_one_indicator(item_id)
    config = parse_tile_config(indicator)
    payload = envelope("indicator get", INDICATORS_URL + "?ids=" + urllib.parse.quote(item_id), {"id": item_id})
    payload["summary"] = indicator_summary(indicator, config)
    payload["items"] = [{"summary": indicator_summary(indicator, config), "textSnippets": text_snippets(config, "", 5), "widgets": widgets(config), "chartSeries": series_summaries(config)}]
    payload["sources"] = sources_for_indicator(indicator, config)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"dashboard-deutschland indicator data --id {item_id} --limit 10", f"dashboard-deutschland indicator source --id {item_id}"]
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = {"indicator": indicator, "config": config}
    emit(payload)


def run_indicator_data(argv):
    parsed = parse_args(argv)
    item_id = required_id(parsed)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    series_term = first_non_empty(parsed["flags"].get("series"), parsed["flags"].get("grep")).lower()
    indicator = fetch_one_indicator(item_id)
    config = parse_tile_config(indicator)
    series = extract_series(config, limit, flag_bool(parsed, "from-start"), series_term)
    payload = envelope("indicator data", INDICATORS_URL + "?ids=" + urllib.parse.quote(item_id), {"id": item_id, "limit": limit, "series": series_term})
    payload["summary"] = {"id": item_id, "title": first_non_empty(config.get("title"), indicator.get("title")), "seriesReturned": len(series), "pointsPerSeries": limit, "dataVersionDate": config.get("dataVersionDate"), "lastUpdated": millis_summary(config.get("lastUpdated"))}
    payload["items"] = series
    payload["sources"] = sources_for_indicator(indicator, config)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"dashboard-deutschland indicator source --id {item_id}"]
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = config
    emit(payload)


def run_indicator_source(argv):
    parsed = parse_args(argv)
    item_id = required_id(parsed)
    indicator = fetch_one_indicator(item_id)
    config = parse_tile_config(indicator)
    payload = envelope("indicator source", INDICATORS_URL + "?ids=" + urllib.parse.quote(item_id), {"id": item_id})
    payload["summary"] = {"id": item_id, "title": first_non_empty(config.get("title"), indicator.get("title")), "sourceCount": len(sources_for_indicator(indicator, config))}
    payload["sources"] = sources_for_indicator(indicator, config)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"dashboard-deutschland indicator data --id {item_id} --limit 10"]
    emit(payload)


def run_dashboard_dossier(argv):
    parsed = parse_args(argv)
    indicator_limit = limit_flag_name(parsed, "indicator-limit", 3, 10)
    dashboards = fetch_dashboards()
    dashboard = find_dashboard(dashboards, parsed)
    ids = dashboard_indicator_ids(dashboard)[:indicator_limit]
    indicators = fetch_indicators(ids) if ids else []
    payload = envelope("dashboard dossier", DASHBOARDS_URL, {"id": dashboard.get("id"), "indicatorLimit": indicator_limit})
    payload["summary"] = compact_dashboard(dashboard)
    payload["items"] = compact_indicators(indicators, len(indicators))
    payload["sources"] = sources_for_dashboard(dashboard)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"dashboard-deutschland indicator data --id {item_id} --limit 10" for item_id in ids[:3]]
    emit(payload)


def fetch_dashboards():
    data = fetch_json(DASHBOARDS_URL)
    return data if isinstance(data, list) else []


def fetch_one_indicator(item_id):
    items = fetch_indicators([item_id])
    if not items:
        raise CLIError(2, "indicator_not_found", "indicator not found: " + item_id)
    return items[0]


def fetch_indicators(ids):
    if not ids:
        raise CLIError(2, "missing_ids", "indicator IDs required")
    out = []
    for start in range(0, len(ids), 20):
        chunk = ids[start:start + 20]
        data = fetch_json(with_params(INDICATORS_URL, {"ids": ";".join(chunk)}))
        if isinstance(data, list):
            out.extend(data)
    return out


def fetch_json(request_url):
    status, _, body = fetch_raw(request_url)
    if not (200 <= status < 300):
        raise RuntimeError(f"upstream status {status} from {request_url}: {truncate(strip_space(body), 300)}")
    return json.loads(body)


def fetch_raw(request_url):
    req = urllib.request.Request(request_url, headers={"User-Agent": "germany-skills/dashboard-deutschland-python"})
    try:
        with urllib.request.urlopen(req, timeout=45) as response:
            return response.status, response.headers.get("Content-Type", ""), response.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.headers.get("Content-Type", ""), exc.read().decode("utf-8", "replace")


def parse_tile_config(indicator):
    raw = indicator.get("json") or ""
    if not raw:
        raise CLIError(2, "missing_embedded_json", "indicator has no embedded json field")
    return json.loads(raw)


def compact_dashboards(dashboards, limit):
    return [compact_dashboard(item) for item in dashboards[:limit]]


def compact_dashboard(dashboard):
    ids = dashboard_indicator_ids(dashboard)
    return {"id": dashboard.get("id"), "name": dashboard.get("name"), "nameEn": dashboard.get("nameEn"), "description": truncate(strip_html(dashboard.get("description") or ""), 420), "category": compact_category(dashboard.get("category") or {}), "tags": dashboard.get("tags") or [], "indicatorCount": len(ids), "indicatorIds": ids[:12], "nextActions": [f"dashboard-deutschland dashboard dossier --id {dashboard.get('id')} --indicator-limit 3"]}


def compact_indicators(indicators, limit):
    out = []
    for indicator in indicators[:limit]:
        try:
            config = parse_tile_config(indicator)
        except Exception:
            config = {}
        summary = indicator_summary(indicator, config)
        summary["nextActions"] = [f"dashboard-deutschland indicator data --id {indicator.get('id')} --limit 10", f"dashboard-deutschland indicator source --id {indicator.get('id')}"]
        out.append(summary)
    return out


def indicator_summary(indicator, config):
    return {"id": indicator.get("id"), "title": first_non_empty(config.get("title"), indicator.get("title")), "apiTitle": indicator.get("title"), "category": config.get("category"), "tags": config.get("tags") or [], "sourceCount": len(source_entries(config)), "sources": source_entries(config), "componentCount": len(config.get("components") or []), "seriesCount": len(series_summaries(config)), "widgetCount": len(widgets(config)), "dataVersionDate": config.get("dataVersionDate"), "dateUpload": config.get("dateUpload"), "lastUpdated": millis_summary(config.get("lastUpdated"))}


def extract_series(config, limit, from_start, series_term):
    out = []
    for component in config.get("components") or []:
        for series in ((component.get("chart") or {}).get("series") or []):
            name = first_non_empty((series.get("custom") or {}).get("name"), series.get("name"))
            sid = series.get("id") or ""
            if series_term and series_term not in (name + " " + sid).lower():
                continue
            points = series.get("data") or []
            selected = points[:limit] if from_start else points[-limit:]
            out.append({"id": sid, "name": name, "color": series.get("color"), "pointCount": len(points), "points": selected, "firstPoint": points[0] if points else None, "lastPoint": points[-1] if points else None})
    return out


def series_summaries(config):
    out = []
    for component in config.get("components") or []:
        for series in ((component.get("chart") or {}).get("series") or []):
            points = series.get("data") or []
            out.append({"id": series.get("id") or "", "name": first_non_empty((series.get("custom") or {}).get("name"), series.get("name")), "pointCount": len(points), "firstPoint": points[0] if points else None, "lastPoint": points[-1] if points else None})
    return out


def widgets(config):
    out = []
    for component in config.get("components") or []:
        for widget in component.get("widgets") or []:
            out.append({"num": widget.get("num"), "desc": strip_html(widget.get("desc") or ""), "icon": widget.get("icon")})
    return out


def text_snippets(config, grep, limit):
    needle = grep.lower()
    out = []
    for component in config.get("components") or []:
        text = strip_html(first_non_empty(component.get("text"), component.get("infoButtonText"), component.get("description")))
        if len(text) > 20 and (not needle or needle in text.lower()):
            out.append({"text": truncate(text, 700), "type": component.get("type")})
        if len(out) >= limit:
            break
    return out


def source_entries(config):
    out = []
    for source in config.get("sources") or []:
        out.append({"title": first_non_empty(source.get("name"), "Dashboard Deutschland source"), "url": source.get("link") or "", "kind": "indicator_source", "quality": source.get("quality")})
    if not out and config.get("source"):
        out.append({"title": "Dashboard source field", "url": "", "kind": "source_text", "text": strip_html(config.get("source") or "")})
    return out


def sources_for_indicator(indicator, config):
    return [{"title": "Dashboard Deutschland indicator API", "url": INDICATORS_URL + "?ids=" + urllib.parse.quote(indicator.get("id") or ""), "kind": "api_endpoint"}, {"title": "Dashboard Deutschland", "url": BASE_URL, "kind": "official_dashboard"}] + source_entries(config)


def sources_for_dashboard(dashboard):
    return [{"title": "Dashboard Deutschland dashboard API", "url": DASHBOARDS_URL, "kind": "api_endpoint"}, {"title": "Dashboard Deutschland", "url": BASE_URL, "kind": "official_dashboard"}, {"title": "Destatis dashboard page", "url": DESTATIS_URL, "kind": "official_context"}]


def default_sources():
    return [{"title": "Dashboard Deutschland", "url": BASE_URL, "kind": "official_dashboard"}, {"title": "Dashboard Deutschland dashboard API", "url": DASHBOARDS_URL, "kind": "api_endpoint"}, {"title": "Dashboard Deutschland indicator API", "url": INDICATORS_URL, "kind": "api_endpoint"}, {"title": "Dashboard Deutschland GeoJSON endpoint", "url": GEO_URL, "kind": "api_endpoint"}, {"title": "Destatis dashboards page", "url": DESTATIS_URL, "kind": "official_context"}, {"title": "BMWE Dashboard Deutschland page", "url": BMWE_URL, "kind": "official_context"}, {"title": "PyPI generated DashboardDeutschland package", "url": PYPI_URL, "kind": "openapi_reference"}, {"title": "Dashboard Deutschland OpenAPI wrapper", "url": OPENAPI_REPO_URL, "kind": "openapi_reference"}]


def default_warnings():
    return ["No exact published API rate limit was found in reviewed materials; use small batches and avoid repeated all-indicator pulls.", "Indicator tiles contain an embedded JSON string; parse it before interpreting chart data, sources, widgets, or update dates.", "The documented GeoJSON endpoint returned 403 AccessDenied in live tests.", "Dashboard Deutschland is curated and mixed-source; for deep statistical table work use Destatis/GENESIS where appropriate."]


def unique_indicator_ids(dashboards):
    seen, ids = set(), []
    for dashboard in dashboards:
        for item_id in dashboard_indicator_ids(dashboard):
            if item_id not in seen:
                seen.add(item_id)
                ids.append(item_id)
    return sorted(ids)


def dashboard_indicator_ids(dashboard):
    ids = []
    for tile in dashboard.get("layoutTiles") or []:
        item_id = first_non_empty(tile.get("indicatorid"), tile.get("indicatorId"))
        if item_id:
            ids.append(item_id)
    return ids


def find_dashboard(dashboards, parsed):
    wanted = first_non_empty(parsed["flags"].get("id"), parsed["flags"].get("name"), " ".join(parsed["positionals"])).lower()
    if not wanted:
        raise CLIError(2, "missing_dashboard", "dashboard dossier requires --id or --name")
    for dashboard in dashboards:
        if (dashboard.get("id") or "").lower() == wanted or wanted in (dashboard.get("name") or "").lower():
            return dashboard
    raise CLIError(2, "dashboard_not_found", "dashboard not found: " + wanted)


def dashboard_search_text(dashboard):
    return " ".join([dashboard.get("id") or "", dashboard.get("name") or "", dashboard.get("nameEn") or "", dashboard.get("description") or "", (dashboard.get("category") or {}).get("name") or ""] + (dashboard.get("tags") or []) + dashboard_indicator_ids(dashboard))


def indicator_search_text(indicator):
    try:
        config = parse_tile_config(indicator)
    except Exception:
        config = {}
    parts = [indicator.get("id") or "", indicator.get("title") or "", config.get("title") or "", config.get("category") or "", config.get("source") or "", config.get("dataVersionDate") or "", config.get("dateUpload") or ""]
    parts.extend(config.get("tags") or [])
    for source in source_entries(config):
        parts.extend([source.get("title") or "", source.get("url") or ""])
    for snippet in text_snippets(config, "", 8):
        parts.append(snippet.get("text") or "")
    return " ".join(parts)


def next_actions_for_indicators(items):
    actions = []
    for item in items[:3]:
        item_id = item.get("id")
        actions.extend([f"dashboard-deutschland indicator get --id {item_id}", f"dashboard-deutschland indicator data --id {item_id} --limit 10"])
    return actions or ['dashboard-deutschland indicator search --term "Arbeitsmarkt" --limit 5']


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


def required_id(parsed):
    item_id = first_non_empty(parsed["flags"].get("id"), parsed["flags"].get("ids"), parsed["positionals"][0] if parsed["positionals"] else "")
    if not item_id:
        raise CLIError(2, "missing_id", "command requires --id")
    return item_id


def envelope(command, request_url, request):
    return {"status": "ok", "tool": APP_NAME, "command": command, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "request": {"method": "GET", "url": request_url, "params": request}, "summary": {}, "items": [], "sources": [], "warnings": [], "nextActions": []}


def emit(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})
    sys.exit(exit_code)


def with_params(base, params):
    return base if not params else base + "?" + urllib.parse.urlencode(params)


def compact_category(category):
    return {"id": category.get("id"), "name": category.get("name"), "nameEn": category.get("nameEn"), "description": truncate(strip_html(category.get("description") or ""), 300)}


def millis_summary(value):
    try:
        ms = int(value)
    except (TypeError, ValueError):
        return {}
    return {"epochMs": ms, "iso": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(ms / 1000))}


def limit_flag(parsed, fallback, max_value):
    return limit_flag_name(parsed, "limit", fallback, max_value)


def limit_flag_name(parsed, name, fallback, max_value):
    try:
        value = int(parsed["flags"].get(name, fallback))
    except ValueError:
        value = fallback
    if value > max_value and not flag_bool(parsed, "allow-large-output"):
        raise CLIError(2, "limit_exceeds_safe_max", f"{name} {value} exceeds safe max {max_value}; pass --allow-large-output to override")
    return max(1, value)


def flag_bool(parsed, key):
    return str(parsed["flags"].get(key, "")).lower() in {"true", "1", "yes", "y"}


def first_non_empty(*values):
    for value in values:
        if value is not None and str(value).strip():
            return str(value).strip()
    return ""


def is_help(value):
    return value in {"--help", "-h", "help"}


TAG_RE = re.compile(r"<[^>]+>")
SPACE_RE = re.compile(r"\s+")


def strip_html(value):
    return strip_space(TAG_RE.sub(" ", value.replace("&nbsp;", " ").replace("\u00a0", " ")))


def strip_space(value):
    return SPACE_RE.sub(" ", value).strip()


def truncate(value, max_len):
    return value if len(value) <= max_len else value[:max_len] + "..."


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
