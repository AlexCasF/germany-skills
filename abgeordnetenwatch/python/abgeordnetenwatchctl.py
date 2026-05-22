#!/usr/bin/env python3
import html
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "abgeordnetenwatchctl"
BASE_URL = "https://www.abgeordnetenwatch.de/api/v2"
ROOT_URL = "https://www.abgeordnetenwatch.de"

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")

LEGACY_ENTITIES = {
    "parliaments": "Parliaments",
    "parliament-periods": "Parliament periods, legislatures, and elections",
    "politicians": "Politicians and candidate/person profile data",
    "candidacies-mandates": "Candidacies and mandates",
    "polls": "Named votes / poll metadata",
    "sidejobs": "Side jobs and disclosed outside income",
    "sidejob-organizations": "Organizations connected to side jobs",
    "votes": "Individual vote records",
    "parties": "Parties",
    "committees": "Committees",
    "committee-memberships": "Committee memberships",
    "fractions": "Parliamentary groups/fractions",
    "electoral-lists": "Electoral lists",
    "constituencies": "Constituencies",
    "election-programs": "Election programs",
    "topics": "Topics",
    "cities": "Cities used in side-job data",
    "countries": "Countries used in side-job data",
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
            run_doctor()
        elif argv[:2] == ["politicians", "search"]:
            run_politician_search(argv[2:])
        elif argv[:2] == ["politicians", "page"]:
            run_politician_page(argv[2:])
        elif argv[:2] == ["politicians", "source"]:
            run_politician_source(argv[2:])
        elif argv[:2] == ["politicians", "dossier"]:
            run_politician_dossier(argv[2:])
        elif argv[:2] == ["mandates", "for-politician"]:
            run_mandates_for_politician(argv[2:])
        elif argv[:2] == ["sidejobs", "for-politician"]:
            run_sidejobs_for_politician(argv[2:])
        elif argv[0] == "page":
            run_politician_page(argv[1:])
        elif argv[0] == "source":
            run_politician_source(argv[1:])
        else:
            run_legacy(argv)
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    return 0


def print_root_help():
    print("""abgeordnetenwatchctl -- abgeordnetenwatch.de public transparency data

Purpose
  Search and cite public politician, mandate, voting, profile, and side-job
  data from abgeordnetenwatch.de.

Fast paths
  abgeordnetenwatchctl doctor
  abgeordnetenwatchctl politicians search --name "Alice Weidel" --limit 3
  abgeordnetenwatchctl politicians dossier --name "Alice Weidel" --grep Nebentätigkeiten

Legacy endpoint commands
  <entity> list|get

Research commands
  doctor
  politicians search
  politicians page
  politicians source
  politicians dossier
  mandates for-politician
  sidejobs for-politician
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "politicians dossier":
        print("""abgeordnetenwatchctl politicians dossier

Builds a compact evidence bundle for one politician with API profile data,
mandates, side jobs, source URLs, page metadata, optional profile-page snippets,
warnings, and next actions.

Examples
  abgeordnetenwatchctl politicians dossier --name "Alice Weidel" --grep Nebentätigkeiten
  abgeordnetenwatchctl politicians dossier --id 108379 --limit 5
""")
    elif joined == "politicians page":
        print("""abgeordnetenwatchctl politicians page

Fetches a public profile page and extracts canonical URL, title, description,
profile ID hints, text preview, and grep snippets.
""")
    elif joined == "politicians search":
        print("""abgeordnetenwatchctl politicians search

Searches politicians by name with a small default limit and normalized source URLs.
""")
    else:
        print_root_help()


def run_doctor():
    data = api_json("/politicians", {"range_end": "1"})
    meta = data.get("meta", {})
    api_info = meta.get("abgeordnetenwatch_api", {})
    result = meta.get("result", {})
    payload = envelope("doctor", f"{BASE_URL}/politicians?range_end=1")
    payload["summary"] = {
        "authRequired": False,
        "baseUrl": BASE_URL,
        "apiVersion": api_info.get("version"),
        "licence": api_info.get("licence"),
        "licenceLink": api_info.get("licence_link"),
        "documentation": [
            "https://www.abgeordnetenwatch.de/api",
            "https://www.abgeordnetenwatch.de/api/response",
            "https://www.abgeordnetenwatch.de/api/version-changelog/aktuell",
        ],
        "publishedRateLimit": "not found in official API docs",
        "resultLimit": "default 100; range_end/pager_limit up to 1000 per official docs",
        "health": {
            "status": meta.get("status"),
            "count": result.get("count"),
            "sampleTotal": result.get("total"),
        },
    }
    payload["sources"] = default_sources()
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = [
        'abgeordnetenwatchctl politicians search --name "Alice Weidel" --limit 3',
        "abgeordnetenwatchctl politicians dossier --id 108379 --grep Nebentätigkeiten",
    ]
    emit(payload)


def run_legacy(argv):
    if len(argv) < 2:
        raise CLIError(2, "unknown_command", "expected <entity> list|get")
    entity, action = argv[0], argv[1]
    if entity not in LEGACY_ENTITIES:
        raise CLIError(2, "unknown_entity", "unknown entity: " + entity)
    parsed = parse_args(argv[2:])
    params = normalize_params(parsed)
    if action == "list":
        print(api_get("/" + entity, params)["body"])
    elif action == "get":
        ident = parsed["flags"].get("id") or (parsed["positionals"][0] if parsed["positionals"] else "")
        if not ident:
            raise CLIError(2, "missing_id", entity + " get requires --id")
        print(api_get("/" + entity + "/" + urllib.parse.quote(str(ident), safe=""), params)["body"])
    else:
        raise CLIError(2, "unknown_action", f"unknown action for {entity}: {action}")


def run_politician_search(argv):
    parsed = parse_args(argv)
    params = normalize_params(parsed)
    limit = limit_flag(parsed, 5, 50)
    params["range_end"] = str(limit)
    flags = parsed["flags"]
    if flags.get("name"):
        params["label[cn]"] = flags["name"]
    if flags.get("first-name"):
        params["first_name[cn]"] = flags["first-name"]
    if flags.get("last-name"):
        params["last_name[cn]"] = flags["last-name"]
    if flags.get("party"):
        params["party[entity.label][cn]"] = flags["party"]
    data = api_json("/politicians", params)
    items = summarize_records(data_list(data), limit)
    payload = envelope("politicians search", BASE_URL + "/politicians?" + urllib.parse.urlencode(params))
    payload["summary"] = {"search": search_summary(parsed), "returned": len(items), "total": total(data), "clientLimit": limit}
    payload["items"] = items
    payload["sources"] = [{"kind": "api", "title": "Politicians endpoint", "url": BASE_URL + "/politicians"}]
    payload["warnings"] = ["Search results are public transparency data; verify official parliamentary records separately when needed."]
    payload["nextActions"] = next_for_politician_items(items)
    emit(payload)


def run_politician_source(argv):
    record, _ = resolve_politician(argv)
    payload = envelope("politicians source", api_url_from_record(record))
    payload["summary"] = {"record": summarize_politician(record), "sources": politician_sources(record)}
    payload["sources"] = politician_sources(record)
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = [
        f"abgeordnetenwatchctl politicians page --id {record.get('id')}",
        f"abgeordnetenwatchctl politicians dossier --id {record.get('id')}",
    ]
    emit(payload)


def run_politician_page(argv):
    parsed = parse_args(argv)
    record, raw_url = resolve_politician(argv)
    profile_url = raw_url or record.get("abgeordnetenwatch_url")
    if not profile_url:
        raise CLIError(1, "missing_profile_url", "politician record has no profile URL")
    page = fetch_page(profile_url, parsed["flags"].get("grep", ""))
    payload = envelope("politicians page", page["url"])
    payload["summary"] = page
    payload["sources"] = [{"kind": "profile", "title": "Public profile page", "url": page["url"]}]
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = [f"abgeordnetenwatchctl politicians dossier --id {record.get('id')}"]
    emit(payload)


def run_politician_dossier(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 10, 50)
    record, _ = resolve_politician(argv)
    ident = str(record.get("id"))
    mandates = fetch_collection("/candidacies-mandates", {"politician": ident, "range_end": str(limit)}, limit)
    sidejobs = sidejobs_for_mandates(mandates, limit)
    page = None
    if record.get("abgeordnetenwatch_url"):
        try:
            page = fetch_page(record["abgeordnetenwatch_url"], parsed["flags"].get("grep", ""))
        except CLIError:
            page = None
    payload = envelope("politicians dossier", api_url_from_record(record))
    payload["summary"] = {
        "politician": summarize_politician(record),
        "mandateCount": len(mandates),
        "mandates": summarize_records(mandates, limit),
        "sidejobCount": len(sidejobs),
        "sidejobs": summarize_records(sidejobs, limit),
        "sidejobIncomeSum": sum_numeric(sidejobs, "income"),
        "profilePage": page,
        "sourceCategories": ["api", "public-profile-page", "mandates", "sidejobs"],
    }
    payload["sources"] = politician_sources(record)
    payload["warnings"] = standard_warnings() + [
        "Side-job income fields may be partial and depend on disclosed Bundestag data as processed by abgeordnetenwatch.",
        "Do not equate outside income or mandates with corruption without independent evidence.",
    ]
    payload["nextActions"] = [
        f"abgeordnetenwatchctl sidejobs for-politician --id {ident} --limit {limit}",
        f"abgeordnetenwatchctl politicians page --id {ident} --grep Nebentätigkeiten",
    ]
    emit(payload)


def run_mandates_for_politician(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 10, 50)
    record, _ = resolve_politician(argv)
    ident = str(record.get("id"))
    mandates = fetch_collection("/candidacies-mandates", {"politician": ident, "range_end": str(limit)}, limit)
    payload = envelope("mandates for-politician", f"{BASE_URL}/candidacies-mandates?politician={urllib.parse.quote(ident)}")
    payload["summary"] = {"politician": summarize_politician(record), "returned": len(mandates)}
    payload["items"] = summarize_records(mandates, limit)
    payload["sources"] = [{"kind": "api", "title": "Candidacies/mandates endpoint", "url": BASE_URL + "/candidacies-mandates"}]
    payload["warnings"] = standard_warnings()
    payload["nextActions"] = [f"abgeordnetenwatchctl sidejobs for-politician --id {ident}"]
    emit(payload)


def run_sidejobs_for_politician(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 10, 50)
    record, _ = resolve_politician(argv)
    ident = str(record.get("id"))
    mandates = fetch_collection("/candidacies-mandates", {"politician": ident, "range_end": str(limit)}, limit)
    sidejobs = sidejobs_for_mandates(mandates, limit)
    payload = envelope("sidejobs for-politician", BASE_URL + "/sidejobs")
    payload["summary"] = {
        "politician": summarize_politician(record),
        "mandates": len(mandates),
        "returned": len(sidejobs),
        "incomeSum": sum_numeric(sidejobs, "income"),
        "clientLimit": limit,
    }
    payload["items"] = summarize_records(sidejobs, limit)
    payload["sources"] = [{"kind": "api", "title": "Sidejobs endpoint", "url": BASE_URL + "/sidejobs"}]
    payload["warnings"] = standard_warnings() + ["Side-job data is disclosure data; interpret categories and income fields cautiously."]
    payload["nextActions"] = [f"abgeordnetenwatchctl politicians dossier --id {ident} --grep Nebentätigkeiten"]
    emit(payload)


def resolve_politician(argv):
    parsed = parse_args(argv)
    flags = parsed["flags"]
    if flags.get("url"):
        raw_url = flags["url"]
        ident = id_from_profile_url(raw_url)
        if not ident:
            try:
                page = fetch_page(raw_url, "")
                ident = page.get("politicianId")
            except CLIError:
                ident = ""
        if not ident:
            raise CLIError(2, "unsupported_profile_url", "could not infer politician ID from URL; use --id or --name")
        return get_politician(ident), raw_url
    if flags.get("id"):
        return get_politician(flags["id"]), ""
    if flags.get("name"):
        data = api_json("/politicians", {"label[cn]": flags["name"], "range_end": "1"})
        rows = data_list(data)
        if not rows:
            raise CLIError(1, "not_found", "no politician found for name: " + flags["name"])
        return get_politician(str(rows[0]["id"])), ""
    raise CLIError(2, "missing_input", "provide --id, --name, or --url")


def get_politician(ident):
    data = api_json("/politicians/" + urllib.parse.quote(str(ident), safe=""))
    record = data.get("data")
    if not isinstance(record, dict):
        raise CLIError(1, "not_found", "politician not found: " + str(ident))
    return record


def fetch_collection(path, params, limit):
    if "range_end" not in params and "pager_limit" not in params:
        params["range_end"] = str(limit)
    data = api_json(path, params)
    return data_list(data)[:limit]


def sidejobs_for_mandates(mandates, limit):
    out = []
    seen = set()
    for mandate in mandates:
        if len(out) >= limit:
            break
        ident = str(mandate.get("id", ""))
        if not ident:
            continue
        try:
            rows = fetch_collection("/sidejobs", {"mandates": ident, "range_end": str(limit)}, limit)
        except CLIError:
            continue
        for row in rows:
            rid = str(row.get("id", ""))
            if rid in seen:
                continue
            seen.add(rid)
            out.append(row)
            if len(out) >= limit:
                break
    return out


def fetch_page(raw_url, grep):
    valid_url = validate_aw_url(raw_url)
    resp = http_get(valid_url, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    body = resp["body"]
    text = strip_html(body)
    page = {
        "url": resp["url"],
        "title": first_match(body, r"(?is)<title[^>]*>(.*?)</title>"),
        "canonical": attr_match(body, "link", "rel", "canonical", "href"),
        "shortlink": attr_match(body, "link", "rel", "shortlink", "href"),
        "description": meta_content(body, "description"),
        "politicianId": politician_id_from_html(body),
        "textLength": len(text),
        "textPreview": text[:1200],
    }
    if grep:
        page["grep"] = grep
        page["snippets"] = snippets(text, grep, 10)
    return page


def api_json(path, params=None):
    body = api_get(path, params or {})["body"]
    try:
        return json.loads(body)
    except json.JSONDecodeError as exc:
        raise CLIError(1, "invalid_json", "API did not return JSON: " + str(exc))


def api_get(path, params=None):
    query = ("?" + urllib.parse.urlencode(params or {})) if params else ""
    return http_get(BASE_URL + path + query, "application/json")


def http_get(raw_url, accept):
    request = urllib.request.Request(raw_url, headers={"Accept": accept, "User-Agent": APP_NAME + "/2.0 (+https://github.com/AlexCasF/democracy-researcher)"})
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return {"url": response.geturl(), "status": response.status, "contentType": response.headers.get("content-type", ""), "body": response.read(8 * 1024 * 1024).decode("utf-8", errors="replace")}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")[:500]
        raise CLIError(1, "request_failed", f"HTTP {exc.code}: {detail}")
    except urllib.error.URLError as exc:
        raise CLIError(1, "request_failed", str(exc.reason))


def parse_args(argv):
    flags = {}
    params = {}
    positionals = []
    i = 0
    while i < len(argv):
        token = argv[i]
        if token in ("--param", "--query") and i + 1 < len(argv):
            if "=" in argv[i + 1]:
                key, value = argv[i + 1].split("=", 1)
                params[key] = value
            i += 2
            continue
        if token.startswith("--"):
            key = token[2:]
            if i + 1 < len(argv) and not argv[i + 1].startswith("--"):
                flags[key] = argv[i + 1]
                i += 2
            else:
                flags[key] = "true"
                i += 1
            continue
        positionals.append(token)
        i += 1
    return {"flags": flags, "params": params, "positionals": positionals}


def normalize_params(parsed):
    params = dict(parsed["params"])
    flags = parsed["flags"]
    if flags.get("limit") and not params.get("range_end") and not params.get("pager_limit"):
        params["range_end"] = flags["limit"]
    if flags.get("page"):
        params["page"] = flags["page"]
    if flags.get("pager-limit"):
        params["pager_limit"] = flags["pager-limit"]
    if flags.get("related-data"):
        params["related_data"] = flags["related-data"]
    return params


def limit_flag(parsed, default, maximum):
    raw = parsed["flags"].get("limit", "")
    try:
        value = int(raw)
    except ValueError:
        return default
    if value < 1:
        return default
    return min(value, maximum)


def summarize_records(rows, limit):
    return [summarize_record(row) for row in rows[:limit]]


def summarize_record(row):
    if row.get("entity_type") == "politician":
        return summarize_politician(row)
    out = {}
    for key in ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "type", "start_date", "end_date", "income", "income_level", "income_total", "interval", "data_change_date", "job_title_extra", "additional_information"]:
        if key in row:
            out[key] = row[key]
    for key in ["sidejob_organization", "party", "parliament_period", "politician"]:
        if isinstance(row.get(key), dict):
            out[key] = summarize_reference(row[key])
    if row.get("api_url"):
        out["sources"] = [{"kind": "api", "title": "API record", "url": row["api_url"]}]
    return out


def summarize_politician(row):
    out = {}
    for key in ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "first_name", "last_name", "sex", "year_of_birth", "education", "occupation", "statistic_questions", "statistic_questions_answered", "ext_id_bundestagsverwaltung", "qid_wikidata"]:
        if key in row:
            out[key] = row[key]
    if isinstance(row.get("party"), dict):
        out["party"] = summarize_reference(row["party"])
    out["sources"] = politician_sources(row)
    return out


def summarize_reference(row):
    return {key: row[key] for key in ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url"] if key in row}


def politician_sources(row):
    sources = []
    if row.get("api_url"):
        sources.append({"kind": "api", "title": "API record", "url": row["api_url"]})
    if row.get("abgeordnetenwatch_url"):
        sources.append({"kind": "profile", "title": "Public profile", "url": row["abgeordnetenwatch_url"]})
    if row.get("id") is not None:
        sources.append({"kind": "api", "title": "Mandates for politician", "url": BASE_URL + "/candidacies-mandates?politician=" + urllib.parse.quote(str(row["id"]))})
    return sources


def data_list(data):
    rows = data.get("data")
    return rows if isinstance(rows, list) else []


def total(data):
    return data.get("meta", {}).get("result", {}).get("total")


def search_summary(parsed):
    return {key: parsed["flags"][key] for key in ["name", "first-name", "last-name", "party", "limit"] if key in parsed["flags"]}


def next_for_politician_items(items):
    out = []
    for item in items:
        ident = item.get("id")
        if ident is None:
            continue
        out.append(f"abgeordnetenwatchctl politicians dossier --id {ident}")
        out.append(f"abgeordnetenwatchctl politicians page --id {ident} --grep Nebentätigkeiten")
        if len(out) >= 4:
            break
    return out


def envelope(command, request_url):
    return {
        "tool": APP_NAME,
        "command": command,
        "status": "ok",
        "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "request": {"method": "GET", "url": request_url, "redactions": []},
        "summary": {},
        "sources": [],
        "warnings": [],
        "nextActions": [],
    }


def default_sources():
    return [
        {"kind": "documentation", "title": "API documentation", "url": "https://www.abgeordnetenwatch.de/api"},
        {"kind": "documentation", "title": "API response format", "url": "https://www.abgeordnetenwatch.de/api/response"},
        {"kind": "documentation", "title": "API changelog", "url": "https://www.abgeordnetenwatch.de/api/version-changelog/aktuell"},
        {"kind": "license", "title": "CC0 1.0", "url": "https://creativecommons.org/publicdomain/zero/1.0/deed.de"},
    ]


def standard_warnings():
    return [
        "abgeordnetenwatch is a transparency platform, not an official parliamentary archive.",
        "Use official Bundestag/DIP records when the exact official parliamentary record matters.",
        "No exact API rate limit was found in official docs; keep requests bounded.",
    ]


def validate_aw_url(raw):
    parsed = urllib.parse.urlparse(raw)
    if parsed.netloc not in ("www.abgeordnetenwatch.de", "abgeordnetenwatch.de"):
        raise CLIError(2, "unsupported_url", "URL must belong to abgeordnetenwatch.de")
    scheme = parsed.scheme or "https"
    netloc = "www.abgeordnetenwatch.de" if parsed.netloc == "abgeordnetenwatch.de" else parsed.netloc
    return urllib.parse.urlunparse((scheme, netloc, parsed.path, parsed.params, parsed.query, parsed.fragment))


def id_from_profile_url(raw):
    match = re.search(r"/politician/([0-9]+)", raw)
    return match.group(1) if match else ""


def politician_id_from_html(body):
    for pattern in [r'currentPath":"politician/([0-9]+)"', r"/politician/([0-9]+)", r'view_args":"([0-9]+)"']:
        match = re.search(pattern, body)
        if match:
            return match.group(1)
    return ""


def strip_html(raw):
    raw = re.sub(r"(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<svg[^>]*>.*?</svg>", " ", raw)
    raw = re.sub(r"(?s)<[^>]+>", " ", raw)
    return clean(html.unescape(raw))


def snippets(text, term, limit):
    out = []
    low = text.lower()
    needle = term.lower()
    start = 0
    while len(out) < limit:
        idx = low.find(needle, start)
        if idx < 0:
            break
        left = max(0, idx - 240)
        right = min(len(text), idx + len(term) + 240)
        out.append(clean(text[left:right]))
        start = idx + len(term)
    return out


def first_match(raw, pattern):
    match = re.search(pattern, raw)
    return clean(html.unescape(match.group(1))) if match else ""


def attr_match(raw, tag, attr_name, attr_value, wanted):
    for match in re.findall(r"(?is)<" + tag + r"[^>]*>", raw):
        low = match.lower()
        if f'{attr_name}="{attr_value}"' in low or f"{attr_name}='{attr_value}'" in low:
            value = attr_value_from_tag(match, wanted)
            if value:
                return html.unescape(value)
    return ""


def meta_content(raw, name):
    for match in re.findall(r"(?is)<meta[^>]*>", raw):
        low = match.lower()
        if f'name="{name.lower()}"' in low or f'property="{name.lower()}"' in low:
            return clean(html.unescape(attr_value_from_tag(match, "content")))
    return ""


def attr_value_from_tag(tag, attr):
    match = re.search(r"(?is)" + re.escape(attr) + r"\s*=\s*[\"']([^\"']+)[\"']", tag)
    return match.group(1) if match else ""


def api_url_from_record(row):
    return row.get("api_url") or (BASE_URL + "/politicians/" + str(row.get("id")) if row.get("id") is not None else BASE_URL)


def sum_numeric(rows, key):
    total_value = 0.0
    for row in rows:
        value = row.get(key)
        if isinstance(value, (int, float)):
            total_value += float(value)
    return total_value


def clean(value):
    return " ".join((value or "").split())


def is_help(value):
    return value in ("--help", "-h", "help")


def emit(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"tool": APP_NAME, "status": "error", "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})
    sys.exit(exit_code)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
