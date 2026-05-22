#!/usr/bin/env python3
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "dipctl"
BASE_URL = "https://search.dip.bundestag.de/api/v1"

ENTITIES = {
    "vorgang": "Proceedings and legislative process metadata",
    "vorgangsposition": "Proceeding positions and parliamentary process steps",
    "drucksache": "Printed paper metadata",
    "drucksache-text": "Printed paper metadata plus full text where available",
    "plenarprotokoll": "Plenary protocol metadata",
    "plenarprotokoll-text": "Plenary protocol metadata plus full text where available",
    "person": "Person master data",
    "aktivitaet": "Parliamentary activities",
}


def main(argv):
    if not argv or is_help(argv[0]):
        print_root_help()
        return 0
    if is_help(argv[-1]):
        print_help_for(argv[:-1])
        return 0

    try:
        if argv[0] == "doctor":
            run_doctor(argv[1:])
        elif len(argv) >= 2 and argv[0] == "person" and argv[1] == "search":
            run_person_search(argv[2:])
        elif len(argv) >= 2 and argv[0] == "person" and argv[1] == "dossier":
            run_person_dossier(argv[2:])
        elif len(argv) >= 2 and argv[0] == "vorgang" and argv[1] == "dossier":
            run_vorgang_dossier(argv[2:])
        elif len(argv) >= 1 and argv[0] == "source":
            run_source(argv[1:])
        elif len(argv) >= 2 and argv[0] in ("plenarprotokoll", "drucksache") and argv[1] == "text":
            run_document_text(argv[0], argv[2:])
        elif len(argv) >= 3 and argv[0:3] == ["plenary", "speech", "search"]:
            run_plenary_speech_search(argv[3:])
        elif len(argv) >= 2 and argv[0] in ENTITIES and argv[1] == "list":
            run_legacy_list(argv[0], argv[2:])
        elif len(argv) >= 2 and argv[0] in ENTITIES and argv[1] == "get":
            run_legacy_get(argv[0], argv[2:])
        else:
            fail(2, "unknown_command", "unknown command path: " + " ".join(argv))
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    return 0


class CLIError(Exception):
    def __init__(self, exit_code, code, message):
        super().__init__(message)
        self.exit_code = exit_code
        self.code = code
        self.message = message


def print_root_help():
    print("""dipctl -- official Bundestag DIP research CLI

Purpose
  Search and cite official parliamentary material from the Bundestag DIP API.

Use this when
  - you need official proceedings, printed papers, protocols, people, or activities
  - you need to distinguish official plenary records from media or campaign quotes
  - you need API-backed source URLs and citation metadata

Do not use this when
  - you need general news context
  - you need lobbying register financial data
  - you need live Bundestag presentation feeds

Fast paths
  Check auth and endpoint health:
    dipctl doctor

  Find a person:
    dipctl person search --name "Gauweiler" --limit 3

  Build an evidence bundle:
    dipctl person dossier --name "Gauweiler"

Legacy endpoint commands
  dipctl vorgang list|get
  dipctl drucksache list|get
  dipctl plenarprotokoll list|get
  dipctl person list|get
  dipctl aktivitaet list|get

Research commands
  doctor
  person search
  person dossier
  vorgang dossier
  source
  plenarprotokoll text
  drucksache text
  plenary speech search

Auth
  Prefer DIP_API_KEY from the environment. --apikey remains supported.
""")


def print_help_for(path):
    if path[:2] == ["person", "dossier"]:
        print("""dipctl person dossier

What it does
  Builds a compact official-source evidence bundle for one person.

Inputs
  --id      Stable DIP person ID
  --name    Name to search when ID is not known
  --limit   Related activity limit, default 10

Examples
  dipctl person dossier --id 760
  dipctl person dossier --name "Gauweiler"
""")
    elif path and path[0] == "doctor":
        print("dipctl doctor\n\nWhat it does\n  Checks auth and endpoint health without printing the API key.")
    elif path[:2] == ["person", "search"]:
        print("dipctl person search\n\nWhat it does\n  Searches official DIP person master data.")
    else:
        print_root_help()


