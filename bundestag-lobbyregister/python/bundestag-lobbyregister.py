#!/usr/bin/env python3
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "bundestag-lobbyregister"
BASE_URL = "https://api.lobbyregister.bundestag.de/rest/v2"
PUBLIC_URL = "https://www.lobbyregister.bundestag.de"
LEGACY_V1_URL = "https://www.lobbyregister.bundestag.de/sucheDetailJson"

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")


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
        elif argv[0] == "statistics":
            run_statistics(argv[1:])
        elif argv[0] == "search":
            run_search(argv[1:])
        elif argv[:2] == ["entry", "get"]:
            run_entry_get(argv[2:])
        elif argv[:2] == ["entry", "source"]:
            run_entry_source(argv[2:])
        elif argv[:2] == ["entry", "dossier"]:
            run_entry_dossier(argv[2:])
        elif argv[:2] == ["financial", "summary"]:
            run_financial_summary(argv[2:])
        elif argv[:2] == ["statements", "list"]:
            run_statements_list(argv[2:])
        elif argv[:2] == ["v1", "search"]:
            run_v1_search(argv[2:])
        else:
            raise CLIError(2, "unknown_command", "unknown command path: " + " ".join(argv))
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""bundestag-lobbyregister -- Bundestag Lobbyregister research CLI

Purpose
  Search and cite public lobby-register data for interests represented
  toward the German Bundestag and Federal Government.

Fast paths
  bundestag-lobbyregister doctor
  bundestag-lobbyregister search --term "Bundesverband Soziokultur" --limit 3
  bundestag-lobbyregister entry dossier --register-number R001255 --grep "Foerderung"
  bundestag-lobbyregister financial summary --register-number R001255

Research commands
  doctor
  statistics
  search
  entry get
  entry source
  entry dossier
  financial summary
  statements list

Legacy command
  v1 search

Auth
  Prefer LOBBYREGISTER_API_KEY from the environment.
  --apikey still works for local compatibility and is redacted from output.
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "entry dossier":
        print("""bundestag-lobbyregister entry dossier

Builds a compact evidence bundle for one register entry.

Examples
  bundestag-lobbyregister entry dossier --register-number R001255 --grep "Laerm"
  bundestag-lobbyregister entry dossier --name "Bundesverband Soziokultur"
""")
    elif joined == "search":
        print("""bundestag-lobbyregister search

Safe V2 free-text search with compact summaries and a small default limit.
""")
    elif joined == "entry get":
        print("""bundestag-lobbyregister entry get

Fetch one official V2 register entry by register number.
""")
    elif joined == "financial summary":
        print("""bundestag-lobbyregister financial summary

Normalize financial ranges, funding, donations, membership fees, public
allowances, annual-report links, and caveats for one register entry.
""")
    else:
        print_root_help()


def run_doctor(argv):
    parsed = parse_args(argv)
    key = api_key(parsed)
    payload = envelope("doctor", f"{BASE_URL}/statistics/registerentries?format=json")
    payload["summary"] = {
        "authRequired": True,
        "apiKeyConfigured": bool(key),
        "apiKeySource": key_source(parsed),
        "baseUrl": BASE_URL,
        "publicRegisterUrl": PUBLIC_URL,
        "openApiYaml": f"{BASE_URL}/R2.21-de.yaml",
        "swaggerUi": f"{BASE_URL}/swagger-ui/",
        "termsAndOpenDataPage": f"{PUBLIC_URL}/informationen-und-hilfe/open-data-1049716",
        "publishedRateLimit": "not found in official docs reviewed; use small limits and retry politely",
        "recommendedDefaultLimit": 5,
    }
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings()
    if not key:
        payload["warnings"].append("LOBBYREGISTER_API_KEY is not configured; live V2 calls will fail.")
        payload["nextActions"] = ["Set LOBBYREGISTER_API_KEY, then run: bundestag-lobbyregister statistics"]
        emit(payload)
        return
    try:
        data, request_url = api_json("/statistics/registerentries", {"format": "json"}, key)
        payload["summary"]["health"] = {
            "ok": True,
            "sourceDate": data.get("sourceDate"),
            "totalLobbyists": get(data, "lobbyists", "totalNumber"),
            "activeLobbyists": get(data, "lobbyists", "active", "number"),
            "inactiveLobbyists": get(data, "lobbyists", "inactive", "number"),
        }
        payload["request"]["url"] = request_url
    except Exception as exc:
        payload["status"] = "error"
        payload["summary"]["health"] = {"ok": False, "error": redact(str(exc))}
    payload["nextActions"] = [
        'bundestag-lobbyregister search --term "Bundesverband" --limit 3',
        "bundestag-lobbyregister entry dossier --register-number R001255",
    ]
    emit(payload)


