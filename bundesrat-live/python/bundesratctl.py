#!/usr/bin/env python3
import html
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

APP_NAME = "bundesratctl"
BASE_URL = "https://www.bundesrat.de"
OPENAPI_URL = "https://github.com/bundesAPI/bundesrat-api"
SERVICE_BUND_URL = "https://www.service.bund.de/Content/DE/DEBehoerden/B/BR/Bundesrat.html"
IMPRINT_URL = BASE_URL + "/DE/service-navi/impressum/impressum-node.html"
PRIVACY_URL = BASE_URL + "/DE/service-navi/datenschutz/datenschutz-node.html"
ROBOTS_URL = BASE_URL + "/robots.txt"
DEFAULT_LIMIT = 10
SAFE_LIMIT = 100

ENDPOINTS = {
    "startlist": BASE_URL + "/iOS/v3/startlist_table.xml",
    "news": BASE_URL + "/iOS/v3/01_Aktuelles/aktuelles_table.xml",
    "dates": BASE_URL + "/iOS/v3/02_Termine/termine_table.xml",
    "plenum compact": BASE_URL + "/iOS/v3/03_Plenum/plenum_kompakt_table.xml",
    "plenum current": BASE_URL + "/iOS/SharedDocs/3_Plenum/plenum_aktuelleSitzung_table.xml",
    "plenum chronological": BASE_URL + "/iOS/SharedDocs/3_Plenum/plenum_toChronologisch_table.xml",
    "plenum next": BASE_URL + "/iOS/SharedDocs/3_Plenum/plenum_naechsteSitzungen.xml",
    "members": BASE_URL + "/iOS/SharedDocs/2_Mitglieder/mitglieder_table.xml",
    "votes": BASE_URL + "/iOS/v3/06_Stimmen/stimmverteilung.xml",
    "presidium": BASE_URL + "/iOS/v3/05_Bundesrat/Praesidium/bundesrat_praesidium.xml",
}

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
        elif argv[0] == "examples":
            print_examples()
        elif argv[0] == "startlist":
            run_feed("startlist", argv[1:])
        elif argv[0] == "news" and len(argv) > 1 and argv[1] == "search":
            run_feed_search("news", argv[2:])
        elif argv[0] == "news" and len(argv) > 1 and argv[1] == "page":
            run_page("news page", argv[2:])
        elif argv[0] == "news":
            run_feed("news", argv[1:])
        elif argv[0] == "dates" and len(argv) > 1 and argv[1] == "search":
            run_feed_search("dates", argv[2:])
        elif argv[0] == "dates" and len(argv) > 1 and argv[1] == "page":
            run_page("dates page", argv[2:])
        elif argv[0] == "dates":
            run_feed("dates", argv[1:])
        elif argv[:2] == ["plenum", "compact"]:
            run_plenum("plenum compact", argv[2:])
        elif argv[:2] == ["plenum", "current"]:
            run_plenum("plenum current", argv[2:])
        elif argv[:2] == ["plenum", "chronological"]:
            run_plenum("plenum chronological", argv[2:])
        elif argv[:2] == ["plenum", "next"]:
            run_plenum_next(argv[2:])
        elif argv[:2] == ["plenum", "dossier"]:
            run_plenum("plenum compact", argv[2:])
        elif argv[0] == "members" and len(argv) > 1 and argv[1] == "search":
            run_members_search(argv[2:])
        elif argv[0] == "members" and len(argv) > 1 and argv[1] == "dossier":
            run_member_dossier(argv[2:])
        elif argv[0] == "members":
            run_members(argv[1:])
        elif argv[0] == "votes" and len(argv) > 1 and argv[1] == "summary":
            run_feed("votes", argv[2:])
        elif argv[0] == "votes":
            run_feed("votes", argv[1:])
        elif argv[0] == "presidium":
            run_feed("presidium", argv[1:])
        elif argv[0] == "page":
            run_page("page", argv[1:])
        elif argv[0] == "source":
            run_source(argv[1:])
        else:
            raise CLIError(2, "unknown_command", "unknown command; run bundesratctl --help")
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""bundesratctl -- Bundesrat live/app XML research CLI