def run_doctor(args):
    flags, _ = parse_args(args)
    key, source = resolve_key(flags)
    out = {
        "status": "ok" if key else "error",
        "tool": APP_NAME,
        "command": "doctor",
        "retrievedAt": now(),
        "summary": {
            "baseUrl": BASE_URL,
            "authRequired": True,
            "apiKeyConfigured": bool(key),
            "apiKeySource": source,
            "maxConcurrentRequests": 25,
            "normalListMaxItems": 100,
            "fullTextListMaxItems": "usually 10",
        },
        "sources": doc_sources(),
        "warnings": [
            "Do not exceed 25 concurrent API requests.",
            "Detailed rate-limit internals beyond official notes are not published.",
            "Use source attribution: Deutscher Bundestag/Bundesrat - DIP.",
        ],
        "nextActions": [
            'dipctl person search --name "Gauweiler"',
            'dipctl plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"',
        ],
    }
    if not key:
        out["error"] = {"code": "missing_api_key", "message": "Set DIP_API_KEY or pass --apikey."}
        write_json(out)
        sys.exit(2)
    body, req_url = api_get("person", {"f.person": ["Steinmeier"], "format": ["json"]}, key)
    out["request"] = request_meta(req_url)
    out["summary"]["healthStatusCode"] = 200
    json.loads(body)
    write_json(out)


def run_legacy_list(entity, args):
    flags, params = parse_args(args)
    key = must_key(flags)
    params.setdefault("format", ["json"])
    body, _ = api_get(entity, params, key)
    if flags.get("limit"):
        data = json.loads(body)
        docs = data.get("documents")
        if isinstance(docs, list):
            limit = positive_int(flags["limit"], "limit")
            data["documents"] = docs[:limit]
            data["clientLimit"] = limit
            data["clientReturned"] = len(data["documents"])
            body = json.dumps(data, ensure_ascii=False, indent=2)
    print(body)


def run_legacy_get(entity, args):
    flags, params = parse_args(args)
    record_id = flags.get("id")
    if not record_id:
        raise CLIError(2, "invalid_arguments", "missing required flag --id")
    key = must_key(flags)
    params.setdefault("format", ["json"])
    body, _ = api_get(f"{entity}/{urllib.parse.quote(record_id, safe='')}", params, key)
    print(body)


def run_person_search(args):
    flags, _ = parse_args(args)
    name = flags.get("name")
    if not name:
        raise CLIError(2, "invalid_arguments", "missing required flag --name")
    limit = positive_int(flags.get("limit", "10"), "limit")
    key = must_key(flags)
    body, req_url = api_get("person", {"f.person": [name], "format": ["json"]}, key)
    data = json.loads(body)
    docs = take_documents(data, limit)
    write_json({
        "status": "ok",
        "tool": APP_NAME,
        "command": "person search",
        "retrievedAt": now(),
        "request": request_meta(req_url),
        "summary": {
            "query": name,
            "numFound": data.get("numFound"),
            "returned": len(docs),
            "clientLimit": limit,
        },
        "items": [compact_item(doc) for doc in docs],
        "sources": [{"title": "DIP API person endpoint", "url": BASE_URL + "/person", "kind": "api"}],
        "warnings": [],
        "nextActions": ["dipctl person dossier --id <id>", 'dipctl aktivitaet list --param "f.person_id=<id>"'],
    })


def run_person_dossier(args):
    flags, _ = parse_args(args)
    key = must_key(flags)
    limit = positive_int(flags.get("limit", "10"), "limit")
    record_id = flags.get("id")
    search_summary = None
    if not record_id:
        name = flags.get("name")
        if not name:
            raise CLIError(2, "invalid_arguments", "pass --id or --name")
        body, req_url = api_get("person", {"f.person": [name], "format": ["json"]}, key)
        data = json.loads(body)
        docs = take_documents(data, 1)
        if not docs:
            raise CLIError(1, "not_found", "no person found for --name")
        record_id = str(docs[0].get("id"))
        search_summary = {"request": request_meta(req_url), "selected": compact_item(docs[0])}
    person_body, person_url = api_get(f"person/{urllib.parse.quote(record_id, safe='')}", {"format": ["json"]}, key)
    person = json.loads(person_body)
    activities = []
    warnings = [
        "Dossier uses official DIP person and activity records only.",
        "Outside quotes, campaign statements, and news context are not included.",
    ]
    try:
        act_body, _ = api_get("aktivitaet", {"f.person_id": [record_id], "format": ["json"]}, key)
        activities = [compact_item(doc) for doc in take_documents(json.loads(act_body), limit)]
    except CLIError as exc:
        warnings.append("Related activities could not be loaded: " + exc.message)
    sources = dedupe_sources(extract_sources(person) + [{"title": "DIP API person detail", "url": BASE_URL + "/person/" + record_id, "kind": "api"}])
    write_json({
        "status": "ok",
        "tool": APP_NAME,
        "command": "person dossier",
        "retrievedAt": now(),
        "request": {"person": request_meta(person_url)},
        "summary": {"person": compact_item(person), "relatedActivitiesShown": len(activities), "search": search_summary},
        "record": person,
        "related": {"activities": activities},
        "sources": sources,
        "warnings": warnings,
        "nextActions": [
            f'dipctl aktivitaet list --param "f.person_id={record_id}"',
            f"dipctl plenary speech search --person-id {record_id} --term <term>",
        ],
    })


