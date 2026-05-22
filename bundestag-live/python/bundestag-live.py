#!/usr/bin/env python3
import html
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET

APP_NAME = "bundestag-live"
BASE_URL = "https://www.bundestag.de"
SPEAKER_URL = BASE_URL + "/static/appdata/plenum/v2/speaker.xml"
CONFERENCES_URL = BASE_URL + "/static/appdata/plenum/v2/conferences.xml"
COMMITTEES_URL = BASE_URL + "/xml/v2/ausschuesse/index.xml"
COMMITTEE_URL = BASE_URL + "/xml/v2/ausschuesse/{id}.xml"
MEMBERS_URL = BASE_URL + "/xml/v2/mdb/index.xml"
MEMBER_URL = BASE_URL + "/xml/v2/mdb/biografien/{id}.xml"
ARTICLE_URL = BASE_URL + "/blueprint/servlet/content/{id}/asAppV2NewsarticleXml"
VIDEO_URL = "http://webtv.bundestag.de/iptv/player/macros/_x_s-144277506/bttv/mobile/feed_vod.xml"
OPENAPI_URL = "https://github.com/bundesAPI/bundestag-api"
OPEN_DATA_URL = BASE_URL + "/services/opendata"
IMPRINT_URL = BASE_URL + "/services/impressum"
MEDIA_TERMS_URL = BASE_URL + "/mediathek/nutzungsbedingungen-247892"
PRIVACY_URL = BASE_URL + "/en/service/privacy"
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
        elif argv[0] == "examples":
            print_examples()
        elif argv[:2] == ["plenum", "speaker"]:
            run_plenum_speaker(argv[2:])
        elif argv[:2] in (["plenum", "conferences"], ["plenum", "agenda"]):
            run_plenum_conferences(argv[2:])
        elif argv[:2] == ["members", "list"]:
            run_members_list(argv[2:])
        elif argv[:2] == ["members", "search"]:
            run_members_search(argv[2:])
        elif argv[:2] == ["members", "biography"]:
            run_member_biography(argv[2:])
        elif argv[:2] == ["members", "dossier"]:
            run_member_dossier(argv[2:])
        elif argv[:2] == ["committees", "list"]:
            run_committees_list(argv[2:])
        elif argv[:2] == ["committees", "search"]:
            run_committees_search(argv[2:])
        elif argv[:2] in (["committees", "get"], ["committees", "dossier"]):
            run_committee_dossier(argv[2:])
        elif argv[:2] == ["article", "get"]:
            run_article_get(argv[2:])
        elif argv[:2] == ["article", "page"]:
            run_article_page(argv[2:])
        elif argv[:2] == ["video", "feed"]:
            run_video_feed(argv[2:])
        elif argv[0] == "source":
            run_source(argv[1:])
        else:
            raise CLIError(2, "unknown_command", "unknown command; run bundestag-live --help")
    except CLIError as exc:
        fail(exc.exit_code, exc.code, exc.message)
    except Exception as exc:
        fail(1, "unexpected_error", str(exc))
    return 0


def print_root_help():
    print("""bundestag-live -- Bundestag live/site XML research CLI

Purpose
  Discover and normalize public Bundestag live/site XML feeds for current
  plenary agenda data, members, biographies, committees, articles, and video
  feed metadata.

Fast paths
  bundestag-live doctor
  bundestag-live members search --name "Amthor" --limit 3
  bundestag-live members dossier --name "Amthor" --grep "TÃ¤tigkeiten"
  bundestag-live committees search --term "Arbeit" --limit 5
  bundestag-live committees dossier --id a11 --member-limit 5
  bundestag-live plenum conferences --limit 2 --item-limit 3
  bundestag-live article get --article-id 1174778

Endpoint-compatible commands
  plenum speaker
  plenum conferences
  committees list
  committees get --id a11
  members list
  members biography --id 2022
  article get --article-id 1174778
  video feed --content-id 7529016
""")


def print_help(path):
    joined = " ".join(path)
    if joined == "members search":
        print('bundestag-live members search --name "Amthor" --limit 3')
    elif joined == "members dossier":
        print('bundestag-live members dossier --id 2022 --grep "TÃ¤tigkeiten"')
    elif joined == "committees dossier":
        print("bundestag-live committees dossier --id a11 --member-limit 5 --news-limit 3")
    elif joined == "article page":
        print('bundestag-live article page --url "https://www.bundestag.de/..." --grep "term"')
    else:
        print_root_help()


