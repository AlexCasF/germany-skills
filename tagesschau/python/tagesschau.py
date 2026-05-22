#!/usr/bin/env python3
"""Agent-friendly Tagesschau public JSON feed CLI mirror."""

from __future__ import annotations

import html
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any


APP_NAME = "tagesschau"
BASE_URL = "https://www.tagesschau.de"
HOMEPAGE_URL = f"{BASE_URL}/api2u/homepage"
NEWS_URL = f"{BASE_URL}/api2u/news"
CHANNELS_URL = f"{BASE_URL}/api2u/channels"
SEARCH_URL = f"{BASE_URL}/api2u/search"
API_DOCS_URL = "https://github.com/bundesAPI/tagesschau-api"
OPENAPI_URL = "https://github.com/bundesAPI/tagesschau-api/raw/refs/heads/main/openapi.yaml"
CC_URL = "https://www.tagesschau.de/multimedia/video/creative-commons-index-100.html"
RSS_INFO_URL = "https://www.tagesschau.de/infoservices/rssfeeds"
USER_AGENT = "germany-skills/tagesschau-python-2.0"
DEFAULT_LIMIT = 10
MAX_LIMIT = 30


class CliError(Exception):
    def __init__(self, code: str, message: str, exit_code: int = 1):
        super().__init__(message)
        self.code = code
        self.message = message
        self.exit_code = exit_code


def main(argv: list[str]) -> int:
    try:
        if not argv or is_help(argv[0]):
            print_root_help()
            return 0
        if is_help(argv[-1]):
            print_help(argv[:-1])
            return 0
        if argv[0] == "doctor":
            run_doctor(argv[1:])
        elif argv[0] == "examples":
            print_examples()
        elif argv[0] == "source":
            run_source(argv[1:])
        elif argv[0] == "fields":
            run_fields(argv[1:])
        elif argv[0] == "homepage":
            run_feed("homepage", HOMEPAGE_URL, argv[1:])
        elif argv[0] == "news":
            run_feed("news", NEWS_URL, argv[1:])
        elif argv[0] == "channels":
            run_feed("channels", CHANNELS_URL, argv[1:])
        elif argv[0] == "search":
            run_search(argv[1:])
        elif match(argv, "article", "get"):
            run_article("article get", argv[2:], False)
        elif match(argv, "article", "source"):
            run_article_source(argv[2:])
        elif match(argv, "article", "dossier"):
            run_article("article dossier", argv[2:], True)
        else:
            raise CliError("unknown_command", f"unknown command: {' '.join(argv)}", 2)
        return 0
    except CliError as exc:
        emit_error(exc.code, exc.message)
        return exc.exit_code
    except Exception as exc:  # pragma: no cover
        emit_error("unexpected_error", str(exc))
        return 1


def print_root_help() -> None:
    print(
        """tagesschau 2.0 - Tagesschau public JSON feed research CLI

Usage:
  tagesschau doctor
  tagesschau homepage --limit 5
  tagesschau news --ressort inland --limit 5
  tagesschau channels --limit 5
  tagesschau search --text "Bundestag" --limit 5
  tagesschau article get --url "https://www.tagesschau.de/...-100.html" --grep "Bundestag"
  tagesschau article dossier --url "https://www.tagesschau.de/...-100.html"

Tagesschau is a current-news context source, not the sole official evidence for parliamentary, legal, fiscal, or statistical claims.
"""
    )


def print_help(args: list[str]) -> None:
    if not args:
        print_root_help()
    elif args[0] == "search":
        print("search flags: --text/--searchText TERM --limit 1-30 --result-page N --include-raw --raw --param key=value")
    elif args[0] == "news":
        print("news flags: --ressort inland|ausland|wirtschaft|sport|video|investigativ|wissen --regions 1,2 --limit 1-30 --include-raw --raw --param key=value")
    elif args[0] == "homepage":
        print("homepage flags: --limit 1-30 --include-regional --include-raw --raw")
    elif match(args, "article", "get"):
        print("article get flags: --url URL --grep TERM --limit 1-30 --include-raw --raw")
    elif match(args, "article", "source"):
        print("article source flags: --url URL")
    elif match(args, "article", "dossier"):
        print("article dossier flags: --url URL --grep TERM --limit 1-30 --include-raw")
    else:
        print_root_help()