def run_vorgang_dossier(args):
    flags, _ = parse_args(args)
    record_id = flags.get("id")
    if not record_id:
        raise CLIError(2, "invalid_arguments", "missing required flag --id")
    key = must_key(flags)
    limit = positive_int(flags.get("limit", "10"), "limit")
    body, req_url = api_get(f"vorgang/{urllib.parse.quote(record_id, safe='')}", {"format": ["json"]}, key)
    record = json.loads(body)
    positions = []
    warnings = ["Dossier uses official DIP proceeding and proceeding-position records."]
    try:
        pos_body, _ = api_get("vorgangsposition", {"f.vorgang": [record_id], "format": ["json"]}, key)
        positions = [compact_item(doc) for doc in take_documents(json.loads(pos_body), limit)]
    except CLIError as exc:
        warnings.append("Related positions could not be loaded: " + exc.message)
    write_json({
        "status": "ok",
        "tool": APP_NAME,
        "command": "vorgang dossier",
        "retrievedAt": now(),
        "request": request_meta(req_url),
        "summary": {"vorgang": compact_item(record), "relatedPositionsShown": len(positions)},
        "record": record,
        "related": {"positions": positions},
        "sources": dedupe_sources(extract_sources(record) + [{"title": "DIP API proceeding detail", "url": BASE_URL + "/vorgang/" + record_id, "kind": "api"}]),
        "warnings": warnings,
        "nextActions": [f'dipctl vorgangsposition list --param "f.vorgang={record_id}"'],
    })


def run_source(args):
    flags, _ = parse_args(args)
    entity = flags.get("type") or flags.get("entity")
    if not entity:
        raise CLIError(2, "invalid_arguments", "missing required flag --type")
    if entity not in ENTITIES:
        raise CLIError(2, "invalid_arguments", "unknown --type: " + entity)
    key = must_key(flags)
    record, request = resolve_record(entity, flags.get("id"), flags.get("document-number"), key)
    sources = extract_sources(record)
    record_id = str(record.get("id", ""))
    if record_id:
        sources.append({"title": "DIP API " + entity + " detail", "url": BASE_URL + "/" + entity + "/" + record_id, "kind": "api"})
    write_json({
        "status": "ok",
        "tool": APP_NAME,
        "command": "source",
        "retrievedAt": now(),
        "request": request,
        "summary": {
            "entity": entity,
            "record": compact_item(record),
            "sourceCount": len(dedupe_sources(sources)),
            "citationSource": "Deutscher Bundestag/Bundesrat - DIP",
        },
        "sources": dedupe_sources(sources),
        "warnings": ["Cite DIP as source. For BT plenary protocols use BT-PlPr. plus document number."],
        "nextActions": ["dipctl " + entity + " get --id <id>"],
    })


