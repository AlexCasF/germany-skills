#!/usr/bin/env python3
"""Agent-friendly Bundeshaushalt Digital CLI mirror."""

from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from collections import deque
from typing import Any


APP_NAME = "bundeshaushalt"
BASE_URL = "https://bundeshaushalt.de"
BUDGET_DATA_URL = f"{BASE_URL}/internalapi/budgetData"
DIGITAL_URL = "https://www.bundeshaushalt.de/DE/Bundeshaushalt-digital/bundeshaushalt-digital.html"
USER_NOTES_URL = "https://www.bundeshaushalt.de/DE/Service/Benutzerhinweise/benutzerhinweise.html"
ROBOTS_URL = "https://www.bundeshaushalt.de/robots.txt"
BMF_BUDGET_URL = "https://www.bundesfinanzministerium.de/Web/DE/Themen/Oeffentliche_Finanzen/Bundeshaushalt/bundeshaushalt.html"
BMF_DATA_USE_URL = "https://www.bundesfinanzministerium.de/Datenportal/Nutzungshinweise/nutzungshinweise.html"
OPENAPI_WRAPPER_URL = "https://github.com/anetz89/bundeshaushalt-api"
USER_AGENT = "germany-skills/bundeshaushalt-python-2.0"

KNOWN_YEARS = list(range(2012, 2027))
EARLIEST_KNOWN_YEAR = 2012
LATEST_TARGET_YEAR = 2026
LATEST_ACTUAL_YEAR = 2024
DEFAULT_LIMIT = 10
SAFE_LIMIT = 100
DEFAULT_SEARCH_DEPTH = 3


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
        elif argv[0] == "fields":
            run_fields(argv[1:])
        elif argv[0] == "source":
            run_source(argv[1:])
        elif match(argv, "years", "list"):
            run_years_list(argv[2:])
        elif match(argv, "budget", "tree"):
            run_budget_tree(argv[2:])
        elif match(argv, "budget", "sample") or argv[0] == "sample":
            run_sample(argv[2:] if match(argv, "budget", "sample") else argv[1:])
        elif match(argv, "title", "get"):
            run_title_get(argv[2:])
        elif argv[0] == "search":
            run_search(argv[1:])
        elif argv[0] == "compare":
            run_compare(argv[1:])
        elif argv[0] == "budget-data":
            run_budget_data(argv[1:])
        else:
            raise CliError("unknown_command", f"unknown command: {' '.join(argv)}", 2)
        return 0
    except CliError as exc:
        emit_error(exc.code, exc.message)
        return exc.exit_code
    except Exception as exc:  # pragma: no cover - safety net for shell use
        emit_error("unexpected_error", str(exc))
        return 1


def print_root_help() -> None:
    print(
        """bundeshaushalt 2.0 - Bundeshaushalt Digital research CLI

Usage:
  bundeshaushalt doctor
  bundeshaushalt years list
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt search --year 2025 --account expenses --term "Buergergeld" --limit 5
  bundeshaushalt title get --year 2025 --account expenses --id 110168112
  bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112
  bundeshaushalt budget-data --year 2025 --account expenses --quota target --unit single --raw

Research commands:
  doctor          Check endpoint health, auth, live-year behavior, and fair-use hints.
  years list      Show known years and likely target/actual availability.
  fields          Explain account, quota, unit, hierarchy, and value fields.
  source          Print canonical source, API, attribution, and terms URLs.
  budget tree     Fetch a hierarchy node with compact children and next actions.
  budget sample   Fetch a tiny representative tree sample.
  search          Traverse labels safely to find budget nodes by term.
  title get       Fetch one exact budget node by internal id.
  compare         Compare the same node across multiple years.

Compatibility command:
  budget-data     Direct endpoint wrapper with --param key=value support.

JSON is the default output. Use --raw on endpoint-style commands to emit upstream JSON.
"""
    )


def print_help(args: list[str]) -> None:
    if not args:
        print_root_help()
        return
    if args[0] == "budget-data":
        print("budget-data flags: --year --account expenses|income --quota target|actual --unit single|function|group --id --param key=value --raw")
    elif args[0] == "search":
        print("search flags: --year --account --quota --unit --term/--q --depth --max-requests --limit --include-raw")
    elif match(args, "budget", "tree"):
        print("budget tree flags: --year --account --quota --unit --id --limit --grep --include-raw --raw")
    elif args[0] == "compare":
        print("compare flags: --years 2024,2025 --account --quota --unit --id")
    else:
        print_root_help()