def run_statistics(argv):
    parsed = parse_args(argv)
    key = require_key(parsed)
    data, request_url = api_json("/statistics/registerentries", {"format": "json"}, key)
    payload = envelope("statistics", request_url)
    payload["summary"] = {
        "source": data.get("source"),
        "sourceDate": data.get("sourceDate"),
        "totalLobbyists": get(data, "lobbyists", "totalNumber"),
        "activeLobbyists": get(data, "lobbyists", "active", "number"),
        "inactiveLobbyists": get(data, "lobbyists", "inactive", "number"),
        "peopleInvolved": get(data, "lobbyists", "peopleInvolvedInLobbyistWork", "totalNumber"),
    }
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    payload["sources"] = default_sources()
    payload["nextActions"] = ['bundestag-lobbyregister search --term "Energie" --limit 5']
    emit(payload)


def run_search(argv):
    parsed = parse_args(argv)
    key = require_key(parsed)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), parsed["flags"].get("name"))
    if not term:
        raise CLIError(2, "missing_term", "search requires --term, --q, or --name")
    limit = limit_flag(parsed, 5, 25)
    params = {"format": "json", "q": term}
    if parsed["flags"].get("cursor"):
        params["cursor"] = parsed["flags"]["cursor"]
    data, request_url = api_json("/registerentries", params, key)
    items = [summarize_entry(entry) for entry in data.get("results", [])[:limit]]
    payload = envelope("search", request_url)
    payload["summary"] = {
        "query": term,
        "returnedByApi": data.get("resultCount"),
        "totalResultCount": data.get("totalResultCount"),
        "limitApplied": limit,
        "cursorPresent": bool(data.get("cursor")),
        "sourceDate": data.get("sourceDate"),
    }
    payload["items"] = items
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = search_next_actions(items)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_entry_get(argv):
    parsed = parse_args(argv)
    entry, request_url = get_entry_from_args(parsed)
    payload = envelope("entry get", request_url)
    payload["summary"] = summarize_entry(entry)
    payload["sources"] = entry_sources(entry)
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = next_actions_for_entry(entry)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = entry
    emit(payload)


def run_entry_source(argv):
    parsed = parse_args(argv)
    entry, request_url = get_entry_from_args(parsed)
    payload = envelope("entry source", request_url)
    payload["summary"] = {
        "registerNumber": entry.get("registerNumber"),
        "name": get(entry, "lobbyistIdentity", "name"),
        "version": get(entry, "registerEntryDetails", "version"),
        "sourceDate": entry.get("sourceDate"),
    }
    payload["sources"] = entry_sources(entry)
    payload["nextActions"] = next_actions_for_entry(entry)
    emit(payload)


def run_entry_dossier(argv):
    parsed = parse_args(argv)
    entry, request_url = get_entry_from_args(parsed)
    limit = limit_flag(parsed, 5, 20)
    payload = envelope("entry dossier", request_url)
    payload["summary"] = summarize_entry(entry)
    payload["financial"] = financial_block(entry)
    payload["regulatoryProjects"] = compact_projects(entry, limit)
    payload["statements"] = compact_statements(entry, parsed["flags"].get("grep", ""), limit)
    payload["sources"] = entry_sources(entry)
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = next_actions_for_entry(entry)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = entry
    emit(payload)


def run_financial_summary(argv):
    parsed = parse_args(argv)
    entry, request_url = get_entry_from_args(parsed)
    payload = envelope("financial summary", request_url)
    payload["summary"] = {
        "registerNumber": entry.get("registerNumber"),
        "name": get(entry, "lobbyistIdentity", "name"),
        "sourceDate": entry.get("sourceDate"),
    }
    payload["financial"] = financial_block(entry)
    payload["sources"] = entry_sources(entry)
    payload["warnings"] = standard_warnings() + ["Financial ranges are register disclosures, not audited findings by this tool."]
    payload["nextActions"] = next_actions_for_entry(entry)
    emit(payload)