def run_document_text(kind, args):
    flags, _ = parse_args(args)
    key = must_key(flags)
    entity = kind + "-text"
    record, request = resolve_record(entity, flags.get("id"), flags.get("document-number"), key)
    text = record.get("text") or ""
    term = flags.get("grep", "")
    context = positive_int(flags.get("context", "220"), "context")
    sources = extract_sources(record)
    record_id = str(record.get("id", ""))
    if record_id:
        sources.append({"title": "DIP API " + entity + " detail", "url": BASE_URL + "/" + entity + "/" + record_id, "kind": "api"})
    out = {
        "status": "ok",
        "tool": APP_NAME,
        "command": kind + " text",
        "retrievedAt": now(),
        "request": request,
        "summary": {"record": compact_item(record), "textLength": len(text), "grep": term, "snippetCount": 0},
        "sources": dedupe_sources(sources),
        "warnings": ["Full text is official DIP text where available.", "Use source attribution: Deutscher Bundestag/Bundesrat - DIP."],
        "nextActions": ["dipctl source --type " + kind + " --id " + record_id],
    }
    if term:
        snips = snippets(text, term, context)
        out["summary"]["snippetCount"] = len(snips)
        out["snippets"] = snips
    else:
        out["textPreview"] = preview(text, 1800)
    write_json(out)


def run_plenary_speech_search(args):
    flags, _ = parse_args(args)
    term = flags.get("term")
    if not term:
        raise CLIError(2, "invalid_arguments", "missing required flag --term")
    if flags.get("document-number") or flags.get("id"):
        run_document_text("plenarprotokoll", args + ["--grep", term])
        return
    key = must_key(flags)
    limit = positive_int(flags.get("limit", "10"), "limit")
    params = {"format": ["json"]}
    if flags.get("person-id"):
        params["f.person_id"] = [flags["person-id"]]
    elif flags.get("person"):
        params["f.person"] = [flags["person"]]
    else:
        raise CLIError(2, "invalid_arguments", "pass --document-number, --person-id, or --person")
    body, req_url = api_get("aktivitaet", params, key)
    matches = []
    for doc in take_documents(json.loads(body), 100):
        if term.lower() in json.dumps(doc, ensure_ascii=False).lower():
            matches.append(compact_item(doc))
            if len(matches) >= limit:
                break
    write_json({
        "status": "ok",
        "tool": APP_NAME,
        "command": "plenary speech search",
        "retrievedAt": now(),
        "request": request_meta(req_url),
        "summary": {"mode": "aktivitaet-search", "term": term, "returned": len(matches), "clientLimit": limit},
        "items": matches,
        "sources": [{"title": "DIP API activity endpoint", "url": BASE_URL + "/aktivitaet", "kind": "api"}],
        "warnings": ["Activity search is official DIP metadata, not a full transcript search."],
        "nextActions": [f'dipctl plenarprotokoll text --document-number <number> --grep "{term}"'],
    })


def resolve_record(entity, record_id, document_number, key):
    if record_id:
        body, req_url = api_get(f"{entity}/{urllib.parse.quote(record_id, safe='')}", {"format": ["json"]}, key)
        return json.loads(body), request_meta(req_url)
    if not document_number:
        raise CLIError(2, "invalid_arguments", "pass --id or --document-number")
    body, req_url = api_get(entity, {"f.dokumentnummer": [document_number], "format": ["json"]}, key)
    docs = take_documents(json.loads(body), 1)
    if not docs:
        raise CLIError(1, "not_found", "no record found for document number")
    return docs[0], request_meta(req_url)


def api_get(path, params, key):
    query = urllib.parse.urlencode(params, doseq=True)
    req_url = BASE_URL + "/" + path.lstrip("/")
    if query:
        req_url += "?" + query
    req = urllib.request.Request(req_url, headers={"Accept": "application/json", "Authorization": "ApiKey " + key})
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.read().decode("utf-8"), req_url
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise CLIError(1, "request_failed", f"DIP API returned HTTP {exc.code}: {preview(body, 500)}")
    except urllib.error.URLError as exc:
        raise CLIError(1, "request_failed", str(exc))


def parse_args(args):
    flags = {}
    params = {}
    i = 0
    while i < len(args):
        arg = args[i]
        if not arg.startswith("--"):
            raise CLIError(2, "invalid_arguments", "unexpected positional argument: " + arg)
        name_value = arg[2:]
        if "=" in name_value:
            name, value = name_value.split("=", 1)
        else:
            name = name_value
            if i + 1 < len(args) and not args[i + 1].startswith("--"):
                i += 1
                value = args[i]
            else:
                value = "true"
        if name == "param":
            if "=" not in value:
                raise CLIError(2, "invalid_arguments", "--param must be key=value")
            key, val = value.split("=", 1)
            params.setdefault(key, []).append(val)
        else:
            flags[name] = value
        i += 1
    return flags, params