def print_examples() -> None:
    print(
        """Examples:
  bundeshaushalt doctor
  bundeshaushalt source
  bundeshaushalt years list
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt search --year 2025 --account expenses --term "Arbeit" --limit 5
  bundeshaushalt title get --year 2025 --account expenses --id 110168112
  bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112
"""
    )


def run_doctor(argv: list[str]) -> None:
    checks: list[dict[str, Any]] = []
    for name, params in [
        ("latestTargetExpenses", {"year": str(LATEST_TARGET_YEAR), "account": "expenses", "quota": "target", "unit": "single"}),
        ("latestActualExpenses", {"year": str(LATEST_ACTUAL_YEAR), "account": "expenses", "quota": "actual", "unit": "single"}),
    ]:
        url = with_params(BUDGET_DATA_URL, params)
        try:
            status, body = fetch_raw(url)
            data = json.loads(body) if status < 300 else {}
            checks.append({"name": name, "ok": status < 300, "url": url, "bodyBytes": len(body), "meta": data.get("meta")})
        except Exception as exc:
            checks.append({"name": name, "ok": False, "url": url, "error": str(exc)})
    payload = envelope("doctor", BUDGET_DATA_URL, {})
    payload["summary"] = {
        "authRequired": False,
        "publishedRateLimit": "No exact public request quota was found. robots.txt publishes Crawl-delay: 30 for crawling-style workflows.",
        "endpointBehavior": "GET /internalapi/budgetData requires at least year and account; some actual values return 404 until accounting data exists.",
        "checks": checks,
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [
        "bundeshaushalt years list",
        f"bundeshaushalt budget tree --year {LATEST_TARGET_YEAR} --account expenses --quota target --limit 8",
        'bundeshaushalt search --year 2025 --account expenses --term "Arbeit" --limit 5',
    ]
    emit(payload)


def run_fields(argv: list[str]) -> None:
    payload = envelope("fields", BUDGET_DATA_URL, {})
    payload["summary"] = {
        "accounts": [{"value": "expenses", "meaning": "Ausgaben"}, {"value": "income", "meaning": "Einnahmen"}],
        "quotas": [{"value": "target", "meaning": "Soll/plan"}, {"value": "actual", "meaning": "Ist/accounting data"}],
        "units": [
            {"value": "single", "meaning": "Einzelplan/ministry and title hierarchy"},
            {"value": "function", "meaning": "Functional classification"},
            {"value": "group", "meaning": "Revenue/expenditure group classification"},
        ],
        "coreFields": ["id", "budgetNumber", "label", "value", "relativeToParentValue", "relativeValue"],
        "valueUnits": "Nominal euro amounts; helper fields expose valueEur and valueBillionEur.",
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundeshaushalt budget tree --year {LATEST_TARGET_YEAR} --account expenses --limit 8"]
    emit(payload)


def run_source(argv: list[str]) -> None:
    payload = envelope("source", BUDGET_DATA_URL, {})
    payload["summary"] = {
        "publisher": "Bundesministerium der Finanzen / Bundeshaushalt Digital",
        "authRequired": False,
        "rateLimit": "No exact public quota found; robots.txt contains Crawl-delay: 30.",
        "knownEndpoint": BUDGET_DATA_URL,
        "knownYears": {"earliest": EARLIEST_KNOWN_YEAR, "latestTarget": LATEST_TARGET_YEAR, "latestActual": LATEST_ACTUAL_YEAR},
    }
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["bundeshaushalt fields", "bundeshaushalt years list"]
    emit(payload)


def run_years_list(argv: list[str]) -> None:
    items = [
        {
            "year": year,
            "targetLikely": True,
            "actualLikely": year <= LATEST_ACTUAL_YEAR,
            "exampleTargetCmd": f"bundeshaushalt budget tree --year {year} --account expenses --quota target --limit 8",
        }
        for year in KNOWN_YEARS
    ]
    payload = envelope("years list", BUDGET_DATA_URL, {})
    payload["summary"] = {
        "count": len(items),
        "earliestKnownYear": EARLIEST_KNOWN_YEAR,
        "latestTargetYear": LATEST_TARGET_YEAR,
        "latestActualYear": LATEST_ACTUAL_YEAR,
        "note": "Known from live endpoint probes; the old OpenAPI enum stops at 2021.",
    }
    payload["items"] = items
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundeshaushalt budget tree --year {LATEST_TARGET_YEAR} --account expenses --quota target --limit 8"]
    emit(payload)


def run_budget_tree(argv: list[str]) -> None:
    parsed = parse_args(argv)
    params = budget_params(parsed, require_year=True)
    raw, data, request_url = fetch_budget(params)
    if flag_bool(parsed, "raw"):
        sys.stdout.write(raw)
        return
    emit_budget_envelope("budget tree", request_url, params, data, parsed)


def run_sample(argv: list[str]) -> None:
    parsed = parse_args(argv)
    parsed["flags"].setdefault("year", str(LATEST_TARGET_YEAR))
    parsed["flags"].setdefault("limit", "5")
    params = budget_params(parsed, require_year=True)
    raw, data, request_url = fetch_budget(params)
    emit_budget_envelope("budget sample", request_url, params, data, parsed)


def run_title_get(argv: list[str]) -> None:
    parsed = parse_args(argv)
    if not first(parsed["flags"].get("id"), parsed["params"].get("id")):
        raise CliError("missing_id", "title get requires --id", 2)
    params = budget_params(parsed, require_year=True)
    raw, data, request_url = fetch_budget(params)
    if flag_bool(parsed, "raw"):
        sys.stdout.write(raw)
        return
    emit_budget_envelope("title get", request_url, params, data, parsed)


def run_search(argv: list[str]) -> None:
    parsed = parse_args(argv)
    term = first(parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CliError("missing_term", "search requires --term", 2)
    params = budget_params(parsed, require_year=True)
    params.pop("id", None)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    depth = int_flag(parsed, "depth", DEFAULT_SEARCH_DEPTH, 6)
    max_requests = int_flag(parsed, "max-requests", 60, 250)
    items, requests = search_hierarchy(params, term, depth, max_requests, limit, flag_bool(parsed, "include-raw"))
    payload = envelope("search", BUDGET_DATA_URL, {"year": params.get("year"), "account": params.get("account"), "quota": params.get("quota"), "unit": params.get("unit"), "term": term, "depth": depth, "maxRequests": max_requests, "limit": limit})
    payload["summary"] = {"term": term, "returned": len(items), "requestsUsed": requests, "requestCap": max_requests, "traversalDepth": depth}
    payload["items"] = items
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = next_actions_from_search(items, params)
    emit(payload)


def run_compare(argv: list[str]) -> None:
    parsed = parse_args(argv)
    raw_years = first(parsed["flags"].get("years"), parsed["params"].get("years"), parsed["flags"].get("year"), parsed["params"].get("year"))
    if not raw_years:
        raise CliError("missing_years", "compare requires --years 2024,2025", 2)
    years = [int(part.strip()) for part in raw_years.replace(";", ",").split(",") if part.strip()]
    if len(years) < 2:
        raise CliError("missing_years", "compare needs at least two years", 2)
    base_params = budget_params(parsed, require_year=False)
    base_params.pop("year", None)
    items: list[dict[str, Any]] = []
    for year in years:
        params = dict(base_params)
        params["year"] = str(year)
        try:
            _raw, data, request_url = fetch_budget(params)
            items.append({"year": year, "ok": True, "requestUrl": request_url, "meta": data.get("meta"), "detail": compact_element(data.get("detail") or {}, data.get("meta") or {}, params), "childCount": len(data.get("children") or [])})
        except Exception as exc:
            items.append({"year": year, "ok": False, "error": str(exc), "requestUrl": with_params(BUDGET_DATA_URL, params)})
    payload = envelope("compare", BUDGET_DATA_URL, {"years": years, "account": base_params.get("account"), "quota": base_params.get("quota"), "unit": base_params.get("unit"), "id": base_params.get("id")})
    payload["summary"] = {"years": years, "returned": len(items), "account": base_params.get("account"), "quota": base_params.get("quota"), "unit": base_params.get("unit"), "id": base_params.get("id")}
    payload["items"] = items
    payload["status"] = "ok" if all(item.get("ok") for item in items) else "partial"
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["bundeshaushalt source"]
    emit(payload)


def run_budget_data(argv: list[str]) -> None:
    parsed = parse_args(argv)
    params = budget_params(parsed, require_year=False, require_account=False)
    if "year" not in params or "account" not in params:
        raise CliError("missing_required_params", "budget-data requires --year and --account (or --param year=... --param account=...)", 2)
    validate_budget_params(params)
    raw, data, request_url = fetch_budget(params)
    if flag_bool(parsed, "raw"):
        sys.stdout.write(raw)
        return
    emit_budget_envelope("budget-data", request_url, params, data, parsed)


def emit_budget_envelope(command: str, request_url: str, params: dict[str, str], data: dict[str, Any], parsed: dict[str, Any]) -> None:
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    grep = parsed["flags"].get("grep", "").lower()
    children = data.get("children") or []
    items = [compact_element(child, data.get("meta") or {}, params) for child in children]
    if grep:
        items = [item for item in items if grep in str(item.get("label", "")).lower()]
    if limit >= 0:
        items = items[:limit]
    payload = envelope(command, request_url, request_params(params, parsed))
    payload["summary"] = {
        "meta": data.get("meta"),
        "detail": compact_element(data.get("detail") or {}, data.get("meta") or {}, params),
        "childrenTotal": len(children),
        "childrenShown": len(items),
        "parentsLevels": len(data.get("parents") or []),
        "relatedKeys": sorted((data.get("related") or {}).keys()) or None,
    }
    payload["items"] = items
    payload["sources"] = [{"kind": "api_request", "title": "Bundeshaushalt API request", "url": request_url}] + default_sources()
    payload["warnings"] = default_warnings_for_response(data)
    payload["nextActions"] = next_actions_from_children(items)
    emit(payload)


def search_hierarchy(params: dict[str, str], term: str, depth_limit: int, max_requests: int, limit: int, include_raw: bool) -> tuple[list[dict[str, Any]], int]:
    needle = term.lower()
    queue = deque([{"id": "", "depth": 0}])
    seen = set()
    out: list[dict[str, Any]] = []
    requests = 0
    while queue and requests < max_requests and len(out) < limit:
        node = queue.popleft()
        node_id = node["id"]
        node_depth = node["depth"]
        if node_id in seen or node_depth > depth_limit:
            continue
        seen.add(node_id)
        request_params = dict(params)
        if node_id:
            request_params["id"] = node_id
        try:
            raw, data, request_url = fetch_budget(request_params)
            requests += 1
        except Exception:
            requests += 1
            continue
        detail = data.get("detail") or {}
        if matches_element(detail, needle):
            item = compact_element(detail, data.get("meta") or {}, request_params)
            item["matchType"] = "detail"
            item["requestUrl"] = request_url
            if include_raw:
                item["raw"] = data
            out.append(item)
            if len(out) >= limit:
                break
        for child in data.get("children") or []:
            child_id = str(child.get("id") or "")
            if matches_element(child, needle):
                item = compact_element(child, data.get("meta") or {}, params)
                item["matchType"] = "child"
                item["parentId"] = node_id
                item["parentLabel"] = detail.get("label")
                if include_raw:
                    item["raw"] = child
                out.append(item)
                if len(out) >= limit:
                    break
            if child_id and node_depth + 1 <= depth_limit:
                queue.append({"id": child_id, "depth": node_depth + 1})
    return out, requests


def compact_element(element: dict[str, Any], meta: dict[str, Any], params: dict[str, str]) -> dict[str, Any]:
    value = float(element.get("value") or 0)
    item = {
        "id": element.get("id") or "",
        "budgetNumber": element.get("budgetNumber") or "",
        "label": element.get("label") or "",
        "value": value,
        "valueEur": value,
        "valueBillionEur": value / 1_000_000_000,
        "relativeToParentValue": element.get("relativeToParentValue"),
        "relativeValue": element.get("relativeValue"),
        "tableLabel": element.get("tableLabel") or "",
        "selectionLabel": element.get("selectionLabel") or "",
        "year": meta.get("year"),
        "account": meta.get("account") or params.get("account"),
        "quota": meta.get("quota") or params.get("quota"),
        "unit": meta.get("unit") or params.get("unit"),
        "entity": meta.get("entity"),
        "levelCur": meta.get("levelCur"),
        "levelMax": meta.get("levelMax"),
    }
    if item["id"]:
        item["nextActions"] = [
            f"bundeshaushalt title get --year {item['year']} --account {item['account']} --quota {item['quota']} --unit {item['unit']} --id {item['id']}",
            f"bundeshaushalt budget tree --year {item['year']} --account {item['account']} --quota {item['quota']} --unit {item['unit']} --id {item['id']} --limit 10",
        ]
    return item


def fetch_budget(params: dict[str, str]) -> tuple[str, dict[str, Any], str]:
    request_url = with_params(BUDGET_DATA_URL, params)
    status, body = fetch_raw(request_url)
    if status < 200 or status >= 300:
        raise CliError("upstream_http_error", f"upstream status {status} from {request_url}: {strip_space(body)[:300]}")
    return body, json.loads(body), request_url


def fetch_raw(url: str) -> tuple[int, str]:
    last_status = 0
    last_body = ""
    last_error: Exception | None = None
    for attempt in range(3):
        if attempt:
            time.sleep(attempt * 0.75)
        req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT, "Accept": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=45) as response:
                body = response.read().decode("utf-8", errors="replace")
                return response.status, body
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            last_status = exc.code
            last_body = body
            if exc.code not in (429, 502, 503, 504):
                return last_status, last_body
        except Exception as exc:
            last_error = exc
    if last_status:
        return last_status, last_body
    raise last_error or CliError("network_error", f"failed to fetch {url}")


def budget_params(parsed: dict[str, Any], require_year: bool, require_account: bool = False) -> dict[str, str]:
    params = dict(parsed["params"])
    for key in ["year", "account", "quota", "unit", "id"]:
        value = parsed["flags"].get(key)
        if value:
            params[key] = value
    params.setdefault("quota", "target")
    params.setdefault("unit", "single")
    if require_year and "year" not in params:
        raise CliError("missing_year", "command requires --year", 2)
    if require_account and "account" not in params:
        raise CliError("missing_account", "command requires --account", 2)
    params.setdefault("account", "expenses")
    validate_budget_params(params)
    return {key: str(value) for key, value in params.items() if str(value).strip()}


def validate_budget_params(params: dict[str, str]) -> None:
    if "year" in params:
        try:
            year = int(params["year"])
        except ValueError as exc:
            raise CliError("invalid_year", "year must be a four-digit year", 2) from exc
        if year < 2000 or year > 2100:
            raise CliError("invalid_year", "year must be a plausible four-digit year", 2)
    if params.get("account") not in (None, "", "expenses", "income"):
        raise CliError("invalid_account", "account must be expenses or income", 2)
    if params.get("quota") not in (None, "", "target", "actual"):
        raise CliError("invalid_quota", "quota must be target or actual", 2)
    if params.get("unit") not in (None, "", "single", "function", "group"):
        raise CliError("invalid_unit", "unit must be single, function, or group", 2)


def parse_args(argv: list[str]) -> dict[str, Any]:
    flags: dict[str, str] = {}
    params: dict[str, str] = {}
    positionals: list[str] = []
    i = 0
    while i < len(argv):
        token = argv[i]
        if token.startswith("--"):
            key = token[2:]
            if key == "param":
                i += 1
                if i >= len(argv) or "=" not in argv[i]:
                    raise CliError("invalid_param", "--param requires key=value", 2)
                param_key, param_value = argv[i].split("=", 1)
                params[param_key] = param_value
            elif i + 1 < len(argv) and not argv[i + 1].startswith("--"):
                flags[key] = argv[i + 1]
                i += 1
            else:
                flags[key] = "true"
        else:
            positionals.append(token)
        i += 1
    return {"flags": flags, "params": params, "positionals": positionals}


def envelope(command: str, request_url: str, request: Any) -> dict[str, Any]:
    return {
        "status": "ok",
        "tool": APP_NAME,
        "command": command,
        "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "request": {"method": "GET", "url": request_url, "params": request or {}},
        "summary": {},
        "items": [],
        "sources": [],
        "warnings": [],
        "nextActions": [],
    }


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=True, indent=2, sort_keys=False))