def print_examples() -> None:
    print(
        """Examples:
  tagesschau doctor
  tagesschau homepage --limit 5
  tagesschau news --ressort inland --limit 5
  tagesschau search --text "Bundestag" --limit 5
  tagesschau search --param searchText=Bundestag --param pageSize=5
  tagesschau article get --url "https://www.tagesschau.de/inland/example-100.html" --grep "Bundestag"
"""
    )


def run_doctor(argv: list[str]) -> None:
    checks: list[dict[str, Any]] = []
    for name, url in [
        ("homepage", HOMEPAGE_URL),
        ("news", with_params(NEWS_URL, {"ressort": "inland"})),
        ("channels", CHANNELS_URL),
        ("search", with_params(SEARCH_URL, {"searchText": "Bundestag", "pageSize": "1"})),
    ]:
        item: dict[str, Any] = {"name": name, "url": url}
        try:
            status, body, content_type = fetch_raw(url)
            item.update({"ok": 200 <= status < 300, "statusCode": status, "contentType": content_type, "bodyBytes": len(body)})
        except Exception as exc:
            item.update({"ok": False, "error": str(exc)})
        checks.append(item)
    payload = envelope("doctor", "GET", "multiple", {})
    payload["summary"] = {
        "authRequired": False,
        "documentedLimit": "The published API documentation states that more than 60 requests per hour are not allowed.",
        "usageRestrictions": "Private, non-commercial use is allowed; publication is not allowed except for content explicitly released under a Creative Commons license.",
        "recommendedRole": "Use as a current-news context layer, not as the sole official source for institutional or statistical claims.",
        "endpointHealth": checks,
        "copyrightSensitive": True,
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['tagesschau search --text "Bundestag" --limit 5', "tagesschau homepage --limit 5", "tagesschau source"]
    emit(payload)


def run_source(argv: list[str]) -> None:
    payload = envelope("source", "GET", API_DOCS_URL, {})
    payload["summary"] = {
        "publisher": "Tagesschau / ARD-aktuell; API documentation mirrored by bundesAPI.",
        "authRequired": False,
        "documentedLimit": "No more than 60 requests per hour.",
        "reuseRestriction": "Private, non-commercial use only; no publication except explicitly CC-licensed offers.",
        "primaryEndpoints": [HOMEPAGE_URL, NEWS_URL, CHANNELS_URL, SEARCH_URL],
        "articleURLPattern": "Public detailsweb URLs can be converted to /api2u/...json detail URLs.",
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["tagesschau fields", 'tagesschau search --text "Bundestag" --limit 5']
    emit(payload)


def run_fields(argv: list[str]) -> None:
    payload = envelope("fields", "GET", API_DOCS_URL, {})
    payload["summary"] = {
        "feeds": [
            {"command": "homepage", "meaning": "Selected current and breaking items shown in the app homepage."},
            {"command": "news", "meaning": "Current news feed; filterable by ressort and region."},
            {"command": "channels", "meaning": "Current livestream/program channels."},
            {"command": "search", "meaning": "Search feed with searchText, resultPage, and pageSize."},
        ],
        "ressorts": ["inland", "ausland", "wirtschaft", "sport", "video", "investigativ", "wissen"],
        "regions": {
            "1": "Baden-W\u00fcrttemberg", "2": "Bayern", "3": "Berlin", "4": "Brandenburg", "5": "Bremen", "6": "Hamburg", "7": "Hessen", "8": "Mecklenburg-Vorpommern",
            "9": "Niedersachsen", "10": "Nordrhein-Westfalen", "11": "Rheinland-Pfalz", "12": "Saarland", "13": "Sachsen", "14": "Sachsen-Anhalt", "15": "Schleswig-Holstein", "16": "Th\u00fcringen",
        },
        "coreArticleFields": ["title", "topline", "date", "details", "detailsweb", "shareURL", "firstSentence", "ressort", "type", "tags"],
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['tagesschau search --text "Bundestag" --limit 5', "tagesschau homepage --limit 5"]
    emit(payload)


def run_feed(command: str, endpoint: str, argv: list[str]) -> None:
    parsed = parse_args(argv)
    params = dict(parsed["params"])
    for key in ("ressort", "regions"):
        if parsed["flags"].get(key):
            params[key] = parsed["flags"][key]
    request_url = with_params(endpoint, params)
    status, body, _content_type = fetch_raw(request_url)
    if not 200 <= status < 300:
        raise CliError("upstream_http_error", f"upstream status {status} from {request_url}: {strip_space(body)[:260]}")
    if flag_bool(parsed, "raw"):
        sys.stdout.write(body)
        return
    data = json.loads(body)
    limit = limit_flag(parsed)
    items = compact_feed_items(data, limit, flag_bool(parsed, "include-regional"))
    payload = envelope(command, "GET", request_url, params)
    payload["summary"] = {"type": data.get("type"), "itemsReturned": len(items), "rawCounts": feed_counts(data)}
    payload["items"] = items
    payload["sources"] = [{"kind": "api_request", "title": "Tagesschau API request", "url": request_url}] + default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_items(items)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_search(argv: list[str]) -> None:
    parsed = parse_args(argv)
    params = dict(parsed["params"])
    text = first(parsed["flags"].get("text"), parsed["flags"].get("searchText"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if text:
        params["searchText"] = text
    limit = limit_flag(parsed)
    params.setdefault("pageSize", str(limit))
    if parsed["flags"].get("page-size") or parsed["flags"].get("pageSize"):
        params["pageSize"] = first(parsed["flags"].get("page-size"), parsed["flags"].get("pageSize"))
    if parsed["flags"].get("result-page") or parsed["flags"].get("resultPage") or parsed["flags"].get("page"):
        params["resultPage"] = first(parsed["flags"].get("result-page"), parsed["flags"].get("resultPage"), parsed["flags"].get("page"))
    request_url = with_params(SEARCH_URL, params)
    status, body, _content_type = fetch_raw(request_url)
    if not 200 <= status < 300:
        raise CliError("upstream_http_error", f"upstream status {status} from {request_url}: {strip_space(body)[:260]}")
    if flag_bool(parsed, "raw"):
        sys.stdout.write(body)
        return
    data = json.loads(body)
    items = compact_array(data.get("searchResults"), limit)
    payload = envelope("search", "GET", request_url, params)
    payload["summary"] = {
        "searchText": data.get("searchText"),
        "totalItemCount": data.get("totalItemCount"),
        "pageSize": data.get("pageSize"),
        "resultPage": data.get("resultPage"),
        "itemsReturned": len(items),
        "copyrightNotice": "Do not republish Tagesschau article text unless content is explicitly CC-licensed.",
    }
    payload["items"] = items
    payload["sources"] = [{"kind": "api_request", "title": "Tagesschau API request", "url": request_url}] + default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_items(items)
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_article(command: str, argv: list[str], dossier: bool) -> None:
    parsed = parse_args(argv)
    input_url = first(parsed["flags"].get("url"), parsed["params"].get("url"), " ".join(parsed["positionals"]))
    if not input_url:
        raise CliError("missing_url", f"{command} requires --url", 2)
    api_url, public_url = article_urls(input_url)
    status, body, _content_type = fetch_raw(api_url)
    if not 200 <= status < 300:
        raise CliError("upstream_http_error", f"upstream status {status} from {api_url}: {strip_space(body)[:260]}")
    if flag_bool(parsed, "raw"):
        sys.stdout.write(body)
        return
    data = json.loads(body)
    grep = first(parsed["flags"].get("grep"), parsed["flags"].get("term"), parsed["flags"].get("q"))
    limit = limit_flag(parsed)
    snippets = article_snippets(data, grep, limit)
    summary = compact_article(data)
    summary["snippetCount"] = len(snippets)
    summary["snippets"] = snippets
    if dossier:
        summary["dossierUse"] = "Use as current-news context; verify institutional/statistical claims against primary official sources."
    payload = envelope(command, "GET", api_url, {"url": input_url, "grep": grep, "limit": limit})
    payload["summary"] = summary
    payload["items"] = snippets
    payload["sources"] = [{"kind": "api_request", "title": "Tagesschau article JSON", "url": api_url}, {"kind": "public_article", "title": "Tagesschau public article", "url": public_url}] + default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f'tagesschau article source --url "{public_url}"']
    if dossier:
        payload["nextActions"].append("tagesschau source")
    if flag_bool(parsed, "include-raw"):
        payload["raw"] = data
    emit(payload)


def run_article_source(argv: list[str]) -> None:
    parsed = parse_args(argv)
    input_url = first(parsed["flags"].get("url"), parsed["params"].get("url"), " ".join(parsed["positionals"]))
    if not input_url:
        raise CliError("missing_url", "article source requires --url", 2)
    api_url, public_url = article_urls(input_url)
    payload = envelope("article source", "GET", api_url, {"url": input_url})
    payload["summary"] = {
        "apiUrl": api_url,
        "publicUrl": public_url,
        "sourceType": "news_context",
        "reuseRestriction": "Do not republish article text except where explicitly CC-licensed.",
        "recommendedUse": "Cite headline/date/public URL; use short snippets only as needed for analysis.",
        "primaryEvidenceUse": False,
    }
    payload["sources"] = [{"kind": "api_request", "title": "Tagesschau article JSON", "url": api_url}, {"kind": "public_article", "title": "Tagesschau public article", "url": public_url}] + default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f'tagesschau article get --url "{public_url}" --limit 5']
    emit(payload)


def compact_feed_items(data: dict[str, Any], limit: int, include_regional: bool) -> list[dict[str, Any]]:
    items = compact_array(data.get("news"), limit)
    if include_regional and len(items) < limit:
        items.extend(compact_array(data.get("regional"), limit - len(items)))
    if not items:
        items.extend(compact_array(data.get("channels"), limit))
    return items[:limit]


def compact_array(value: Any, limit: int) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        return []
    return [compact_article(item) for item in value if isinstance(item, dict)][:limit]


def compact_article(obj: dict[str, Any]) -> dict[str, Any]:
    details = str(obj.get("details") or "")
    public_url = first(str(obj.get("detailsweb") or ""), str(obj.get("detailsWeb") or ""), str(obj.get("shareURL") or ""))
    if not public_url and details:
        _api, public_url = article_urls(details)
    item: dict[str, Any] = {
        "title": obj.get("title") or "",
        "topline": obj.get("topline") or "",
        "date": obj.get("date") or "",
        "type": obj.get("type") or "",
        "firstSentence": strip_html(str(obj.get("firstSentence") or "")),
        "sophoraId": obj.get("sophoraId") or "",
        "externalId": obj.get("externalId") or "",
        "details": details,
        "detailsweb": public_url,
        "shareURL": obj.get("shareURL") or "",
        "ressort": obj.get("ressort") or "",
        "tags": tag_strings(obj.get("tags")),
    }
    if public_url:
        item["sourceUrl"] = public_url
        item["nextActions"] = [f'tagesschau article get --url "{public_url}" --limit 5', f'tagesschau article source --url "{public_url}"']
    return item


def article_snippets(data: dict[str, Any], grep: str, limit: int) -> list[dict[str, Any]]:
    content = data.get("content")
    if not isinstance(content, list):
        return []
    needle = grep.lower()
    snippets = []
    for index, block in enumerate(content):
        if not isinstance(block, dict) or block.get("type") not in ("text", "headline"):
            continue
        text = strip_html(str(block.get("value") or ""))
        if not text:
            continue
        if needle and needle not in text.lower():
            continue
        snippets.append({"index": index, "type": block.get("type"), "text": truncate(text, 520), "matched": not needle or needle in text.lower()})
        if len(snippets) >= limit:
            break
    return snippets


def article_urls(input_url: str) -> tuple[str, str]:
    parsed = urllib.parse.urlparse(input_url.strip())
    if not parsed.scheme or not parsed.netloc:
        raise CliError("invalid_url", "expected an absolute Tagesschau URL", 2)
    if not parsed.netloc.endswith("tagesschau.de"):
        raise CliError("invalid_url", "expected a tagesschau.de URL", 2)
    path = parsed.path
    if path.startswith("/api2u/"):
        public_path = path.removeprefix("/api2u").removesuffix(".json") + ".html"
        return urllib.parse.urlunparse(("https", "www.tagesschau.de", path, "", "", "")), urllib.parse.urlunparse(("https", "www.tagesschau.de", public_path, "", "", ""))
    api_path = path.removesuffix(".html") + ".json"
    return urllib.parse.urlunparse(("https", "www.tagesschau.de", "/api2u" + api_path, "", "", "")), urllib.parse.urlunparse(("https", "www.tagesschau.de", path, "", "", ""))


def fetch_raw(request_url: str) -> tuple[int, str, str]:
    last_status = 0
    last_body = ""
    last_content_type = ""
    last_error: Exception | None = None
    for attempt in range(2):
        if attempt:
            time.sleep(0.75)
        req = urllib.request.Request(request_url, headers={"User-Agent": USER_AGENT, "Accept": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=35) as response:
                return response.status, response.read().decode("utf-8", errors="replace"), response.headers.get("Content-Type", "")
        except urllib.error.HTTPError as exc:
            last_status = exc.code
            last_body = exc.read().decode("utf-8", errors="replace")
            last_content_type = exc.headers.get("Content-Type", "")
            if exc.code not in (429, 502, 503, 504):
                return last_status, last_body, last_content_type
        except Exception as exc:
            last_error = exc
    if last_status:
        return last_status, last_body, last_content_type
    raise last_error or CliError("network_error", f"failed to fetch {request_url}")


def parse_args(argv: list[str]) -> dict[str, Any]:
    flags: dict[str, str] = {}
    params: dict[str, str] = {}
    positionals: list[str] = []
    index = 0
    while index < len(argv):
        token = argv[index]
        if token.startswith("--"):
            key = token[2:]
            if key == "param":
                index += 1
                if index >= len(argv) or "=" not in argv[index]:
                    raise CliError("invalid_param", "--param requires key=value", 2)
                param_key, param_value = argv[index].split("=", 1)
                params[param_key] = param_value
            elif index + 1 < len(argv) and not argv[index + 1].startswith("--"):
                flags[key] = argv[index + 1]
                index += 1
            else:
                flags[key] = "true"
        else:
            positionals.append(token)
        index += 1
    return {"flags": flags, "params": params, "positionals": positionals}


def envelope(command: str, method: str, request_url: str, params: Any) -> dict[str, Any]:
    return {"status": "ok", "tool": APP_NAME, "command": command, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "request": {"method": method, "url": request_url, "params": params}, "summary": {}, "items": [], "sources": [], "warnings": [], "nextActions": []}


def default_sources() -> list[dict[str, str]]:
    return [
        {"kind": "api_docs", "title": "bundesAPI Tagesschau API documentation", "url": API_DOCS_URL},
        {"kind": "openapi", "title": "Tagesschau OpenAPI YAML", "url": OPENAPI_URL},
        {"kind": "public_service", "title": "tagesschau.de", "url": BASE_URL + "/"},
        {"kind": "usage", "title": "Tagesschau RSS and reuse notice", "url": RSS_INFO_URL},
        {"kind": "license", "title": "Creative Commons videos", "url": CC_URL},
    ]


def default_warnings() -> list[str]:
    return [
        "Published API documentation says not to make more than 60 requests per hour.",
        "Tagesschau content use is private/non-commercial; publication is not allowed except for content explicitly under Creative Commons.",
        "Use this as current-news context, not as the only evidence for official parliamentary, legal, fiscal, or statistical claims.",
        "Avoid reproducing long article text; cite the public article URL and use short snippets only when needed.",
    ]


def feed_counts(data: dict[str, Any]) -> dict[str, int]:
    return {key: len(data[key]) for key in ("news", "regional", "channels", "searchResults") if isinstance(data.get(key), list)}


def next_actions_from_items(items: list[dict[str, Any]]) -> list[str]:
    actions = [f'tagesschau article get --url "{item["sourceUrl"]}" --limit 5' for item in items[:3] if item.get("sourceUrl")]
    return actions or ["tagesschau source"]


def with_params(base: str, params: dict[str, str]) -> str:
    return base + ("?" + urllib.parse.urlencode(params) if params else "")


def tag_strings(value: Any) -> list[str]:
    if not isinstance(value, list):
        return []
    tags = [str(item.get("tag")) for item in value if isinstance(item, dict) and item.get("tag")]
    return sorted(tags)


TAG_PATTERN = re.compile(r"<[^>]+>")
SPACE_PATTERN = re.compile(r"\s+")


def strip_html(value: str) -> str:
    value = value.replace("<br />", " ").replace("<br/>", " ")
    value = TAG_PATTERN.sub(" ", value)
    return strip_space(html.unescape(value))


def strip_space(value: str) -> str:
    return SPACE_PATTERN.sub(" ", value).strip()


def truncate(value: str, max_length: int) -> str:
    return value if len(value) <= max_length else value[:max_length] + "..."


def first(*values: Any) -> str:
    for value in values:
        if value is not None and str(value).strip():
            return str(value).strip()
    return ""


def flag_bool(parsed: dict[str, Any], key: str) -> bool:
    return str(parsed["flags"].get(key, "")).lower() in ("1", "true", "yes", "on")


def limit_flag(parsed: dict[str, Any]) -> int:
    value = parsed["flags"].get("limit")
    if not value:
        return DEFAULT_LIMIT
    try:
        return max(0, min(int(value), MAX_LIMIT))
    except ValueError as exc:
        raise CliError("invalid_limit", "--limit must be an integer", 2) from exc


def is_help(value: str) -> bool:
    return value in ("--help", "-h", "help")


def match(args: list[str], *expected: str) -> bool:
    return len(args) >= len(expected) and all(args[index] == expected[index] for index in range(len(expected)))


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=True, indent=2))


def emit_error(code: str, message: str) -> None:
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