def resolve_key(flags):
    if flags.get("apikey"):
        return flags["apikey"], "flag"
    if os.environ.get("DIP_API_KEY"):
        return os.environ["DIP_API_KEY"], "env:DIP_API_KEY"
    return "", "missing"


def must_key(flags):
    key, _ = resolve_key(flags)
    if not key:
        raise CLIError(2, "missing_api_key", "set DIP_API_KEY or pass --apikey")
    return key


def compact_item(doc):
    keys = ["id", "typ", "dokumentart", "vorgangstyp", "titel", "dokumentnummer", "wahlperiode", "herausgeber", "datum", "aktualisiert", "person_id"]
    out = {key: doc[key] for key in keys if key in doc}
    if doc.get("titel"):
        out["title"] = doc["titel"]
    sources = extract_sources(doc)
    if sources:
        out["sources"] = sources
    return out


def extract_sources(value):
    out = []

    def walk(node, key=""):
        if isinstance(node, dict):
            for k, v in node.items():
                walk(v, k)
        elif isinstance(node, list):
            for item in node:
                walk(item, key)
        elif isinstance(node, str) and (node.startswith("https://") or node.startswith("http://")):
            lower = key.lower()
            kind = "url"
            if "pdf" in lower:
                kind = "pdf"
            elif "xml" in lower:
                kind = "xml"
            elif "api" in lower:
                kind = "api"
            out.append({"title": key, "url": node, "kind": kind})

    walk(value)
    return dedupe_sources(out)


def dedupe_sources(sources):
    seen = set()
    out = []
    for source in sources:
        url = source.get("url")
        if url and url not in seen:
            seen.add(url)
            out.append(source)
    return sorted(out, key=lambda item: item["url"])


def snippets(text, term, context):
    out = []
    lower = text.lower()
    needle = term.lower()
    start_at = 0
    while len(out) < 10:
        idx = lower.find(needle, start_at)
        if idx < 0:
            break
        end = idx + len(term)
        s = max(0, idx - context)
        e = min(len(text), end + context)
        out.append({"start": idx, "end": end, "snippet": clean(text[s:e])})
        start_at = end
    return out


def take_documents(data, limit):
    docs = data.get("documents")
    if not isinstance(docs, list):
        return []
    return [doc for doc in docs[:limit] if isinstance(doc, dict)]


def request_meta(req_url):
    return {"method": "GET", "url": redact_url(req_url), "redactions": ["Authorization", "apikey"]}


def redact_url(req_url):
    parsed = urllib.parse.urlparse(req_url)
    qs = urllib.parse.parse_qs(parsed.query, keep_blank_values=True)
    if "apikey" in qs:
        qs["apikey"] = ["REDACTED"]
    return urllib.parse.urlunparse(parsed._replace(query=urllib.parse.urlencode(qs, doseq=True)))


def write_json(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    print(json.dumps({
        "status": "error",
        "tool": APP_NAME,
        "retrievedAt": now(),
        "error": {"code": code, "message": message},
    }, ensure_ascii=False, indent=2), file=sys.stderr)
    sys.exit(exit_code)


def positive_int(raw, name):
    try:
        value = int(raw)
    except ValueError:
        raise CLIError(2, "invalid_arguments", f"--{name} must be a positive integer")
    if value < 1:
        raise CLIError(2, "invalid_arguments", f"--{name} must be a positive integer")
    return value


def doc_sources():
    return [
        {"title": "DIP API help", "url": "https://dip.bundestag.de/%C3%BCber-dip/hilfe/api", "kind": "documentation"},
        {"title": "DIP short documentation PDF", "url": "https://dip.bundestag.de/documents/informationsblatt_zur_dip_api.pdf", "kind": "documentation"},
        {"title": "DIP terms PDF", "url": "https://dip.bundestag.de/documents/nutzungsbedingungen_dip.pdf", "kind": "terms"},
        {"title": "DIP OpenAPI YAML", "url": "https://search.dip.bundestag.de/api/v1/openapi.yaml", "kind": "openapi"},
    ]


def preview(text, max_len):
    cleaned = clean(text)
    if len(cleaned) <= max_len:
        return cleaned
    return cleaned[:max_len] + "..."


def clean(text):
    return " ".join(str(text).split())


def now():
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def is_help(arg):
    return arg in ("--help", "-h", "help")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