def emit_error(code: str, message: str) -> None:
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})


def default_sources() -> list[dict[str, str]]:
    return [
        {"kind": "official_application", "title": "Bundeshaushalt Digital", "url": DIGITAL_URL},
        {"kind": "api_endpoint", "title": "Bundeshaushalt internal API endpoint", "url": BUDGET_DATA_URL},
        {"kind": "official_context", "title": "BMF Bundeshaushalt overview", "url": BMF_BUDGET_URL},
        {"kind": "terms", "title": "BMF Datenportal usage notes", "url": BMF_DATA_USE_URL},
        {"kind": "terms", "title": "Bundeshaushalt user notes", "url": USER_NOTES_URL},
        {"kind": "fair_use", "title": "Bundeshaushalt robots.txt", "url": ROBOTS_URL},
        {"kind": "openapi_reference", "title": "OpenAPI wrapper", "url": OPENAPI_WRAPPER_URL},
    ]


def default_warnings() -> list[str]:
    return [
        "No exact public rate limit for the Bundeshaushalt Digital API was found; robots.txt publishes Crawl-delay: 30 for crawling-like workflows.",
        "Actual/Ist values are only available after accounting data exists; newer years can return 404 for quota=actual.",
        "The old OpenAPI enum is stale and stops at 2021; live endpoint checks show newer target years are available.",
        "Budget values are nominal euro amounts; use statistical APIs for inflation, population, or macroeconomic context.",
        "Use BMF attribution and preserve dataset/page URLs in final citations.",
    ]


