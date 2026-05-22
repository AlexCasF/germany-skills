#!/usr/bin/env python3
import html
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "rechtsinformationen-bund"
BASE_URL = "https://testphase.rechtsinformationen.bund.de/v1"
ROOT_URL = "https://testphase.rechtsinformationen.bund.de"

LEGACY_COMMANDS = {
    "statistics": ("/statistics", "singleton"),
    "documents list": ("/document", "list"),
    "documents search": ("/document/lucene-search", "list"),
    "documents search-administrative-directive": ("/document/lucene-search/administrative-directive", "list"),
    "documents search-case-law": ("/document/lucene-search/case-law", "list"),
    "documents search-legislation": ("/document/lucene-search/legislation", "list"),
    "documents search-literature": ("/document/lucene-search/literature", "list"),
    "administrative-directive list": ("/administrative-directive", "list"),
    "administrative-directive get": ("/administrative-directive/{documentNumber}", "document"),
    "administrative-directive html": ("/administrative-directive/{documentNumber}.html", "text"),
    "administrative-directive xml": ("/administrative-directive/{documentNumber}.xml", "text"),
    "case-law list": ("/case-law", "list"),
    "case-law courts": ("/case-law/courts", "singleton"),
    "case-law get": ("/case-law/{documentNumber}", "document"),
    "case-law html": ("/case-law/{documentNumber}.html", "text"),
    "case-law xml": ("/case-law/{documentNumber}.xml", "text"),
    "legislation list": ("/legislation", "list"),
    "legislation get": ("/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}", "legislation"),
    "legislation html": ("/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.html", "text"),
    "legislation xml": ("/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.xml", "text"),
    "legislation article-html": ("/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}/{articleEid}.html", "text"),
    "literature list": ("/literature", "list"),
    "literature get": ("/literature/{documentNumber}", "document"),
    "literature html": ("/literature/{documentNumber}.html", "text"),
    "literature xml": ("/literature/{documentNumber}.xml", "text"),
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
        elif argv[0] in ("source",) or argv[:2] == ["documents", "source"]:
            run_source(argv[1:] if argv[0] == "source" else argv[2:])
        elif argv[:2] == ["documents", "text"]:
            run_text(argv[2:])
        elif argv[:2] == ["documents", "dossier"]:
            run_dossier(argv[2:])
        elif argv[:2] == ["case-law", "dossier"]:
            run_dossier(["--type", "case-law"] + argv[2:])
        elif argv[:2] == ["legislation", "dossier"]:
            run_dossier(["--type", "legislation"] + argv[2:])
        elif argv[0] == "cite":
            run_cite(argv[1:])
        else:
            command, rest = resolve_raw(argv)
            run_raw(command, rest)
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    return 0


def print_root_help():
    print("""rechtsinformationen-bund -- official German federal legal information preview API

Purpose
  Search and cite legal information from the Rechtsinformationen des Bundes
  trial service: federal legislation, federal case law, legal literature, and
  administrative directives.

Fast paths
  rechtsinformationen-bund doctor
  rechtsinformationen-bund documents search --search-term "Suchbegriff" --limit 3
  rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision

Raw endpoint commands
  statistics
  documents list|search|search-case-law|search-legislation
  administrative-directive list|get|html|xml
  case-law list|courts|get|html|xml
  legislation list|get|html|xml|article-html
  literature list|get|html|xml

Research commands
  doctor
  source / documents source
  documents text
  documents dossier
  cite
""")


def print_help(path):
    if path == ["documents", "dossier"]:
        print("""rechtsinformationen-bund documents dossier

Builds a compact evidence bundle with metadata, source URLs, optional text
snippets, warnings, and next actions.

Examples
  rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision
  rechtsinformationen-bund documents dossier --search-term "Suchbegriff" --grep Suchbegriff
""")
    elif path == ["documents", "text"]:
        print("""rechtsinformationen-bund documents text

Fetches the best HTML/XML source rendition for a known document and extracts
plain text plus optional grep snippets.
""")
    else:
        print_root_help()


def run_doctor():
    stats = api_json("/statistics")
    emit(envelope("doctor", {
        "authRequired": False,
        "baseUrl": BASE_URL,
        "rateLimit": "600 requests per minute per client IP",
        "rateLimitExceeded": "may return HTTP 503",
        "statistics": stats,
        "trialService": True,
    }, "/statistics"))


def run_raw(command, argv):
    path_template, kind = LEGACY_COMMANDS[command]
    parsed = parse_args(argv)
    path = fill_path(path_template, parsed, command)
    params = raw_params(parsed)
    if command == "documents search" and parsed["flags"].get("search-term"):
        run_compact_search(path, params, parsed)
        return
    resp = api_get(path, params)
    if kind == "text":
        print(resp["body"].decode("utf-8", errors="replace"))
        return
    try:
        emit(json.loads(resp["body"].decode("utf-8")))
    except json.JSONDecodeError:
        print(resp["body"].decode("utf-8", errors="replace"))


def run_compact_search(path, params, parsed):
    data = api_json(path, params)
    members = member_list(data)
    limit = int(parsed["flags"].get("limit") or parsed["flags"].get("size") or len(members) or 10)
    items = [summarize_record(item) for item in members[:limit]]
    payload = envelope("documents search", {
        "searchTerm": parsed["flags"].get("search-term"),
        "returned": len(items),
        "clientLimit": limit,
        "totalItems": data.get("totalItems"),
        "nextPage": nested_get(data, ["view", "next"]) or nested_get(data, ["hydra:view", "hydra:next"]),
    }, path_with_query(path, params))
    payload["items"] = items
    emit(payload)


def run_source(argv):
    source = build_source(argv)
    emit(envelope("source", source, source["record"].get("@id") or source["record"].get("id") or "/source"))


def run_text(argv):
    parsed = parse_args(argv)
    source = build_source(argv)
    text, used_url = source_text(source)
    grep = parsed["flags"].get("grep")
    emit(envelope("documents text", {
        "record": source["record"],
        "textLength": len(text),
        "sourceUrl": used_url,
        "grep": grep,
        "snippets": snippets(text, grep) if grep else [],
        "textPreview": text[:1200],
    }, used_url))


def run_dossier(argv):
    parsed = parse_args(argv)
    flags = parsed["flags"]
    source = build_source(argv)
    summary = {
        "record": source["record"],
        "sourceCount": source["sourceCount"],
        "citation": citation(source["record"]),
    }
    if flags.get("grep"):
        text, used_url = source_text(source)
        summary["textSourceUrl"] = used_url
        summary["textLength"] = len(text)
        summary["grep"] = flags["grep"]
        summary["snippets"] = snippets(text, flags["grep"])
    emit(envelope("documents dossier", summary, source["record"].get("@id") or "/dossier"))


def run_cite(argv):
    source = build_source(argv)
    emit(envelope("cite", {
        "citation": citation(source["record"]),
        "record": source["record"],
        "sources": source["record"].get("sources", []),
    }, source["record"].get("@id") or "/cite"))


def build_source(argv):
    parsed = parse_args(argv)
    flags = parsed["flags"]
    if flags.get("search-term"):
        params = {"searchTerm": flags["search-term"], "size": "1"}
        data = api_json("/document/lucene-search", params)
        members = member_list(data)
        if not members:
            raise CLIError(1, "not_found", "search returned no records")
        rec = members[0]
        doc_type, doc_id = infer_record_identity(rec)
        rec = api_json(record_path(doc_type, doc_id))
    elif flags.get("url"):
        rec = api_json(url_to_api_path(flags["url"]))
    else:
        doc_type = flags.get("type") or infer_type(flags.get("document-number") or flags.get("eli") or flags.get("id"))
        doc_id = flags.get("document-number") or flags.get("eli") or flags.get("id")
        if not doc_type or not doc_id:
            raise CLIError(2, "missing_input", "provide --type and --document-number/--eli, --url, or --search-term")
        rec = api_json(record_path(doc_type, doc_id))
    sources = source_links(rec)
    compact = summarize_record(rec)
    compact["sources"] = sources
    return {
        "citationSource": "Rechtsinformationen des Bundes",
        "record": compact,
        "sourceCount": len(sources),
    }


def source_text(source):
    links = source["record"].get("sources") or []
    chosen = None
    for link in links:
        if link.get("kind") == "html":
            chosen = link["url"]
            break
    if not chosen:
        for link in links:
            if link.get("kind") == "xml":
                chosen = link["url"]
                break
    if not chosen:
        raise CLIError(1, "no_text_source", "no HTML or XML source URL was found")
    resp = http_get_absolute(chosen)
    content_type = resp["content_type"]
    body = resp["body"].decode("utf-8", errors="replace")
    if "html" in content_type or chosen.endswith(".html"):
        return strip_html(body), resp["url"]
    return strip_xml(body), resp["url"]


def resolve_raw(argv):
    for width in (3, 2, 1):
        key = " ".join(argv[:width])
        if key in LEGACY_COMMANDS:
            return key, argv[width:]
    raise CLIError(2, "unknown_command", "unknown command path: " + " ".join(argv))


def parse_args(argv):
    flags = {}
    params = {}
    positionals = []
    i = 0
    while i < len(argv):
        token = argv[i]
        if token in ("--param", "--query") and i + 1 < len(argv):
            add_key_value(params, argv[i + 1])
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


def raw_params(parsed):
    params = dict(parsed["params"])
    flags = parsed["flags"]
    direct = {
        "search-term": "searchTerm",
        "size": "size",
        "limit": "size",
        "page-index": "pageIndex",
        "court-type": "courtType",
        "file-number": "fileNumber",
        "decision-date": "decisionDate",
        "document-type": "documentType",
    }
    for flag, param in direct.items():
        if flag in flags:
            params[param] = flags[flag]
    return params


def fill_path(path, parsed, command):
    flags = parsed["flags"]
    positionals = list(parsed["positionals"])
    values = dict(flags)
    if "documentNumber" not in values and "document-number" in values:
        values["documentNumber"] = values["document-number"]
    placeholders = re.findall(r"{([^}]+)}", path)
    for name in placeholders:
        value = values.get(name) or values.get(camel_to_kebab(name))
        if not value and positionals:
            value = positionals.pop(0)
        if not value:
            raise CLIError(2, "missing_argument", f"{command} needs --{camel_to_kebab(name)}")
        path = path.replace("{" + name + "}", urllib.parse.quote(value, safe=""))
    return path


def api_json(path, params=None):
    resp = api_get(path, params)
    try:
        return json.loads(resp["body"].decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise CLIError(1, "invalid_json", f"API did not return JSON: {exc}")


def api_get(path, params=None):
    url = path if path.startswith("http") else BASE_URL + path_with_query(path, params or {})
    return http_get_absolute(url)


def http_get_absolute(url):
    req = urllib.request.Request(url, headers={"User-Agent": APP_NAME, "Accept": "application/json, text/html, application/xml;q=0.9, */*;q=0.8"})
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return {
                "url": resp.geturl(),
                "status": resp.status,
                "content_type": resp.headers.get("content-type", ""),
                "body": resp.read(),
            }
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")[:400]
        raise CLIError(1, "request_failed", f"API returned HTTP {exc.code}: {detail}")
    except urllib.error.URLError as exc:
        raise CLIError(1, "request_failed", str(exc.reason))


def path_with_query(path, params):
    if not params:
        return path
    return path + "?" + urllib.parse.urlencode(params, doseq=True)


def url_to_api_path(raw):
    parsed = urllib.parse.urlparse(raw)
    if not parsed.netloc:
        return raw
    if parsed.netloc != "testphase.rechtsinformationen.bund.de":
        raise CLIError(2, "unsupported_url", "URL must belong to testphase.rechtsinformationen.bund.de")
    path = parsed.path
    if path.startswith("/v1/"):
        path = path[3:]
    return path + (("?" + parsed.query) if parsed.query else "")


def record_path(doc_type, doc_id):
    if doc_type == "case-law":
        return "/case-law/" + urllib.parse.quote(doc_id, safe="")
    if doc_type == "literature":
        return "/literature/" + urllib.parse.quote(doc_id, safe="")
    if doc_type == "administrative-directive":
        return "/administrative-directive/" + urllib.parse.quote(doc_id, safe="")
    if doc_type == "legislation":
        if doc_id.startswith("eli/"):
            return "/legislation/" + doc_id
        return "/legislation/eli/" + doc_id
    raise CLIError(2, "unsupported_type", "unsupported --type: " + str(doc_type))


def source_links(record):
    links = []
    rid = record.get("@id") or record.get("id")
    if rid:
        links.append({"kind": "api", "title": "@id", "url": ROOT_URL + rid})
    for enc in record.get("encoding") or []:
        if not isinstance(enc, dict):
            continue
        url = enc.get("contentUrl")
        if not url:
            continue
        kind = "source"
        fmt = enc.get("encodingFormat", "")
        if "html" in fmt or url.endswith(".html"):
            kind = "html"
        elif "xml" in fmt or url.endswith(".xml"):
            kind = "xml"
        elif "zip" in fmt or url.endswith(".zip"):
            kind = "zip"
        links.append({"kind": kind, "title": fmt or kind, "url": ROOT_URL + url if url.startswith("/") else url})
    return links


def summarize_record(record):
    if record.get("@type") == "SearchResult" and isinstance(record.get("item"), dict):
        out = summarize_record(record["item"])
        matches = record.get("textMatches") or []
        out["textMatchCount"] = len(matches)
        if matches:
            out["firstTextMatch"] = matches[0]
        out["sources"] = source_links(record["item"])
        return out

    out = {}
    for key in [
        "@id", "@type", "documentNumber", "ecli", "eli", "legislationIdentifier",
        "headline", "name", "abbreviation", "decisionDate", "courtName",
        "courtType", "documentType", "inLanguage"
    ]:
        if key in record:
            out[key] = record[key]
    if not out:
        for key, value in list(record.items())[:12]:
            if isinstance(value, (str, int, float, bool)) or value is None:
                out[key] = value
    return out


def infer_record_identity(record):
    if record.get("@type") == "SearchResult" and isinstance(record.get("item"), dict):
        return infer_record_identity(record["item"])

    if record.get("documentNumber"):
        rid = record["documentNumber"]
        return infer_type(rid), rid
    if record.get("legislationIdentifier"):
        return "legislation", record["legislationIdentifier"]
    if record.get("eli"):
        return "legislation", record["eli"]
    rid = record.get("@id") or ""
    if "/case-law/" in rid:
        return "case-law", rid.rsplit("/", 1)[-1]
    if "/legislation/" in rid:
        return "legislation", rid.split("/legislation/", 1)[1]
    raise CLIError(1, "unrecognized_record", "could not infer document identity from search result")


def infer_type(identifier):
    if not identifier:
        return ""
    if identifier.startswith("eli/"):
        return "legislation"
    if identifier.upper().startswith("K"):
        return "case-law"
    return ""


def citation(record):
    if record.get("ecli"):
        return ", ".join(filter(None, [record.get("courtName"), record.get("decisionDate"), record.get("headline"), record.get("ecli")]))
    if record.get("legislationIdentifier") or record.get("eli"):
        return ", ".join(filter(None, [record.get("name"), record.get("abbreviation"), record.get("legislationIdentifier") or record.get("eli")]))
    return record.get("headline") or record.get("name") or record.get("documentNumber") or "Rechtsinformationen des Bundes"


def snippets(text, term):
    if not term:
        return []
    low = text.lower()
    needle = term.lower()
    result = []
    start = 0
    while len(result) < 10:
        idx = low.find(needle, start)
        if idx < 0:
            break
        left = max(0, idx - 220)
        right = min(len(text), idx + len(term) + 220)
        result.append(clean(text[left:right]))
        start = idx + len(term)
    return result


def strip_html(raw):
    raw = re.sub(r"(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>", " ", raw)
    raw = re.sub(r"(?s)<[^>]+>", " ", raw)
    return clean(html.unescape(raw))


def strip_xml(raw):
    raw = re.sub(r"(?s)<[^>]+>", " ", raw)
    return clean(html.unescape(raw))


def clean(value):
    return re.sub(r"\s+", " ", value or "").strip()


def member_list(data):
    value = data.get("member") or data.get("hydra:member") or []
    return value if isinstance(value, list) else []


def nested_get(data, path):
    cur = data
    for part in path:
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def add_key_value(params, raw):
    if "=" not in raw:
        raise CLIError(2, "bad_param", "--param expects key=value")
    key, value = raw.split("=", 1)
    params[key] = value


def camel_to_kebab(value):
    return re.sub(r"([a-z0-9])([A-Z])", r"\1-\2", value).lower()


def envelope(command, summary, path_or_url):
    request_url = path_or_url if str(path_or_url).startswith("http") else BASE_URL + str(path_or_url)
    return {
        "tool": APP_NAME,
        "command": command,
        "status": "ok",
        "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "request": {"method": "GET", "url": request_url, "redactions": []},
        "summary": summary,
        "sources": [
            {"kind": "portal", "title": "Portal", "url": ROOT_URL + "/"},
            {"kind": "documentation", "title": "API documentation", "url": "https://docs.rechtsinformationen.bund.de/"},
            {"kind": "openapi", "title": "OpenAPI JSON", "url": ROOT_URL + "/openapi.json"},
        ],
        "warnings": [
            "This is a trial service and may change.",
            "The dataset is not yet complete.",
            "Use existing official sources for production-grade legal research.",
        ],
        "nextActions": [
            'rechtsinformationen-bund documents search --search-term "Suchbegriff" --limit 3',
            "rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision",
        ],
    }


def is_help(value):
    return value in ("--help", "-h", "help")


def emit(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"tool": APP_NAME, "status": "error", "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})
    sys.exit(exit_code)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