Purpose
  Discover and normalize public Bundesrat app XML feeds for news, dates,
  plenary-session summaries and agenda items, members, vote distribution,
  presidium/context pages, and source URLs.

Fast paths
  bundesratctl doctor
  bundesratctl news --limit 5
  bundesratctl news search --term "Bovenschulte" --limit 3
  bundesratctl members search --name "Ã–zdemir" --limit 3
  bundesratctl members dossier --name "Ã–zdemir" --grep "Bundesrat"
  bundesratctl plenum compact --limit 1 --top-limit 3
  bundesratctl plenum current --limit 1 --top-limit 5
  bundesratctl plenum next

Endpoint-compatible commands
  startlist
  news
  dates
  plenum compact
  plenum current
  plenum chronological
  plenum next
  members
  votes
  presidium
""")


def print_help(path):
    joined = " ".join(path)
    if joined in {"news search", "dates search"}:
        print('bundesratctl news search --term "Bovenschulte" --limit 3')
    elif joined == "members search":
        print('bundesratctl members search --name "Ã–zdemir" --limit 3')
    elif joined == "members dossier":
        print('bundesratctl members dossier --name "Ã–zdemir" --grep "Bundesrat"')
    elif joined in {"page", "news page", "dates page"}:
        print('bundesratctl page --url "https://www.bundesrat.de/..." --grep "term"')
    else:
        print_root_help()


def print_examples():
    print("""bundesratctl examples