def default_warnings_for_response(data: dict[str, Any]) -> list[str]:
    warnings = default_warnings()
    meta = data.get("meta") or {}
    if meta.get("quota") == "actual" and int(meta.get("year") or 0) > LATEST_ACTUAL_YEAR:
        warnings.append("This actual/Ist year is newer than the latest actual year observed during testing; verify availability carefully.")
    if meta.get("unit") in ("function", "group"):
        warnings.append("Function and group views classify titles differently from Einzelplan ministry structure; do not mix categories without saying so.")
    return warnings


def next_actions_from_children(items: list[dict[str, Any]]) -> list[str]:
    actions = ["bundeshaushalt source"]
    for item in items[:3]:
        if item.get("id"):
            actions.append(f"bundeshaushalt budget tree --year {item.get('year')} --account {item.get('account')} --quota {item.get('quota')} --unit {item.get('unit')} --id {item.get('id')} --limit 10")
    return actions


def next_actions_from_search(items: list[dict[str, Any]], params: dict[str, str]) -> list[str]:
    actions = []
    for item in items[:3]:
        if item.get("id"):
            actions.append(f"bundeshaushalt title get --year {item.get('year')} --account {item.get('account')} --quota {item.get('quota')} --unit {item.get('unit')} --id {item.get('id')}")
    if not actions:
        actions.append(f"bundeshaushalt budget tree --year {params.get('year')} --account {params.get('account')} --limit 8")
    return actions


