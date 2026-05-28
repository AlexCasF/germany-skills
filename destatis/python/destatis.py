#!/usr/bin/env python3
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "destatis"
BASE_URL = "https://www-genesis.destatis.de/genesisWS/rest/2020"
UI_URL = "https://www-genesis.destatis.de/datenbank/online"
DOCS_URL = "https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html"

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")

RAW_PATHS = {
    "catalogue statistics": "/catalogue/statistics",
    "catalogue tables": "/catalogue/tables",
    "catalogue variables": "/catalogue/variables",
    "metadata table": "/metadata/table",
    "metadata timeseries": "/metadata/timeseries",
    "data table": "/data/table",
    "data timeseries": "/data/timeseries",
    "find search": "/find/find",
}


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
        elif argv[0] == "search":
            run_search(argv[1:])
        elif argv[:2] == ["table", "source"]:
            run_table_source(argv[2:])
        elif argv[:2] == ["table", "dossier"]:
            run_table_dossier(argv[2:])
        elif argv[:2] == ["table", "sample"]:
            run_table_sample(argv[2:])
        elif argv[:2] == ["timeseries", "dossier"]:
            run_timeseries_dossier(argv[2:])
        elif argv[:2] == ["variables", "explain"]:
            run_variables_explain(argv[2:])
        else:
            run_raw(argv)
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""destatis -- Destatis GENESIS-Online statistics CLI

Purpose
  Search and retrieve official German statistics from Destatis GENESIS-Online.

Fast paths
  destatis doctor
  destatis search --term "Indikator" --limit 5
  destatis table source --name <table-name>
  destatis table dossier --name <table-name>

Raw endpoint commands
  catalogue statistics|tables|variables
  metadata table|timeseries
  data table|timeseries
  find search

Research commands
  doctor
  search
  table source
  table dossier
  table sample
  timeseries dossier
  variables explain

Auth
  Prefer DESTATIS_USERNAME and DESTATIS_PASSWORD from the environment.
  --username and --password still work and are redacted from output.
  If no credentials are configured, the CLI uses GAST/GAST for public discovery.
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "table dossier":
        print("""destatis table dossier

Build a cautious evidence bundle for one GENESIS table code. With full
credentials it tries metadata and a small data sample; with guest credentials it
returns source metadata and structured warnings if protected endpoints return 401.
""")
    elif joined == "search":
        print("""destatis search

Friendly alias for the GENESIS find endpoint. Keeps output compact.

Example
  destatis search --term "Indikator" --limit 5
""")
    else:
        print_root_help()


def run_doctor(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    payload = envelope("doctor", "/helloworld/logincheck", None, cred)
    payload["summary"] = {
        "baseUrl": BASE_URL,
        "webUi": UI_URL,
        "docs": DOCS_URL,
        "authConfigured": not cred["guest"] or bool(os.environ.get("DESTATIS_USERNAME") or os.environ.get("DESTATIS_PASSWORD")),
        "credentialSource": cred["source"],
        "guestFallbackEnabled": cred["guest"],
        "publishedRateLimit": "not found in official Destatis docs reviewed; use small pagelength values and avoid parallel broad requests",
        "license": "Datenlizenz Deutschland - Namensnennung - Version 2.0 for GENESIS-Online usage per Destatis Open Data page",
    }
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings(cred)
    try:
        login = api_post("/helloworld/logincheck", {}, cred)
        payload["summary"]["health"] = {
            "ok": True,
            "message": login.get("Status"),
            "username": redact_username(login.get("Username", "")),
        }
    except Exception as exc:
        payload["status"] = "error"
        payload["summary"]["health"] = {"ok": False, "error": redact(str(exc))}
    try:
        found = api_post("/find/find", {"term": "Indikator", "category": "all", "pagelength": "1", "language": "de"}, cred)
        payload["summary"]["findCheck"] = {
            "ok": True,
            "status": found.get("Status"),
            "tablesFound": len(found.get("Tables") or []),
        }
    except Exception as exc:
        payload["summary"]["findCheck"] = {"ok": False, "error": redact(str(exc))}
    payload["nextActions"] = ['destatis search --term "Indikator" --limit 5', "destatis table source --name <table-name>"]
    emit(payload)


def run_search(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), parsed["flags"].get("selection"))
    if not term:
        raise CLIError(2, "missing_term", "search requires --term")
    limit = limit_flag(parsed, 5, 25)
    params = dict(parsed["params"])
    params.update({
        "term": term,
        "category": first_non_empty(parsed["flags"].get("category"), params.get("category"), "all"),
        "pagelength": str(limit),
        "language": first_non_empty(parsed["flags"].get("language"), params.get("language"), "de"),
    })
    data = api_post("/find/find", params, cred)
    items = compact_find(data, limit)
    payload = envelope("search", "/find/find", params, cred)
    payload["summary"] = {
        "term": term,
        "limitApplied": limit,
        "status": data.get("Status"),
        "statistics": len(data.get("Statistics") or []),
        "tables": len(data.get("Tables") or []),
        "timeseries": len(data.get("Timeseries") or []),
    }
    payload["items"] = items
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings(cred)
    payload["nextActions"] = next_actions_for_find(items)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_table_source(argv):
    parsed = parse_args(argv)
    name = required_name(parsed)
    cred = resolve_credentials(parsed)
    payload = envelope("table source", "/metadata/table", {"name": name}, cred)
    payload["summary"] = table_source_summary(name)
    payload["sources"] = sources_for_table(name)
    payload["warnings"] = ["Source URLs identify official GENESIS locations; table availability and metadata detail can depend on credentials."]
    payload["nextActions"] = [f"destatis table dossier --name {name}", f"destatis metadata table --param name={name}"]
    emit(payload)