def run_statements_list(argv):
    parsed = parse_args(argv)
    entry, request_url = get_entry_from_args(parsed)
    limit = limit_flag(parsed, 10, 50)
    payload = envelope("statements list", request_url)
    payload["summary"] = {
        "registerNumber": entry.get("registerNumber"),
        "name": get(entry, "lobbyistIdentity", "name"),
        "statementsPresent": get(entry, "statements", "statementsPresent"),
        "statementsCount": get(entry, "statements", "statementsCount"),
        "limitApplied": limit,
    }
    payload["items"] = compact_statements(entry, parsed["flags"].get("grep", ""), limit)
    payload["sources"] = entry_sources(entry)
    payload["warnings"] = standard_warnings() + ["Statement text may include copyrighted material; quote only short excerpts."]
    payload["nextActions"] = next_actions_for_entry(entry)
    emit(payload)


def run_v1_search(argv):
    parsed = parse_args(argv)
    params = dict(parsed["params"])
    for key, value in parsed["flags"].items():
        if key not in {"include-raw", "timeout"}:
            params[key] = value
    url = LEGACY_V1_URL + "?" + urllib.parse.urlencode(params)
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=int(parsed["flags"].get("timeout", "60"))) as resp:
        body = resp.read().decode("utf-8", "replace")
    print(body)


def get_entry_from_args(parsed):
    key = require_key(parsed)
    register_number = first_non_empty(
        parsed["flags"].get("register-number"),
        parsed["flags"].get("registerNumber"),
        parsed["flags"].get("id"),
    )
    if not register_number and parsed["flags"].get("name"):
        first, _ = search_first(parsed["flags"]["name"], key)
        register_number = first.get("registerNumber")
    if not register_number:
        raise CLIError(2, "missing_register_number", "requires --register-number or --name")
    if not re.match(r"^R[0-9]{6}$", register_number):
        raise CLIError(2, "invalid_register_number", "register number must look like R001255")
    path = "/registerentries/" + urllib.parse.quote(register_number)
    if parsed["flags"].get("version"):
        path += "/" + urllib.parse.quote(parsed["flags"]["version"])
    return api_json(path, {"format": "json"}, key)


def search_first(term, key):
    data, request_url = api_json("/registerentries", {"format": "json", "q": term}, key)
    results = data.get("results", [])
    if not results:
        raise CLIError(1, "not_found", "no register entry found for name: " + term)
    return results[0], request_url