def print_examples():
    print("""bundestag-live examples

1. bundestag-live doctor
2. bundestag-live members search --name "Amthor" --limit 3
3. bundestag-live members dossier --id 2022 --grep "TÃ¤tigkeiten"
4. bundestag-live committees search --term "Arbeit" --limit 5
5. bundestag-live committees dossier --id a11 --member-limit 5 --news-limit 3
6. bundestag-live plenum conferences --limit 2 --item-limit 5
7. bundestag-live article get --article-id 1174778
8. bundestag-live article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778" --grep "Meinungsfreiheit"
9. bundestag-live members biography --id 2022 --raw
10. Use dip-bundestag for full parliamentary proceedings and historical protocol research.
""")


def run_doctor(argv):
    parsed = parse_args(argv)
    limit = limit_flag(parsed, 3, 10)
    checks = [("speaker", SPEAKER_URL), ("conferences", CONFERENCES_URL), ("committees", COMMITTEES_URL), ("members", MEMBERS_URL)]
    summary = {
        "authRequired": False,
        "publishedRateLimit": "No exact published request quota was found for these public Bundestag XML feeds. Use small limits, cache repeated index calls, and back off on 429/5xx responses.",
        "fairUseHints": [
            "Use search commands before fetching detail records.",
            "Avoid repeated full member index downloads during one run.",
            "Use --limit and --item-limit on broad feeds.",
            "Treat video/media URLs under Bundestag media terms.",
        ],
        "endpoints": [],
    }
    status = "ok"
    for name, url in checks[:limit]:
        code, content_type, body = fetch_raw(url)
        ok = 200 <= code < 300
        if not ok:
            status = "degraded"
        summary["endpoints"].append({"name": name, "url": url, "statusCode": code, "contentType": content_type, "ok": ok, "bodyPreview": truncate(strip_space(body), 180)})
    payload = envelope("doctor", BASE_URL, {"limit": limit})
    payload["status"] = status
    payload["summary"] = summary
    payload["sources"] = default_sources()
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['bundestag-live members search --name "Amthor" --limit 3', "bundestag-live committees search --term Arbeit --limit 5"]
    emit(payload)