def run_table_dossier(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    name = required_name(parsed)
    payload = envelope("table dossier", "/metadata/table", {"name": name}, cred)
    payload["summary"] = table_source_summary(name)
    payload["sources"] = sources_for_table(name)
    payload["warnings"] = standard_warnings(cred)
    payload["nextActions"] = [f"destatis table sample --name {name}", f"destatis variables explain --table {name}"]
    try:
        meta = api_post("/metadata/table", {"name": name, "language": parsed["flags"].get("language", "de")}, cred)
        payload["metadata"] = summarize_destatis_payload(meta)
        if flag_bool(parsed, "include-raw"):
            payload["rawMetadata"] = meta
    except Exception as exc:
        payload["metadata"] = {"available": False, "error": redact(str(exc))}
        payload["warnings"].append("Metadata request failed; guest credentials can be insufficient for metadata/data endpoints.")
    if flag_bool(parsed, "sample"):
        try:
            sample = api_post_text("/data/table", {"name": name, "area": "all", "format": "ffcsv", "compress": "true", "transpose": "false", "language": "de"}, cred)
            payload["sample"] = {"available": True, "preview": truncate(sample, 1200)}
        except Exception as exc:
            payload["sample"] = {"available": False, "error": redact(str(exc))}
            payload["warnings"].append("Data sample request failed; use personal GENESIS credentials for protected data endpoints.")
    emit(payload)


def run_table_sample(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    name = required_name(parsed)
    params = dict(parsed["params"])
    params.update({
        "name": name,
        "area": first_non_empty(parsed["flags"].get("area"), params.get("area"), "all"),
        "format": first_non_empty(parsed["flags"].get("format"), params.get("format"), "ffcsv"),
        "compress": first_non_empty(parsed["flags"].get("compress"), params.get("compress"), "true"),
        "transpose": first_non_empty(parsed["flags"].get("transpose"), params.get("transpose"), "false"),
        "language": first_non_empty(parsed["flags"].get("language"), params.get("language"), "de"),
    })
    payload = envelope("table sample", "/data/table", params, cred)
    payload["summary"] = table_source_summary(name)
    payload["sources"] = sources_for_table(name)
    payload["warnings"] = standard_warnings(cred)
    try:
        sample = api_post_text("/data/table", params, cred)
        payload["sample"] = {"available": True, "preview": truncate(sample, 1600)}
    except Exception as exc:
        payload["status"] = "partial"
        payload["sample"] = {"available": False, "error": redact(str(exc))}
    emit(payload)


def run_timeseries_dossier(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    name = required_name(parsed)
    params = {"name": name, "language": parsed["flags"].get("language", "de")}
    payload = envelope("timeseries dossier", "/metadata/timeseries", params, cred)
    payload["summary"] = {"name": name, "kind": "timeseries", "webUi": f"{UI_URL}/timeseries/{urllib.parse.quote(name)}"}
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings(cred)
    try:
        data = api_post("/metadata/timeseries", params, cred)
        payload["metadata"] = summarize_destatis_payload(data)
    except Exception as exc:
        payload["status"] = "partial"
        payload["metadata"] = {"available": False, "error": redact(str(exc))}
    emit(payload)


def run_variables_explain(argv):
    parsed = parse_args(argv)
    cred = resolve_credentials(parsed)
    table = first_non_empty(parsed["flags"].get("table"), parsed["flags"].get("name"), parsed["flags"].get("code"))
    if not table:
        raise CLIError(2, "missing_table", "variables explain requires --table")
    params = {"name": table, "language": parsed["flags"].get("language", "de")}
    payload = envelope("variables explain", "/catalogue/tables2variable", params, cred)
    payload["summary"] = {"table": table, "purpose": "discover variables/dimensions connected to a GENESIS table"}
    payload["sources"] = sources_for_table(table)
    payload["warnings"] = standard_warnings(cred)
    try:
        data = api_post("/catalogue/tables2variable", params, cred)
        payload["variables"] = summarize_destatis_payload(data)
    except Exception as exc:
        payload["status"] = "partial"
        payload["variables"] = {"available": False, "error": redact(str(exc))}
    payload["nextActions"] = [f"destatis table dossier --name {table}"]
    emit(payload)


def run_raw(argv):
    if len(argv) < 2:
        raise CLIError(2, "unknown_command", "expected command group and action")
    command = " ".join(argv[:2])
    path = RAW_PATHS.get(command)
    if not path:
        raise CLIError(2, "unknown_command", "unknown command path: " + " ".join(argv))
    parsed = parse_args(argv[2:])
    cred = resolve_credentials(parsed)
    params = dict(parsed["params"])
    for key, value in parsed["flags"].items():
        if key not in {"username", "password", "limit", "include-raw", "sample"}:
            params[key] = value
    if command == "find search" and not params.get("term") and parsed["flags"].get("selection"):
        params["term"] = parsed["flags"]["selection"]
    params.setdefault("language", "de")
    params.setdefault("pagelength", str(limit_flag(parsed, 10, 100)))
    print(api_post_text(path, params, cred))


def api_post(path, params, cred):
    return json.loads(api_post_text(path, params, cred))


def api_post_text(path, params, cred):
    data = dict(params or {})
    data["username"] = cred["username"]
    data["password"] = cred["password"]
    encoded = urllib.parse.urlencode(data).encode("utf-8")
    req = urllib.request.Request(
        BASE_URL + path,
        data=encoded,
        headers={"Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json,text/plain,*/*"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            return resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", "replace")
        raise CLIError(1, "http_error", f"HTTP {exc.code} from Destatis GENESIS API: {truncate(body, 280)}")


def compact_find(data, limit):
    items = []
    for kind, key in [("statistic", "Statistics"), ("table", "Tables"), ("timeseries", "Timeseries"), ("cube", "Cubes")]:
        for row in data.get(key) or []:
            if len(items) >= limit:
                return items
            code = row.get("Code", "")
            items.append({
                "kind": kind,
                "code": code,
                "title": row.get("Content", ""),
                "time": row.get("Time", ""),
                "cubes": row.get("Cubes", ""),
                "sources": source_links(kind, code),
            })
    return items


def summarize_destatis_payload(data):
    return {
        "status": data.get("Status"),
        "ident": data.get("Ident"),
        "parameters": redact_param_map(data.get("Parameter") or {}),
        "objectKeys": list(data.keys()),
        "preview": truncate(json.dumps(data, ensure_ascii=False), 1400),
    }


def table_source_summary(name):
    return {
        "name": name,
        "kind": "table",
        "apiBaseUrl": BASE_URL,
        "webUi": f"{UI_URL}/table/{urllib.parse.quote(name)}",
        "license": "Datenlizenz Deutschland - Namensnennung - Version 2.0 per Destatis Open Data page",
    }


def source_links(kind, code):
    if kind == "table":
        return sources_for_table(code)
    if kind == "statistic":
        return [{"title": "GENESIS statistic page", "url": f"{UI_URL}/statistic/{urllib.parse.quote(code)}", "kind": "web-ui"}, {"title": "GENESIS REST API", "url": BASE_URL, "kind": "api"}]
    return default_sources()


def sources_for_table(name):
    return [
        {"title": "GENESIS table page", "url": f"{UI_URL}/table/{urllib.parse.quote(name)}", "kind": "web-ui"},
        {"title": "GENESIS metadata endpoint", "url": f"{BASE_URL}/metadata/table", "kind": "api"},
        {"title": "GENESIS data endpoint", "url": f"{BASE_URL}/data/table", "kind": "api"},
        {"title": "Destatis GENESIS API/Webservice page", "url": DOCS_URL, "kind": "docs"},
    ]


def default_sources():
    return [
        {"title": "Destatis GENESIS API/Webservices page", "url": DOCS_URL, "kind": "docs"},
        {"title": "GENESIS-Online database", "url": UI_URL, "kind": "web-ui"},
        {"title": "GENESIS REST base URL", "url": BASE_URL, "kind": "api"},
    ]


def standard_warnings(cred):
    warnings = [
        "Use small pagelength values for discovery; inspect metadata before requesting data.",
        "Preserve table/statistic codes, units, time periods, and source dates in final answers.",
        "Credentials are redacted from normalized output and errors.",
    ]
    if cred["guest"]:
        warnings.append("Using GAST/GAST fallback: discovery works, but metadata/data endpoints may return 401; configure DESTATIS_USERNAME and DESTATIS_PASSWORD for full access.")
    return warnings


def next_actions_for_find(items):
    actions = []
    for item in items:
        if item.get("kind") == "table":
            actions.append(f"destatis table dossier --name {item['code']}")
        elif item.get("kind") == "timeseries":
            actions.append(f"destatis timeseries dossier --name {item['code']}")
        elif item.get("kind") == "statistic":
            actions.append(f"destatis catalogue tables --param name={item['code']}")
        if len(actions) >= 5:
            break
    return actions


def parse_args(argv):
    out = {"flags": {}, "params": {}, "positionals": []}
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg == "--param" and i + 1 < len(argv):
            add_param(out["params"], argv[i + 1])
            i += 2
            continue
        if arg.startswith("--param="):
            add_param(out["params"], arg[len("--param="):])
            i += 1
            continue
        if arg.startswith("--"):
            name = arg[2:]
            if "=" in name:
                key, value = name.split("=", 1)
                out["flags"][key] = value
            elif i + 1 < len(argv) and not argv[i + 1].startswith("--"):
                out["flags"][name] = argv[i + 1]
                i += 1
            else:
                out["flags"][name] = "true"
        else:
            out["positionals"].append(arg)
        i += 1
    return out


def add_param(params, raw):
    if "=" in raw:
        key, value = raw.split("=", 1)
        params[key] = value


def resolve_credentials(parsed):
    username = first_non_empty(parsed["flags"].get("username"), os.environ.get("DESTATIS_USERNAME"), "GAST")
    password = first_non_empty(parsed["flags"].get("password"), os.environ.get("DESTATIS_PASSWORD"), "GAST")
    source = "guest:GAST"
    if parsed["flags"].get("username") or parsed["flags"].get("password"):
        source = "flags:redacted"
    elif os.environ.get("DESTATIS_USERNAME") or os.environ.get("DESTATIS_PASSWORD"):
        source = "env:DESTATIS_USERNAME/DESTATIS_PASSWORD"
    return {"username": username, "password": password, "source": source, "guest": username == "GAST" and password == "GAST"}


def envelope(command, path, params, cred):
    request = {"method": "POST", "url": BASE_URL + path, "credentialSource": cred["source"], "redactedFields": ["username", "password"]}
    if params is not None:
        request["params"] = redact_param_map(params)
    return {"status": "ok", "tool": APP_NAME, "command": command, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "request": request}


def required_name(parsed):
    name = first_non_empty(parsed["flags"].get("name"), parsed["flags"].get("code"), parsed["flags"].get("table"))
    if not name and parsed["positionals"]:
        name = parsed["positionals"][0]
    if not name:
        raise CLIError(2, "missing_name", "requires --name, --code, or --table")
    return name


def redact_param_map(params):
    return {key: ("REDACTED" if is_secret_key(key) else value) for key, value in params.items()}


def is_secret_key(key):
    lower = key.lower()
    return lower in {"username", "password", "passwort"} or "token" in lower


def emit(payload):
    print(json.dumps(payload, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": redact(message)}})
    sys.exit(exit_code)


def limit_flag(parsed, default, maximum):
    raw = parsed["flags"].get("limit") or parsed["params"].get("pagelength")
    try:
        value = int(raw or default)
    except ValueError:
        value = default
    return max(1, min(value, maximum))


def flag_bool(parsed, name):
    return str(parsed["flags"].get(name, "")).lower() in {"1", "true", "yes"}


def first_non_empty(*values):
    for value in values:
        if value is not None and str(value).strip():
            return str(value)
    return ""


def truncate(text, limit):
    text = " ".join(str(text).split())
    return text if len(text) <= limit else text[: limit - 3] + "..."


def redact_username(username):
    return username if username in {"", "GAST"} else "REDACTED"


def redact(text):
    text = re.sub(r"(?i)(username|password|passwort|token)=([^&\s]+)", r"\1=REDACTED", str(text))
    text = re.sub(r"(?i)(--(?:username|password|token)\s+)([^\s]+)", r"\1REDACTED", text)
    return text


def is_help(arg):
    return arg in {"-h", "--help", "help"}


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