def api_json(path, params, key):
    query = urllib.parse.urlencode(params)
    request_url = f"{BASE_URL}{path}?{query}"
    req = urllib.request.Request(
        request_url,
        headers={"Authorization": "ApiKey " + key, "Accept": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            body = resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", "replace")
        raise CLIError(1, "http_error", f"HTTP {exc.code} from Lobbyregister API: {truncate(body, 300)}")
    return json.loads(body), sanitize_url(request_url)


def summarize_entry(entry):
    return {
        "registerNumber": entry.get("registerNumber"),
        "name": get(entry, "lobbyistIdentity", "name"),
        "identity": get(entry, "lobbyistIdentity", "identity"),
        "legalForm": first_non_empty(get(entry, "lobbyistIdentity", "legalForm", "de"), get(entry, "lobbyistIdentity", "legalForm", "en")),
        "activeLobbyist": get(entry, "accountDetails", "activeLobbyist"),
        "firstPublicationDate": get(entry, "accountDetails", "firstPublicationDate"),
        "lastUpdateDate": get(entry, "accountDetails", "lastUpdateDate"),
        "version": get(entry, "registerEntryDetails", "version"),
        "detailsPageUrl": get(entry, "registerEntryDetails", "detailsPageUrl"),
        "pdfUrl": get(entry, "registerEntryDetails", "pdfUrl"),
        "financialExpensesEuro": get(entry, "financialExpenses", "financialExpensesEuro"),
        "financialFiscalYear": fiscal_year(entry, "financialExpenses"),
        "employeeFTE": get(entry, "employeesInvolvedInLobbying", "employeeFTE"),
        "fieldsOfInterest": labels_from_array(get(entry, "activitiesAndInterests", "fieldsOfInterest") or [], 10),
        "activityDescriptionHint": truncate(get(entry, "activitiesAndInterests", "activityDescription") or "", 280),
        "mainFundingSources": labels_from_array(get(entry, "mainFundingSources", "mainFundingSources") or [], 8),
        "totalDonationsEuro": get(entry, "donators", "totalDonationsEuro"),
        "totalMembershipFees": get(entry, "membershipFees", "totalMembershipFees"),
        "publicAllowancesPresent": get(entry, "publicAllowances", "publicAllowancesPresent"),
        "regulatoryProjectsCount": get(entry, "regulatoryProjects", "regulatoryProjectsCount"),
        "statementsCount": get(entry, "statements", "statementsCount"),
        "contractsCount": get(entry, "contracts", "contractsCount"),
    }


def financial_block(entry):
    return {
        "financialExpenses": {
            "fiscalYear": fiscal_year(entry, "financialExpenses"),
            "rangeEuro": get(entry, "financialExpenses", "financialExpensesEuro"),
        },
        "mainFundingSources": labels_from_array(get(entry, "mainFundingSources", "mainFundingSources") or [], 20),
        "publicAllowances": get(entry, "publicAllowances"),
        "donations": {
            "fiscalYear": fiscal_year(entry, "donators"),
            "totalEuro": get(entry, "donators", "totalDonationsEuro"),
            "items": compact_named_items(get(entry, "donators", "donators") or [], 20),
        },
        "membershipFees": {
            "fiscalYear": fiscal_year(entry, "membershipFees"),
            "totalEuro": get(entry, "membershipFees", "totalMembershipFees"),
            "individualContributors": compact_named_items(get(entry, "membershipFees", "individualContributors") or [], 20),
        },
        "annualReport": {
            "exists": get(entry, "annualReports", "annualReportLastFiscalYearExists"),
            "pdfUrl": get(entry, "annualReports", "annualReportPdfUrl"),
        },
    }


def compact_projects(entry, limit):
    out = []
    for project in (get(entry, "regulatoryProjects", "regulatoryProjects") or [])[:limit]:
        out.append({
            "number": project.get("regulatoryProjectNumber"),
            "title": project.get("title"),
            "descriptionHint": truncate(project.get("description") or "", 320),
            "affectedLaws": labels_from_array(project.get("affectedLaws") or [], 8),
            "fieldsOfInterest": labels_from_array(project.get("fieldsOfInterest") or [], 8),
            "projectUrl": project.get("projectUrl"),
        })
    return out


def compact_statements(entry, grep, limit):
    out = []
    for statement in get(entry, "statements", "statements") or []:
        if len(out) >= limit:
            break
        text = get(statement, "text", "text") or ""
        item = {
            "regulatoryProjectNumber": statement.get("regulatoryProjectNumber"),
            "regulatoryProjectTitle": statement.get("regulatoryProjectTitle"),
            "pdfUrl": statement.get("pdfUrl"),
            "pdfPageCount": statement.get("pdfPageCount"),
            "recipientGroups": statement.get("recipientGroups"),
            "textPreview": truncate(text, 420),
        }
        if grep:
            hits = snippets(text, grep, 3)
            if not hits:
                continue
            item["snippets"] = hits
        out.append(item)
    return out


def entry_sources(entry):
    sources = default_sources()
    add_source(sources, "Public detail page", get(entry, "registerEntryDetails", "detailsPageUrl"), "public-page")
    add_source(sources, "Public PDF export", get(entry, "registerEntryDetails", "pdfUrl"), "pdf")
    add_source(sources, "Annual report PDF", get(entry, "annualReports", "annualReportPdfUrl"), "pdf")
    for statement in get(entry, "statements", "statements") or []:
        add_source(sources, "Statement PDF: " + (statement.get("regulatoryProjectTitle") or ""), statement.get("pdfUrl"), "statement-pdf")
    return sources


def add_source(sources, title, url, kind):
    if url:
        sources.append({"title": title, "url": url, "kind": kind})


def default_sources():
    return [
        {"title": "Bundestag Lobbyregister", "url": PUBLIC_URL, "kind": "official-register"},
        {"title": "Open Data/API page", "url": f"{PUBLIC_URL}/informationen-und-hilfe/open-data-1049716", "kind": "terms"},
        {"title": "Swagger UI V2", "url": f"{BASE_URL}/swagger-ui/", "kind": "api-docs"},
        {"title": "OpenAPI YAML V2", "url": f"{BASE_URL}/R2.21-de.yaml", "kind": "openapi"},
    ]


def standard_warnings():
    return [
        "V2 API calls require an API key; this tool redacts keys from normalized output.",
        "Register disclosures describe published self-reported register data; corroborate contentious claims with additional official sources.",
        "Use small limits for broad searches; the upstream search endpoint returns full-detail records.",
    ]


def next_actions_for_entry(entry):
    rn = entry.get("registerNumber")
    if not rn:
        return []
    return [
        f"bundestag-lobbyregister entry source --register-number {rn}",
        f"bundestag-lobbyregister financial summary --register-number {rn}",
        f"bundestag-lobbyregister statements list --register-number {rn} --grep <term>",
    ]


def search_next_actions(items):
    return [f"bundestag-lobbyregister entry dossier --register-number {item['registerNumber']}" for item in items if item.get("registerNumber")][:5]


def parse_args(argv):
    out = {"flags": {}, "params": {}, "positionals": []}
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg == "--param" and i + 1 < len(argv):
            key, value = split_key_value(argv[i + 1])
            if key:
                out["params"][key] = value
            i += 2
            continue
        if arg.startswith("--param="):
            key, value = split_key_value(arg[len("--param="):])
            if key:
                out["params"][key] = value
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


def split_key_value(raw):
    if "=" not in raw:
        return "", ""
    return raw.split("=", 1)


def require_key(parsed):
    key = api_key(parsed)
    if not key:
        raise CLIError(2, "missing_api_key", "set LOBBYREGISTER_API_KEY or pass --apikey")
    return key


def api_key(parsed):
    return parsed["flags"].get("apikey") or os.environ.get("LOBBYREGISTER_API_KEY", "")


def key_source(parsed):
    if parsed["flags"].get("apikey"):
        return "flag:redacted"
    if os.environ.get("LOBBYREGISTER_API_KEY"):
        return "env:LOBBYREGISTER_API_KEY"
    return "missing"


def envelope(command, request_url):
    return {
        "status": "ok",
        "tool": APP_NAME,
        "command": command,
        "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "request": {
            "method": "GET",
            "url": sanitize_url(request_url),
            "authConfigured": True,
            "redactedHeaders": ["Authorization"],
            "redactedQueryKeys": ["apikey"],
        },
    }


def emit(payload):
    print(json.dumps(payload, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({
        "status": "error",
        "tool": APP_NAME,
        "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "error": {"code": code, "message": redact(message)},
    })
    sys.exit(exit_code)


def get(obj, *path):
    cur = obj
    for key in path:
        if not isinstance(cur, dict):
            return None
        cur = cur.get(key)
    return cur


def labels_from_array(items, limit):
    labels = []
    for item in items[:limit]:
        if isinstance(item, dict):
            label = first_non_empty(item.get("de"), item.get("title"), item.get("name"), item.get("en"), item.get("code"))
            if label:
                labels.append(label)
    return labels


def compact_named_items(items, limit):
    out = []
    for item in items[:limit]:
        if isinstance(item, dict):
            out.append({
                "name": first_non_empty(item.get("name"), item.get("lastName")),
                "rawHint": truncate(json.dumps(item, ensure_ascii=False), 240),
            })
    return out


def fiscal_year(entry, block):
    return {
        "finished": get(entry, block, "relatedFiscalYearFinished"),
        "start": get(entry, block, "relatedFiscalYearStart"),
        "end": get(entry, block, "relatedFiscalYearEnd"),
    }


def snippets(text, term, limit):
    hits = []
    lower = text.lower()
    needle = term.lower()
    start = 0
    while len(hits) < limit:
        idx = lower.find(needle, start)
        if idx < 0:
            break
        left = max(0, idx - 160)
        right = min(len(text), idx + len(term) + 160)
        hits.append(collapse_space(text[left:right]))
        start = idx + len(term)
    return hits


def truncate(text, limit):
    text = collapse_space(str(text))
    return text if len(text) <= limit else text[: limit - 3] + "..."


def collapse_space(text):
    return " ".join(str(text).split())


def limit_flag(parsed, default, maximum):
    try:
        value = int(parsed["flags"].get("limit", default))
    except ValueError:
        return default
    return max(1, min(value, maximum))


def flag_bool(parsed, name):
    return str(parsed["flags"].get(name, "")).lower() in {"1", "true", "yes"}


def first_non_empty(*values):
    for value in values:
        if value is not None and str(value).strip():
            return str(value)
    return ""


def sanitize_url(raw):
    parsed = urllib.parse.urlparse(raw)
    query = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
    redacted = [(key, "REDACTED" if key.lower() == "apikey" else value) for key, value in query]
    return urllib.parse.urlunparse(parsed._replace(query=urllib.parse.urlencode(redacted)))


def redact(text):
    text = re.sub(r"(?i)(apikey=)[^&\s]+", r"\1REDACTED", str(text))
    text = re.sub(r"(?i)ApiKey\s+[A-Za-z0-9._-]+", "ApiKey REDACTED", text)
    text = re.sub(r"(?i)(--apikey\s+)[A-Za-z0-9._-]+", r"\1REDACTED", text)
    return text


def is_help(arg):
    return arg in {"-h", "--help", "help"}


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