def run_members_list(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_xml_with_params(MEMBERS_URL, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    members = member_items(root)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    payload = envelope("members list", request_url, {"limit": limit})
    payload["summary"] = {"totalMembers": len(members), "returned": min(limit, len(members)), "documentStand": find_text(root, "dokumentInfo/dokumentStand")}
    payload["items"] = [compact_member(item, flag_bool(parsed, "include-raw")) for item in members[:limit]]
    payload["sources"] = source("Bundestag member XML index", MEMBERS_URL, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ['bundestag-live members search --name "Amthor" --limit 3']
    emit(payload)


def run_members_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("name"), parsed["flags"].get("term"), parsed["flags"].get("q"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "members search requires --name or --term")
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    body, request_url = fetch_xml_with_params(MEMBERS_URL, {})
    root = ET.fromstring(body)
    members = member_items(root)
    matches = [item for item in members if term.lower() in member_search_text(item).lower()]
    payload = envelope("members search", request_url, {"term": term, "limit": limit})
    payload["summary"] = {"term": term, "matches": len(matches), "returned": min(limit, len(matches)), "searchedMembers": len(members), "documentStand": find_text(root, "dokumentInfo/dokumentStand")}
    payload["items"] = [compact_member(item, flag_bool(parsed, "include-raw")) for item in matches[:limit]]
    payload["sources"] = source("Bundestag member XML index", MEMBERS_URL, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundestag-live members dossier --id {item['id']}" for item in matches[:3]] or ['bundestag-live members search --name "Amthor" --limit 3']
    emit(payload)


def run_member_biography(argv):
    parsed = parse_args(argv)
    member_id = first_non_empty(parsed["flags"].get("id"), parsed["positionals"][0] if parsed["positionals"] else "")
    if not member_id:
        raise CLIError(2, "missing_id", "members biography requires --id")
    url = MEMBER_URL.format(id=urllib.parse.quote(member_id))
    body, request_url = fetch_xml_with_params(url, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    payload = envelope("members biography", request_url, {"id": member_id})
    payload["summary"] = compact_biography(root, parsed["flags"].get("grep", ""))
    payload["items"] = [member_evidence(root, parsed["flags"].get("grep", ""))]
    payload["sources"] = sources_for_member(root, request_url)
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundestag-live members dossier --id {member_id} --grep TÃ¤tigkeiten"]
    if flag_bool(parsed, "include-raw"):
        payload["rawXml"] = body
    emit(payload)


def run_member_dossier(argv):
    parsed = parse_args(argv)
    member_id = parsed["flags"].get("id", "")
    resolved = None
    if not member_id:
        name = first_non_empty(parsed["flags"].get("name"), parsed["flags"].get("term"), " ".join(parsed["positionals"]))
        if not name:
            raise CLIError(2, "missing_member", "members dossier requires --id or --name")
        resolved = resolve_member(name)
        member_id = resolved["id"]
    url = MEMBER_URL.format(id=urllib.parse.quote(member_id))
    body, request_url = fetch_xml_with_params(url, {})
    root = ET.fromstring(body)
    grep = parsed["flags"].get("grep", "")
    payload = envelope("members dossier", request_url, {"id": member_id, "name": parsed["flags"].get("name", ""), "grep": grep})
    payload["summary"] = compact_biography(root, grep)
    payload["items"] = [member_evidence(root, grep)]
    payload["sources"] = sources_for_member(root, request_url)
    payload["warnings"] = default_warnings() + ["Member biography and disclosure fields are based on Bundestag profile XML; disclosure text may reflect self-reported data and Bundestag publication rules."]
    payload["nextActions"] = [f"bundestag-live members biography --id {member_id} --raw"]
    if resolved:
        payload["resolvedFromIndex"] = compact_member(resolved, False)
    if flag_bool(parsed, "include-raw"):
        payload["rawXml"] = body
    emit(payload)


def run_committees_list(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_xml_with_params(COMMITTEES_URL, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    committees = committee_items(root)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    payload = envelope("committees list", request_url, {"limit": limit})
    payload["summary"] = {"totalCommittees": len(committees), "returned": min(limit, len(committees)), "documentStand": find_text(root, "dokumentInfo/dokumentStand")}
    payload["items"] = [compact_committee(item, flag_bool(parsed, "include-raw")) for item in committees[:limit]]
    payload["sources"] = source("Bundestag committee XML index", COMMITTEES_URL, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = ["bundestag-live committees search --term Arbeit --limit 5"]
    emit(payload)


def run_committees_search(argv):
    parsed = parse_args(argv)
    term = first_non_empty(parsed["flags"].get("term"), parsed["flags"].get("q"), parsed["flags"].get("name"), " ".join(parsed["positionals"]))
    if not term:
        raise CLIError(2, "missing_term", "committees search requires --term")
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    body, request_url = fetch_xml_with_params(COMMITTEES_URL, {})
    root = ET.fromstring(body)
    committees = committee_items(root)
    matches = [item for item in committees if term.lower() in committee_search_text(item).lower()]
    payload = envelope("committees search", request_url, {"term": term, "limit": limit})
    payload["summary"] = {"term": term, "matches": len(matches), "returned": min(limit, len(matches)), "searchedCommittees": len(committees)}
    payload["items"] = [compact_committee(item, flag_bool(parsed, "include-raw")) for item in matches[:limit]]
    payload["sources"] = source("Bundestag committee XML index", COMMITTEES_URL, "api_endpoint")
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundestag-live committees dossier --id {item['id']} --member-limit 5" for item in matches[:3]] or ["bundestag-live committees search --term Arbeit --limit 5"]
    emit(payload)


def run_committee_dossier(argv):
    parsed = parse_args(argv)
    committee_id = first_non_empty(parsed["flags"].get("id"), parsed["positionals"][0] if parsed["positionals"] else "")
    if not committee_id:
        raise CLIError(2, "missing_id", "committees dossier requires --id")
    body, request_url = fetch_xml_with_params(COMMITTEE_URL.format(id=urllib.parse.quote(committee_id)), parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    member_limit = limit_flag_name(parsed, "member-limit", DEFAULT_LIMIT, SAFE_LIMIT)
    news_limit = limit_flag_name(parsed, "news-limit", 5, 50)
    grep = parsed["flags"].get("grep", "")
    members = committee_members(root)
    news = committee_news(root, grep)
    payload = envelope("committees get", request_url, {"id": committee_id, "memberLimit": member_limit, "newsLimit": news_limit})
    payload["summary"] = {"id": find_text(root, "ausschussId"), "name": find_text(root, "ausschussName"), "sourceUrl": find_text(root, "ausschussSourceURL"), "chairId": find_text(root, "ausschussVorsitzId"), "memberCount": len(members), "newsCount": len(news), "taskSnippets": grep_snippets(strip_html(find_text(root, "ausschussAufgabe")), grep, 3, 650), "contact": strip_html(find_text(root, "ausschussKontakt")), "membersShown": min(member_limit, len(members)), "newsShown": min(news_limit, len(news))}
    payload["items"] = [{"task": truncate(strip_html(find_text(root, "ausschussAufgabe")), 1200), "contact": strip_html(find_text(root, "ausschussKontakt")), "members": members[:member_limit], "news": news[:news_limit]}]
    payload["sources"] = [{"title": "Bundestag committee detail XML", "url": request_url, "kind": "api_endpoint"}]
    if find_text(root, "ausschussSourceURL"):
        payload["sources"].append({"title": "Bundestag committee page", "url": find_text(root, "ausschussSourceURL"), "kind": "public_page"})
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundestag-live committees dossier --id {committee_id} --member-limit 5"]
    emit(payload)


def run_plenum_speaker(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_xml_with_params(SPEAKER_URL, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    speakers = [{"firstName": find_text(item, "firstName"), "lastName": find_text(item, "lastName"), "name": find_text(item, "name"), "fraction": find_text(item, "fraction"), "party": find_text(item, "party"), "id": find_text(item, "id")} for item in root.findall("./speakers/speaker")]
    payload = envelope("plenum speaker", request_url, None)
    payload["summary"] = {"live": find_text(root, "live"), "topicNumber": find_text(root, "topicNumber"), "speakerCount": len(speakers)}
    payload["items"] = speakers
    payload["sources"] = source("Bundestag current speaker XML", SPEAKER_URL, "api_endpoint")
    payload["warnings"] = default_warnings() + ["The current speaker feed can be empty when no plenary sitting is live."]
    payload["nextActions"] = ["bundestag-live plenum conferences --limit 2 --item-limit 5"]
    emit(payload)


def run_plenum_conferences(argv):
    parsed = parse_args(argv)
    body, request_url = fetch_xml_with_params(CONFERENCES_URL, parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    limit = limit_flag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)
    item_limit = limit_flag_name(parsed, "item-limit", DEFAULT_LIMIT, SAFE_LIMIT)
    days = []
    next_actions = []
    for day in root.findall("./tagesordnung")[:limit]:
        agenda = []
        all_items = day.findall("./diskussionspunkte/diskussionspunkt")
        for item in all_items[:item_limit]:
            article_id = find_text(item, "articleId")
            if article_id and len(next_actions) < 3:
                next_actions.append(f"bundestag-live article get --article-id {article_id}")
            agenda.append({"startTime": find_text(item, "startzeit"), "endTime": find_text(item, "endzeit"), "status": find_text(item, "status"), "title": find_text(item, "titel"), "articleId": article_id, "top": find_text(item, "top"), "nextActions": [f"bundestag-live article get --article-id {article_id}"] if article_id else []})
        days.append({"date": find_text(day, "date"), "active": find_text(day, "active"), "sessionNumber": find_text(day, "sitzungsnummer"), "name": find_text(day, "name"), "itemCount": len(all_items), "items": agenda})
    payload = envelope("plenum conferences", request_url, {"limit": limit, "itemLimit": item_limit})
    payload["summary"] = {"totalDays": len(root.findall("./tagesordnung")), "returned": len(days)}
    payload["items"] = days
    payload["sources"] = source("Bundestag plenary conference XML", CONFERENCES_URL, "api_endpoint")
    payload["warnings"] = default_warnings() + ["Agenda article IDs point to Bundestag article XML/page records, not full plenary protocols."]
    payload["nextActions"] = next_actions or ["bundestag-live plenum speaker"]
    emit(payload)


def run_article_get(argv):
    parsed = parse_args(argv)
    article_id = first_non_empty(parsed["flags"].get("article-id"), parsed["flags"].get("id"), parsed["positionals"][0] if parsed["positionals"] else "")
    if not article_id and parsed["flags"].get("url"):
        article_id = article_id_from_url(parsed["flags"]["url"])
    if not article_id:
        raise CLIError(2, "missing_article_id", "article get requires --article-id or --url")
    body, request_url = fetch_xml_with_params(ARTICLE_URL.format(id=urllib.parse.quote(article_id)), parsed["params"])
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    grep = parsed["flags"].get("grep", "")
    payload = envelope("article get", request_url, {"articleId": article_id, "grep": grep})
    payload["summary"] = compact_article(root, grep)
    payload["items"] = [article_evidence(root, grep)]
    payload["sources"] = [{"title": "Bundestag article XML", "url": request_url, "kind": "api_endpoint"}]
    if find_text(root, "sourceURL"):
        payload["sources"].append({"title": "Bundestag public article page", "url": find_text(root, "sourceURL"), "kind": "public_page"})
        payload["nextActions"] = [f"bundestag-live article page --url \"{find_text(root, 'sourceURL')}\""]
    payload["warnings"] = default_warnings()
    emit(payload)


def run_article_page(argv):
    parsed = parse_args(argv)
    source_url = parsed["flags"].get("url", "")
    article_id = first_non_empty(parsed["flags"].get("article-id"), parsed["flags"].get("id"))
    if not source_url and article_id:
        body, _ = fetch_xml_with_params(ARTICLE_URL.format(id=urllib.parse.quote(article_id)), {})
        source_url = find_text(ET.fromstring(body), "sourceURL")
    if not source_url:
        raise CLIError(2, "missing_url", "article page requires --url or --article-id")
    if not source_url.startswith(BASE_URL):
        raise CLIError(2, "unsafe_url", "article page only accepts www.bundestag.de URLs")
    code, content_type, body = fetch_raw(source_url)
    text = strip_html(body)
    title = html_title(body)
    grep = parsed["flags"].get("grep", "")
    payload = envelope("article page", source_url, {"url": source_url, "grep": grep})
    payload["summary"] = {"url": source_url, "statusCode": code, "contentType": content_type, "title": title, "textLength": len(text), "snippetCount": len(grep_snippets(text, grep, 5, 650))}
    payload["items"] = grep_snippets(text, grep, 5, 650)
    payload["sources"] = source("Bundestag public article page", source_url, "public_page")
    payload["warnings"] = default_warnings() + ["Public HTML page extraction is best-effort; use article get for structured XML metadata when possible."]
    article_id = article_id_from_url(source_url)
    payload["nextActions"] = [f"bundestag-live article get --article-id {article_id}"] if article_id else []
    emit(payload)


def run_video_feed(argv):
    parsed = parse_args(argv)
    content_id = first_non_empty(parsed["flags"].get("content-id"), parsed["flags"].get("contentid"), parsed["params"].get("contentId"), parsed["params"].get("contentid"))
    params = dict(parsed["params"])
    if content_id:
        params.setdefault("contentId", content_id)
    body, request_url = fetch_xml_with_params(VIDEO_URL, params)
    if flag_bool(parsed, "raw"):
        print(body, end="")
        return
    root = ET.fromstring(body)
    groups = []
    for group in root.findall("./group"):
        groups.append({"type": group.attrib.get("type", ""), "streams": [{"bandwidth": stream.attrib.get("bandwidth", ""), "href": stream.attrib.get("href", "")} for stream in group.findall("./stream")]})
    payload = envelope("video feed", request_url, {"contentId": content_id})
    payload["summary"] = {"contentId": content_id, "groups": len(groups), "streamCount": sum(len(group["streams"]) for group in groups)}
    payload["items"] = groups
    payload["sources"] = [{"title": "Bundestag WebTV feed", "url": request_url, "kind": "api_endpoint"}, {"title": "Bundestag audio/video terms", "url": MEDIA_TERMS_URL, "kind": "terms"}]
    payload["warnings"] = default_warnings() + ["Video/audio material is governed by Bundestag media terms; cite Deutscher Bundestag and avoid misleading edits."]
    payload["nextActions"] = ["bundestag-live plenum conferences --limit 2 --item-limit 5"]
    emit(payload)


def run_source(argv):
    parsed = parse_args(argv)
    source_url = first_non_empty(parsed["flags"].get("url"), parsed["positionals"][0] if parsed["positionals"] else "")
    if not source_url:
        raise CLIError(2, "missing_url", "source requires --url")
    payload = envelope("source", source_url, {"url": source_url})
    payload["summary"] = {"url": source_url, "kind": source_kind(source_url), "citation": "Deutscher Bundestag, " + source_url}
    payload["sources"] = source("Bundestag source", source_url, source_kind(source_url))
    payload["warnings"] = default_warnings()
    payload["nextActions"] = [f"bundestag-live article get --article-id {article_id_from_url(source_url)}"] if article_id_from_url(source_url) else []
    emit(payload)


def member_items(root):
    out = []
    for item in root.findall("./mdbs/mdb"):
        out.append({
            "id": find_text(item, "mdbID"),
            "status": first_non_empty(find_attr(item, "mdbID", "status"), find_attr(item, "mdbName", "status")),
            "name": find_text(item, "mdbName"),
            "fraction": item.attrib.get("fraktion", ""),
            "state": find_text(item, "mdbLand"),
            "constituency": {"number": find_text(item, "mdbWahlkreis/mdbWahlkreisNummer"), "name": find_text(item, "mdbWahlkreis/mdbWahlkreisName"), "url": ""},
            "electionType": find_text(item, "mdbGewaehlt"),
            "bioUrl": find_text(item, "mdbBioURL"),
            "infoXmlUrl": find_text(item, "mdbInfoXMLURL"),
            "lastChanged": find_text(item, "lastChanged"),
            "raw": ET.tostring(item, encoding="unicode"),
        })
    return out


def compact_member(item, include_raw):
    out = {key: item.get(key) for key in ["id", "status", "name", "fraction", "state", "constituency", "electionType", "bioUrl", "infoXmlUrl", "lastChanged"]}
    out["sources"] = [{"title": "Bundestag member profile", "url": item.get("bioUrl", ""), "kind": "public_profile"}, {"title": "Bundestag member biography XML", "url": item.get("infoXmlUrl", ""), "kind": "api_endpoint"}]
    out["nextActions"] = [f"bundestag-live members dossier --id {item.get('id')}", f"bundestag-live members biography --id {item.get('id')} --raw"]
    if include_raw:
        out["raw"] = item.get("raw")
    return out


def member_search_text(item):
    return " ".join(str(item.get(key, "")) for key in ["id", "name", "fraction", "state", "electionType", "bioUrl"]) + " " + str(item.get("constituency", {}))


def resolve_member(term):
    body, _ = fetch_xml_with_params(MEMBERS_URL, {})
    for item in member_items(ET.fromstring(body)):
        if term.lower() in item["name"].lower():
            return item
    raise CLIError(2, "member_not_found", "member not found: " + term)


def compact_biography(root, grep):
    info = root.find("./mdbInfo")
    if info is None:
        info = root
    bio_text = strip_html(find_text(info, "mdbBiografischeInformationen"))
    disclosure_text = strip_html(find_text(info, "mdbVeroeffentlichungspflichtigeAngaben"))
    media = root.find("./mdbMedien")
    return {
        "id": find_text(info, "mdbID"),
        "status": find_attr(info, "mdbID", "status"),
        "name": strip_space(" ".join([find_text(info, "mdbAkademischerTitel"), find_text(info, "mdbVorname"), find_text(info, "mdbZuname")])),
        "party": find_text(info, "mdbPartei"),
        "fraction": find_text(info, "mdbFraktion"),
        "state": find_text(info, "mdbLand"),
        "profession": find_text(info, "mdbBeruf"),
        "birthDate": find_text(info, "mdbGeburtsdatum"),
        "constituency": {"number": find_text(info, "mdbWahlkreis/mdbWahlkreisNummer"), "name": find_text(info, "mdbWahlkreis/mdbWahlkreisName"), "url": find_text(info, "mdbWahlkreis/mdbWahlkreisURL")},
        "electionType": find_text(info, "mdbGewaehlt"),
        "profileUrl": first_non_empty(find_text(info, "sourceURL"), find_text(info, "mdbBioURL")),
        "homepageUrl": find_text(info, "mdbHomepageURL"),
        "speechesUrl": find_text(media, "mdbRedenVorPlenumURL") if media is not None else "",
        "speechesRss": find_text(media, "mdbRedenVorPlenumRSS") if media is not None else "",
        "biographySnippets": grep_snippets(bio_text, grep, 3, 650),
        "disclosureSnippets": grep_snippets(disclosure_text, grep, 5, 650),
    }


def member_evidence(root, grep):
    info = root.find("./mdbInfo")
    media = root.find("./mdbMedien")
    photo = media.find("./mdbFoto") if media is not None else None
    websites = []
    if info is not None:
        for website in info.findall("./mdbSonstigeWebsites/mdbSonstigeWebsite"):
            websites.append({"title": find_text(website, "mdbSonstigeWebsiteTitel"), "url": find_text(website, "mdbSonstigeWebsiteURL")})
    return {
        "biography": truncate(strip_html(find_text(info, "mdbBiografischeInformationen")), 1500),
        "disclosures": grep_snippets(strip_html(find_text(info, "mdbVeroeffentlichungspflichtigeAngaben")), grep, 8, 650),
        "websites": websites,
        "media": {"photoUrl": find_text(photo, "mdbFotoURL"), "photoSource": find_text(photo, "mdbFotoCopyright"), "speechesUrl": find_text(media, "mdbRedenVorPlenumURL"), "speechesRss": find_text(media, "mdbRedenVorPlenumRSS")},
    }


def sources_for_member(root, request_url):
    info = root.find("./mdbInfo")
    media = root.find("./mdbMedien")
    out = [{"title": "Bundestag member biography XML", "url": request_url, "kind": "api_endpoint"}]
    profile = first_non_empty(find_text(info, "sourceURL"), find_text(info, "mdbBioURL"))
    if profile:
        out.append({"title": "Bundestag member profile", "url": profile, "kind": "public_profile"})
    if media is not None and find_text(media, "mdbRedenVorPlenumURL"):
        out.append({"title": "Bundestag mediathek speeches filter", "url": find_text(media, "mdbRedenVorPlenumURL"), "kind": "media_search"})
    if media is not None and find_text(media, "mdbRedenVorPlenumRSS"):
        out.append({"title": "Bundestag speeches RSS", "url": find_text(media, "mdbRedenVorPlenumRSS"), "kind": "rss"})
    return out


def committee_items(root):
    out = []
    for item in root.findall("./ausschuesse/ausschuss"):
        out.append({"id": item.attrib.get("id", ""), "name": find_text(item, "ausschussName"), "shortName": find_text(item, "ausschussKurzName"), "teaser": strip_html(find_text(item, "ausschussTeaser")), "detailXmlUrl": find_text(item, "ausschussDetailXML"), "imageUrl": find_text(item, "imageURL"), "imageSource": find_text(item, "imageCopyright"), "lastChanged": find_text(item, "lastChanged"), "raw": ET.tostring(item, encoding="unicode")})
    return out


def compact_committee(item, include_raw):
    out = {key: item.get(key) for key in ["id", "name", "shortName", "teaser", "detailXmlUrl", "imageUrl", "imageSource", "lastChanged"]}
    out["sources"] = source("Bundestag committee XML", item.get("detailXmlUrl", ""), "api_endpoint")
    out["nextActions"] = [f"bundestag-live committees dossier --id {item.get('id')} --member-limit 5"]
    if include_raw:
        out["raw"] = item.get("raw")
    return out


def committee_search_text(item):
    return " ".join(str(item.get(key, "")) for key in ["id", "name", "shortName", "teaser", "detailXmlUrl"])


def committee_members(root):
    out = []
    for item in root.findall("./ausschussMitglieder/mdb"):
        member_id = find_text(item, "mdbID")
        out.append({"id": member_id, "name": find_text(item, "mdbName"), "fraction": item.attrib.get("fraktion", ""), "state": find_text(item, "mdbLand"), "role": find_text(item, "role"), "bioUrl": find_text(item, "mdbBioURL"), "infoXmlUrl": find_text(item, "mdbInfoXMLURL"), "lastChanged": find_text(item, "lastChanged"), "nextActions": [f"bundestag-live members dossier --id {member_id}"]})
    return out


def committee_news(root, grep):
    out = []
    for item in root.findall("./newslist/news"):
        text = strip_html(find_text(item, "teaser"))
        if grep and grep.lower() not in (text + " " + find_text(item, "title")).lower():
            continue
        article_id = item.attrib.get("articleId", "")
        out.append({"articleId": article_id, "date": find_text(item, "date"), "title": find_text(item, "title"), "teaser": truncate(text, 500), "detailsXml": find_text(item, "detailsXML"), "videoUrl": find_text(item, "video-stream/url"), "fields": [field.text or "" for field in item.findall("./politikfelder/politikfeld")], "changedDateTime": find_text(item, "changedDateTime"), "nextActions": [f"bundestag-live article get --article-id {article_id}"]})
    return out


def compact_article(root, grep):
    text = strip_html(find_text(root, "text"))
    return {"articleId": find_text(root, "articleId"), "date": find_text(root, "date"), "title": find_text(root, "title"), "sourceUrl": find_text(root, "sourceURL"), "fields": [field.text or "" for field in root.findall("./politikfelder/politikfeld")], "changedDateTime": find_text(root, "changedDateTime"), "textLength": len(text), "snippets": grep_snippets(text, grep, 5, 650)}


def article_evidence(root, grep):
    text = strip_html(find_text(root, "text"))
    return {"text": truncate(text, 1800), "snippets": grep_snippets(text, grep, 5, 650), "imageUrl": find_text(root, "imageURL"), "imageSource": find_text(root, "imageCopyright"), "imageAltText": find_text(root, "imageAltText")}


def fetch_xml_with_params(base, params):
    request_url = with_params(base, params)
    status, _, body = fetch_raw(request_url)
    if not (200 <= status < 300):
        raise RuntimeError(f"upstream status {status} from {request_url}: {truncate(strip_space(body), 300)}")
    return body, request_url


def fetch_raw(request_url):
    req = urllib.request.Request(request_url, headers={"User-Agent": "germany-skills/bundestag-live-python-2.0"})
    try:
        with urllib.request.urlopen(req, timeout=45) as response:
            return response.status, response.headers.get("Content-Type", ""), response.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.headers.get("Content-Type", ""), exc.read().decode("utf-8", "replace")


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


def with_params(base, params):
    return base if not params else base + "?" + urllib.parse.urlencode(params)


def find_text(root, path):
    if root is None:
        return ""
    item = root.find(path)
    return "" if item is None or item.text is None else item.text.strip()


def find_attr(root, path, attr):
    if root is None:
        return ""
    item = root.find(path)
    return "" if item is None else item.attrib.get(attr, "")


def envelope(command, request_url, request):
    return {"status": "ok", "tool": APP_NAME, "command": command, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "request": {"method": "GET", "url": request_url, "params": request}, "summary": {}, "items": [], "sources": [], "warnings": [], "nextActions": []}


def emit(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def fail(exit_code, code, message):
    emit({"status": "error", "tool": APP_NAME, "retrievedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "error": {"code": code, "message": message}})
    sys.exit(exit_code)


def source(title, url, kind):
    return [{"title": title, "url": url, "kind": kind}] if url else []


def default_sources():
    return [{"title": "Bundestag live XML OpenAPI wrapper", "url": OPENAPI_URL, "kind": "openapi_reference"}, {"title": "Deutscher Bundestag Open Data", "url": OPEN_DATA_URL, "kind": "official_context"}, {"title": "Bundestag website terms/imprint", "url": IMPRINT_URL, "kind": "terms"}, {"title": "Bundestag audio/video terms", "url": MEDIA_TERMS_URL, "kind": "terms"}, {"title": "Bundestag privacy policy", "url": PRIVACY_URL, "kind": "privacy"}]


def default_warnings():
    return ["No exact public rate limit for these Bundestag XML feeds was found; use small limits and avoid repeated broad index pulls.", "This live/site XML surface is not the full parliamentary archive. Use dip-bundestag for complete proceedings, printed papers, and plenary protocol research.", "Official Bundestag profile/disclosure data can include self-reported fields; preserve source URLs and timestamps in final citations.", "Website, image, and video materials may have separate usage terms; inspect the relevant source page/terms before republication."]


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


def is_help(value):
    return value in {"--help", "-h", "help"}


def first_non_empty(*values):
    for value in values:
        if value is not None and str(value).strip():
            return str(value).strip()
    return ""


def source_kind(url):
    if "/xml/" in url:
        return "api_endpoint"
    if "webtv.bundestag.de" in url:
        return "media_feed"
    if "/mediathek" in url:
        return "media_page"
    if "/abgeordnete/" in url:
        return "public_profile"
    return "public_page"


def article_id_from_url(value):
    match = re.search(r"(\d{5,})(?:\.xml)?/?(?:$|[?#])", value)
    return match.group(1) if match else ""


SCRIPT_STYLE_RE = re.compile(r"(?is)<(script|style)[^>]*>.*?</(script|style)>")
TAG_RE = re.compile(r"<[^>]+>")
SPACE_RE = re.compile(r"\s+")
TITLE_RE = re.compile(r"(?is)<title[^>]*>(.*?)</title>")


def strip_html(value):
    value = SCRIPT_STYLE_RE.sub(" ", value or "")
    value = html.unescape(value).replace("&nbsp;", " ").replace("\u00a0", " ")
    return strip_space(TAG_RE.sub(" ", value))


def strip_space(value):
    return SPACE_RE.sub(" ", value or "").strip()


def truncate(value, max_len):
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


def html_title(value):
    match = TITLE_RE.search(value or "")
    return strip_html(match.group(1)) if match else ""


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