def with_params(base: str, params: dict[str, str]) -> str:
    return base + "?" + urllib.parse.urlencode({key: value for key, value in params.items() if str(value).strip()})


def request_params(params: dict[str, str], parsed: dict[str, Any]) -> dict[str, Any]:
    out: dict[str, Any] = dict(params)
    out["limit"] = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    out["grep"] = parsed["flags"].get("grep", "")
    return out


def flag_bool(parsed: dict[str, Any], key: str) -> bool:
    return str(parsed["flags"].get(key, "")).lower() in ("1", "true", "yes", "on")


def int_flag(parsed: dict[str, Any], key: str, default: int, max_value: int) -> int:
    value = parsed["flags"].get(key)
    if not value:
        return default
    try:
        number = int(value)
    except ValueError as exc:
        raise CliError("invalid_integer", f"--{key} must be an integer", 2) from exc
    return max(0, min(number, max_value))


def limit_flag(parsed: dict[str, Any], default: int, max_value: int) -> int:
    return int_flag(parsed, "limit", default, max_value)


def matches_element(element: dict[str, Any], needle: str) -> bool:
    return needle in f"{element.get('id', '')} {element.get('budgetNumber', '')} {element.get('label', '')}".lower()


def strip_space(value: str) -> str:
    return " ".join(value.split())


def first(*values: Any) -> str:
    for value in values:
        if value is not None and str(value).strip():
            return str(value).strip()
    return ""


def is_help(value: str) -> bool:
    return value in ("--help", "-h", "help")


def match(args: list[str], *expected: str) -> bool:
    return len(args) >= len(expected) and all(args[index] == expected[index] for index in range(len(expected)))


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