1. bundesratctl doctor
2. bundesratctl startlist --limit 12
3. bundesratctl news --limit 5
4. bundesratctl news search --term "Bovenschulte" --limit 3
5. bundesratctl dates --limit 5
6. bundesratctl members search --name "Ã–zdemir" --limit 3
7. bundesratctl members dossier --name "Ã–zdemir" --grep "Bundesrat"
8. bundesratctl plenum compact --limit 1 --top-limit 3
9. bundesratctl plenum current --limit 1 --top-limit 5
10. bundesratctl plenum compact --raw
""")


def run_doctor(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 5, 10)
    checks = ["startlist", "news", "dates", "plenum compact", "members", "votes"][:limit]
    summary = {
        "authRequired": False,
        "publishedRateLimit": "No exact public request quota was found in the OpenAPI wrapper or Bundesrat website material. The site robots.txt currently publishes Crawl-delay: 30; use small limits, cache repeated feed calls, and back off on 429/5xx responses.",
        "fairUseHints": [
            "Prefer search and dossier commands before broad source-page expansion.",
            "Respect robots.txt Crawl-delay: 30 for crawling-like workflows.",
            "Use --limit and --top-limit on broad feeds.",
            "Preserve source URLs, retrieval timestamps, and image/media copyright fields.",
        ],
        "endpoints": [],
    }
    status = "ok"
    for name in checks:
        request_url = with_default_view(ENDPOINTS[name], {})
        code, content_type, body = fetch_raw(request_url)
        ok = 200 <= code < 300
        if not ok:
            status = "degraded"
        summary["endpoints"].append({"name": name, "url": ENDPOINTS[name], "statusCode": code, "contentType": content_type, "ok": ok, "bodyPreview": truncate(strip_space(body), 180)})
    payload = envelope("doctor", BASE_URL, {"limit": limit})
    payload["status"] = status
    payload["summary"] = summary
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["bundesratctl news --limit 5", 'bundesratctl members search --name "Ã–zdemir" --limit 3', "bundesratctl plenum compact --limit 1 --top-limit 3"]
    emit(payload)


def run_feed(key, argv):
    parsed = parse_args(argv)
    body, request_url = fetch_endpoint(key, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    grep = first_non_empty(parsed["flags"].get("grep"), parsed["flags"].get("term"), parsed["flags"].get("q"))
    items = compact_items(body, key, limit, grep, flag_bool(parsed, "include-raw"))
    payload = envelope(key, request_url, {"limit": limit, "grep": grep})
    payload["summary"] = {"totalItems": count_item_like(body), "returned": len(items), "grep": grep}
    payload["items"] = items
    payload["sources"] = source("Bundesrat " + key + " XML feed", request_url, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_items(items, key)
    emit(payload)


def run_feed_search(key, argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), parsed["flags"].get("name"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", key + " search requires --term")
    parsed["flags"]["grep"] = term
    run_feed(key, rebuild_args(parsed))


def run_members(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_endpoint("members", parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    items = compact_employees(body, limit, "", flag_bool(parsed, "include-raw"))
    payload = envelope("members", request_url, {"limit": limit})
    payload["summary"] = {"totalMembers": len(blocks(body, "employee")), "returned": len(items)}
    payload["items"] = items
    payload["sources"] = source("Bundesrat member XML feed", request_url, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['bundesratctl members search --name "Ã–zdemir" --limit 3']
    emit(payload)


def run_members_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("name"), parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "members search requires --name or --term")
    body, request_url = fetch_endpoint("members", {})
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    items = compact_employees(body, limit, term, flag_bool(parsed, "include-raw"))
    payload = envelope("members search", request_url, {"term": term, "limit": limit})
    payload["summary"] = {"term": term, "totalMembers": len(blocks(body, "employee")), "matchesReturned": len(items)}
    payload["items"] = items
    payload["sources"] = source("Bundesrat member XML feed", request_url, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_employees(items)
    emit(payload)


def run_member_dossier(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("name"), parsed["flags"].get("url"), parsed["flags"].get("term"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_member", "members dossier requires --name or --url")
    body, request_url = fetch_endpoint("members", {})
    grep = parsed["flags"].get("grep", "")
    matches = compact_employees(body, SAFE_LIMIT, term, flag_bool(parsed, "include-raw"))
    if not matches:
        raise CLIError(2, "member_not_found", "member not found in current Bundesrat feed: " + term)
    item = matches[0]
    text = item.get("evidenceText", "")
    payload = envelope("members dossier", request_url, {"term": term, "grep": grep})
    payload["summary"] = {"name": item.get("name"), "party": item.get("party"), "state": item.get("state"), "profileUrl": item.get("url"), "snippetCount": len(grep_snippets(text, grep, 8, 650))}
    payload["items"] = [{"profile": item, "snippets": grep_snippets(text, grep, 8, 650)}]
    payload["sources"] = [{"title": "Bundesrat member XML feed", "url": request_url, "kind": "api_endpoint"}]
    if item.get("url"):
        payload["sources"].append({"title": "Bundesrat member profile", "url": item["url"], "kind": "public_profile"})
        payload["nextActions"] = [f'bundesratctl page --url "{item["url"]}" --grep "{first_non_empty(grep, "Bundesrat")}"']
    payload["warnings"] = default_warnings()
    emit(payload)


def run_plenum(key, argv):
    parsed = parse_args(argv)
    body, request_url = fetch_endpoint(key, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    top_limit = limit_flag_name(parsed, "top-limit", DEFAULT_LIMIT, SAFE_LIMIT)
    grep = first_non_empty(parsed["flags"].get("grep"), parsed["flags"].get("term"), parsed["flags"].get("q"))
    items = compact_plenum(body, key, limit, top_limit, grep, flag_bool(parsed, "include-raw"))
    payload = envelope(key, request_url, {"limit": limit, "topLimit": top_limit, "grep": grep})
    payload["summary"] = plenum_summary(body, key, len(items), grep)
    payload["items"] = items
    payload["sources"] = source("Bundesrat " + key + " XML feed", request_url, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_items(items, key)
    emit(payload)


def run_plenum_next(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_endpoint("plenum next", parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    item_blocks = blocks(body, "item")
    items = compact_items(body, "plenum next", limit, parsed["flags"].get("grep", ""), flag_bool(parsed, "include-raw"))
    sessions = table_rows(tag(item_blocks[0], "detail")) if item_blocks else []
    payload = envelope("plenum next", request_url, {"limit": limit})
    payload["summary"] = {"returned": len(items), "upcomingSessions": sessions}
    payload["items"] = items
    payload["sources"] = source("Bundesrat next plenary sessions XML feed", request_url, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["bundesratctl plenum current --limit 1 --top-limit 5", "bundesratctl plenum compact --limit 1 --top-limit 5"]
    emit(payload)


def run_page(command, argv):
    parsed = parse_args(argv)
    source_url = first_non_empty(parsed["flags"].get("url"), parsed["flags"].get("source-url"), " ".join(parsed["positionals"]))
    if not source_url:
        raise CLIError(2, "missing_url", command + " requires --url")
    if not source_url.startswith(BASE_URL + "/"):
        raise CLIError(2, "unsafe_url", "page only accepts https://www.bundesrat.de URLs")
    code, content_type, body = fetch_raw(source_url)
    grep = parsed["flags"].get("grep", "")
    text = strip_html(body)
    payload = envelope(command, source_url, {"url": source_url, "grep": grep})
    payload["summary"] = {"url": source_url, "statusCode": code, "contentType": content_type, "title": html_title(body), "textLength": len(text), "snippetCount": len(grep_snippets(text, grep, 8, 650))}
    payload["items"] = grep_snippets(text, grep, 8, 650)
    payload["sources"] = source("Bundesrat public source page", source_url, "public_page")
    payload["warnings"] = default_warnings() + ["Public HTML extraction is best-effort; prefer XML feed fields for structured metadata."]
    payload["nextActions"] = [f'bundesratctl source --url "{source_url}"']
    emit(payload)


def run_source(argv):
    parsed = parse_args(argv)
    source_url = first_non_empty(parsed["flags"].get("url"), " ".join(parsed["positionals"]))
    if not source_url:
        raise CLIError(2, "missing_url", "source requires --url")
    payload = envelope("source", source_url, {"url": source_url})
    payload["summary"] = {"url": source_url, "kind": source_kind(source_url), "citation": "Bundesrat, " + source_url}
    payload["sources"] = source("Bundesrat source", source_url, source_kind(source_url))
    payload["warnings"] = default_warnings()
    if source_url.startswith(BASE_URL + "/"):
        payload["nextActions"] = [f'bundesratctl page --url "{source_url}"']
    emit(payload)


def fetch_endpoint(key, params):
    if key not in ENDPOINTS:
        raise CLIError(2, "unknown_endpoint", "unknown endpoint: " + key)
    request_url = with_default_view(ENDPOINTS[key], params or {})
    code, _, body = fetch_raw(request_url)
    if not (200 <= code < 300):
        raise RuntimeError(f"upstream status {code} from {request_url}: {truncate(strip_space(body), 300)}")
    return body, request_url


def fetch_raw(request_url):
    req = urllib.request.Request(request_url, headers={"User-Agent": "germany-skills/bundesratctl-python-2.0"})
    try:
        with urllib.request.urlopen(req, timeout=45) as response:
            return response.status, response.headers.get("Content-Type", ""), response.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.headers.get("Content-Type", ""), exc.read().decode("utf-8", "replace")


def with_default_view(base, params):
    query = dict(params or {})
    query.setdefault("view", "renderXml")
    return base + "?" + urllib.parse.urlencode(query)


def compact_items(raw, key, limit, grep, include_raw):
    out = []
    for item in blocks(raw, "item"):
        if grep and grep.lower() not in item_search_text(item).lower():
            continue
        out.append(compact_item(item, key, grep, include_raw))
        if len(out) >= limit:
            break
    return out


def compact_item(item, key, grep, include_raw):
    detail = tag(item, "detail")
    text = first_non_empty(tag(item, "bodyText"), tag(item, "description"), tag(item, "abstract"), strip_html(detail))
    source_url = tag(item, "url")
    out = {
        "type": tag(item, "type"),
        "id": tag(item, "id"),
        "name": tag(item, "name"),
        "title": first_non_empty(tag(item, "title"), tag(item, "name")),
        "url": source_url,
        "date": tag(item, "date"),
        "dateOfIssue": tag(item, "dateOfIssue"),
        "startDate": tag(item, "startdate"),
        "stopDate": tag(item, "stopdate"),
        "summary": truncate(strip_html(text), 700),
        "imageUrl": tag(item, "imagePath"),
        "imageDate": tag(item, "imageDate"),
        "imageCaption": tag(item, "imageCaption"),
        "sources": source("Bundesrat source", source_url, source_kind(source_url)),
        "links": extract_links(detail, 10),
        "snippets": grep_snippets(strip_html(detail + " " + text), grep, 4, 650),
        "nextActions": next_actions_for_url(source_url, key),
    }
    if include_raw:
        out["raw"] = item
    return out


def compact_employees(raw, limit, term, include_raw):
    out = []
    for employee in blocks(raw, "employee"):
        if term and term.lower() not in employee_search_text(employee).lower():
            continue
        first_name = tag(employee, "firstname")
        last_name = tag(employee, "name")
        source_url = tag(employee, "url")
        evidence = strip_html(" ".join([tag(employee, "detail1"), tag(employee, "detail2"), tag(employee, "detail3")]))
        item = {
            "name": strip_space(first_name + " " + last_name),
            "firstName": first_name,
            "lastName": last_name,
            "party": tag(employee, "party"),
            "state": tag(employee, "state"),
            "isBundesratMember": tag(employee, "brmitglied"),
            "isMember": tag(employee, "mitglied"),
            "isBevollmaechtigt": tag(employee, "bv"),
            "url": source_url,
            "imageUrl": tag(employee, "imagePath"),
            "roles": truncate(strip_html(tag(employee, "detail1")), 1000),
            "biography": truncate(strip_html(tag(employee, "detail2")), 1000),
            "contact": truncate(strip_html(tag(employee, "detail3")), 1000),
            "evidenceText": evidence,
            "sources": source("Bundesrat member profile", source_url, "public_profile"),
            "nextActions": next_actions_for_url(source_url, "members"),
        }
        if include_raw:
            item["raw"] = employee
        out.append(item)
        if len(out) >= limit:
            break
    return out


def compact_plenum(raw, key, limit, top_limit, grep, include_raw):
    out = []
    header = block(raw, "header")
    if header and "<url>" in header:
        out.append({
            "kind": "header",
            "url": tag(header, "url"),
            "title": first_non_empty(tag(header, "titel2"), tag(header, "title")),
            "subtitle": strip_space(" ".join([tag(header, "titel1"), tag(header, "titel3"), tag(header, "titelAlt")])),
            "detailType": tag(header, "detailTyp"),
            "summary": truncate(strip_html(first_non_empty(tag(header, "vorschautext"), tag(header, "detail"))), 1000),
            "sources": source("Bundesrat plenary page", tag(header, "url"), "public_page"),
            "links": extract_links(tag(header, "detail"), 10),
            "snippets": grep_snippets(strip_html(tag(header, "detail")), grep, 4, 650),
            "nextActions": next_actions_for_url(tag(header, "url"), key),
        })
    for top in blocks(raw, "top"):
        if grep and grep.lower() not in strip_html(top).lower():
            continue
        detail = first_non_empty(tag(top, "detail"), tag(top, "topdetail"))
        source_url = tag(top, "url")
        item = {
            "kind": "top",
            "top": first_non_empty(tag(top, "nr"), tag(top, "toptitle")),
            "printMatter": tag(top, "topdrucksache"),
            "filter": tag(top, "filter"),
            "title": first_non_empty(tag(top, "title"), tag(top, "topheader")),
            "url": source_url,
            "summary": truncate(strip_html(first_non_empty(tag(top, "topheader"), detail)), 900),
            "links": extract_links(detail, 14),
            "snippets": grep_snippets(strip_html(detail), grep, 5, 650),
            "sources": source("Bundesrat plenary TOP", source_url, source_kind(source_url)),
            "nextActions": next_actions_for_url(source_url, key),
        }
        if include_raw:
            item["raw"] = top
        out.append(item)
        if len(out) >= limit + top_limit:
            break
    return (out or compact_items(raw, key, limit, grep, include_raw))[: limit + top_limit]


def plenum_summary(raw, key, returned, grep):
    return {"title": tag(raw, "title"), "header": truncate(strip_html(block(raw, "header")), 900), "topCount": len(blocks(raw, "top")), "itemCount": len(blocks(raw, "item")), "returned": returned, "grep": grep, "sourceUrl": first_non_empty(tag(block(raw, "header"), "url"), ENDPOINTS[key])}


def item_search_text(item):
    return strip_html(" ".join([tag(item, "type"), tag(item, "id"), tag(item, "name"), tag(item, "title"), tag(item, "url"), tag(item, "date"), tag(item, "dateOfIssue"), tag(item, "bodyText"), tag(item, "description"), tag(item, "abstract"), tag(item, "detail")]))


def employee_search_text(employee):
    return strip_html(" ".join([tag(employee, "firstname"), tag(employee, "name"), tag(employee, "party"), tag(employee, "state"), tag(employee, "url"), tag(employee, "detail1"), tag(employee, "detail2"), tag(employee, "detail3")]))


def next_actions_from_items(items, key):
    actions = []
    for item in items:
        actions.extend(next_actions_for_url(str(item.get("url", "")), key))
        if len(actions) >= 4:
            return actions
    if key == "news":
        return ['bundesratctl news search --term "Bovenschulte" --limit 3']
    if key == "dates":
        return ['bundesratctl dates search --term "Ausschuss" --limit 5']
    return ["bundesratctl plenum compact --limit 1 --top-limit 3"]


def next_actions_from_employees(items):
    actions = [f'bundesratctl members dossier --name "{item.get("name")}"' for item in items[:3] if item.get("name")]
    return actions or ['bundesratctl members search --name "Ã–zdemir" --limit 3']


def next_actions_for_url(source_url, key):
    if not source_url:
        return []
    actions = []
    if source_url.startswith(BASE_URL + "/"):
        actions.append(f'bundesratctl page --url "{source_url}"')
    if key == "news":
        actions.append(f'bundesratctl news page --url "{source_url}"')
    if key == "dates":
        actions.append(f'bundesratctl dates page --url "{source_url}"')
    return actions


def default_sources():
    return [{"title": "bundesAPI Bundesrat OpenAPI wrapper", "url": OPENAPI_URL, "kind": "openapi_reference"}, {"title": "service.bund.de Bundesrat profile", "url": SERVICE_BUND_URL, "kind": "official_context"}, {"title": "Bundesrat robots.txt", "url": ROBOTS_URL, "kind": "fair_use"}, {"title": "Bundesrat Impressum", "url": IMPRINT_URL, "kind": "terms"}, {"title": "Bundesrat DatenschutzerklÃ¤rung", "url": PRIVACY_URL, "kind": "privacy"}]


def default_warnings():
    return ["No exact public rate limit for these Bundesrat XML feeds was found; robots.txt publishes Crawl-delay: 30, so avoid crawling-style rapid page expansion.", "This app/live XML surface is current-publication oriented, not a complete historical archive.", "Bundesrat public pages can include image/media copyright notices; preserve source URLs and copyright fields in final artifacts.", "Votes by individual Land are generally not always recorded by the Bundesrat itself; inspect plenary records and linked state pages where the distinction matters."]


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


def rebuild_args(parsed):
    args = []
    for key, value in parsed["flags"].items():
        args.extend(["--" + key, str(value)])
    for key, value in parsed["params"].items():
        args.extend(["--param", key + "=" + str(value)])
    args.extend(parsed["positionals"])
    return args


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


def count_item_like(raw):
    count = len(blocks(raw, "item"))
    return count or len(blocks(raw, "employee")) + len(blocks(raw, "top"))


def source_kind(source_url):
    if not source_url:
        return "unknown"
    if "dip.bundestag.de" in source_url:
        return "dip_reference"
    if "/SharedDocs/personen/" in source_url:
        return "public_profile"
    if "/SharedDocs/drucksachen/" in source_url or "/drs.html" in source_url:
        return "official_document"
    if "/DE/plenum/" in source_url:
        return "plenary_page"
    if "/SharedDocs/pm/" in source_url:
        return "press_release"
    return "public_page"


def source(title, url, kind):
    return [{"title": title, "url": url, "kind": kind}] if url else []


def block(xml, name):
    match = re.search(rf"<{re.escape(name)}(?:\s[^>]*)?>(.*?)</{re.escape(name)}>", xml or "", re.I | re.S)
    return match.group(1) if match else ""


def blocks(xml, name):
    return re.findall(rf"<{re.escape(name)}(?:\s[^>]*)?>.*?</{re.escape(name)}>", xml or "", re.I | re.S)


def tag(xml, name):
    value = block(xml, name)
    value = re.sub(r"^<!\[CDATA\[", "", value)
    value = re.sub(r"\]\]>$", "", value)
    return html.unescape(value).strip()


SCRIPT_STYLE_RE = re.compile(r"<(script|style)[^>]*>.*?</(script|style)>", re.I | re.S)
TAG_RE = re.compile(r"<[^>]+>", re.S)
SPACE_RE = re.compile(r"\s+")
TITLE_RE = re.compile(r"<title[^>]*>(.*?)</title>", re.I | re.S)
LINK_RE = re.compile(r"<a\s+[^>]*href=[\"']([^\"']+)[\"'][^>]*>(.*?)</a>", re.I | re.S)
TABLE_ROW_RE = re.compile(r"<tr>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*</tr>", re.I | re.S)


def strip_html(value):
    value = re.sub(r"^<!\[CDATA\[", "", value or "")
    value = re.sub(r"\]\]>$", "", value)
    value = SCRIPT_STYLE_RE.sub(" ", value)
    value = value.replace("<br/>", " ").replace("<br>", " ").replace("<br />", " ")
    return strip_space(html.unescape(TAG_RE.sub(" ", value)))


def strip_space(value):
    return SPACE_RE.sub(" ", value or "").strip()


def truncate(value, max_len):
    value = strip_space(value)
    return value if len(value) <= max_len else value[:max_len] + "..."


def grep_snippets(text, grep, limit, max_len):
    text = strip_space(text)
    if not text:
        return []
    if not grep:
        return [{"text": truncate(text, max_len)}]
    lower = text.lower()
    needle = grep.lower().strip()
    out, seen, start_from = [], set(), 0
    while len(out) < limit:
        idx = lower.find(needle, start_from)
        if idx < 0:
            break
        start = max(0, idx - max_len // 2)
        end = min(len(text), start + max_len)
        snippet = text[start:end].strip()
        key = snippet[:180]
        if key not in seen:
            out.append({"grep": grep, "text": snippet})
            seen.add(key)
        start_from = idx + len(needle)
    return out


def extract_links(value, limit):
    out, seen = [], set()
    for match in LINK_RE.finditer(value or ""):
        raw_url = html.unescape(match.group(1)).strip()
        if not raw_url or raw_url.startswith(("mailto:", "tel:")):
            continue
        if raw_url.startswith("/"):
            raw_url = BASE_URL + raw_url
        if not raw_url.startswith("http"):
            raw_url = BASE_URL + "/" + raw_url.lstrip("./")
        if raw_url in seen:
            continue
        seen.add(raw_url)
        out.append({"title": truncate(strip_html(match.group(2)), 160), "url": raw_url, "kind": source_kind(raw_url)})
        if len(out) >= limit:
            break
    return out


def table_rows(value):
    return [{"date": strip_html(match.group(1)), "time": strip_html(match.group(2))} for match in TABLE_ROW_RE.finditer(value or "")]


def html_title(value):
    match = TITLE_RE.search(value or "")
    return strip_html(match.group(1)) if match else ""


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
