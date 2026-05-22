#!/usr/bin/env node
"use strict";
const APP_NAME = "bundesratctl";
const BASE_URL = "https://www.bundesrat.de";
const OPENAPI_URL = "https://github.com/bundesAPI/bundesrat-api";
const SERVICE_BUND_URL = "https://www.service.bund.de/Content/DE/DEBehoerden/B/BR/Bundesrat.html";
const IMPRINT_URL = `${BASE_URL}/DE/service-navi/impressum/impressum-node.html`;
const PRIVACY_URL = `${BASE_URL}/DE/service-navi/datenschutz/datenschutz-node.html`;
const ROBOTS_URL = `${BASE_URL}/robots.txt`;
const DEFAULT_LIMIT = 10;
const SAFE_LIMIT = 100;
const ENDPOINTS = {
    startlist: `${BASE_URL}/iOS/v3/startlist_table.xml`,
    news: `${BASE_URL}/iOS/v3/01_Aktuelles/aktuelles_table.xml`,
    dates: `${BASE_URL}/iOS/v3/02_Termine/termine_table.xml`,
    "plenum compact": `${BASE_URL}/iOS/v3/03_Plenum/plenum_kompakt_table.xml`,
    "plenum current": `${BASE_URL}/iOS/SharedDocs/3_Plenum/plenum_aktuelleSitzung_table.xml`,
    "plenum chronological": `${BASE_URL}/iOS/SharedDocs/3_Plenum/plenum_toChronologisch_table.xml`,
    "plenum next": `${BASE_URL}/iOS/SharedDocs/3_Plenum/plenum_naechsteSitzungen.xml`,
    members: `${BASE_URL}/iOS/SharedDocs/2_Mitglieder/mitglieder_table.xml`,
    votes: `${BASE_URL}/iOS/v3/06_Stimmen/stimmverteilung.xml`,
    presidium: `${BASE_URL}/iOS/v3/05_Bundesrat/Praesidium/bundesrat_praesidium.xml`
};
class CLIError extends Error {
    exitCode;
    code;
    constructor(exitCode, code, message) {
        super(message);
        this.exitCode = exitCode;
        this.code = code;
    }
}
async function main(argv) {
    if (!argv.length || isHelp(argv[0])) {
        printRootHelp();
        return 0;
    }
    if (isHelp(argv[argv.length - 1])) {
        printHelp(argv.slice(0, -1));
        return 0;
    }
    try {
        if (argv[0] === "doctor")
            await runDoctor(argv.slice(1));
        else if (argv[0] === "examples")
            printExamples();
        else if (argv[0] === "startlist")
            await runFeed("startlist", argv.slice(1));
        else if (argv[0] === "news" && argv[1] === "search")
            await runFeedSearch("news", argv.slice(2));
        else if (argv[0] === "news" && argv[1] === "page")
            await runPage("news page", argv.slice(2));
        else if (argv[0] === "news")
            await runFeed("news", argv.slice(1));
        else if (argv[0] === "dates" && argv[1] === "search")
            await runFeedSearch("dates", argv.slice(2));
        else if (argv[0] === "dates" && argv[1] === "page")
            await runPage("dates page", argv.slice(2));
        else if (argv[0] === "dates")
            await runFeed("dates", argv.slice(1));
        else if (matches(argv, "plenum", "compact"))
            await runPlenum("plenum compact", argv.slice(2));
        else if (matches(argv, "plenum", "current"))
            await runPlenum("plenum current", argv.slice(2));
        else if (matches(argv, "plenum", "chronological"))
            await runPlenum("plenum chronological", argv.slice(2));
        else if (matches(argv, "plenum", "next"))
            await runPlenumNext(argv.slice(2));
        else if (matches(argv, "plenum", "dossier"))
            await runPlenum("plenum compact", argv.slice(2));
        else if (argv[0] === "members" && argv[1] === "search")
            await runMembersSearch(argv.slice(2));
        else if (argv[0] === "members" && argv[1] === "dossier")
            await runMemberDossier(argv.slice(2));
        else if (argv[0] === "members")
            await runMembers(argv.slice(1));
        else if (argv[0] === "votes" && argv[1] === "summary")
            await runFeed("votes", argv.slice(2));
        else if (argv[0] === "votes")
            await runFeed("votes", argv.slice(1));
        else if (argv[0] === "presidium")
            await runFeed("presidium", argv.slice(1));
        else if (argv[0] === "page")
            await runPage("page", argv.slice(1));
        else if (argv[0] === "source")
            runSource(argv.slice(1));
        else
            throw new CLIError(2, "unknown_command", "unknown command; run bundesratctl --help");
    }
    catch (error) {
        if (error instanceof CLIError) {
            fail(error.exitCode, error.code, error.message);
            return error.exitCode;
        }
        fail(1, "unexpected_error", error instanceof Error ? error.message : String(error));
        return 1;
    }
    return 0;
}
function printRootHelp() {
    console.log(`bundesratctl -- Bundesrat live/app XML research CLI

Purpose
  Discover and normalize public Bundesrat app XML feeds for news, dates,
  plenary-session summaries and agenda items, members, vote distribution,
  presidium/context pages, and source URLs.

Fast paths
  bundesratctl doctor
  bundesratctl news --limit 5
  bundesratctl news search --term "Bovenschulte" --limit 3
  bundesratctl members search --name "Özdemir" --limit 3
  bundesratctl members dossier --name "Özdemir" --grep "Bundesrat"
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
`);
}
function printHelp(path) {
    const joined = path.join(" ");
    if (joined === "news search" || joined === "dates search")
        console.log('bundesratctl news search --term "Bovenschulte" --limit 3');
    else if (joined === "members search")
        console.log('bundesratctl members search --name "Özdemir" --limit 3');
    else if (joined === "members dossier")
        console.log('bundesratctl members dossier --name "Özdemir" --grep "Bundesrat"');
    else if (["page", "news page", "dates page"].includes(joined))
        console.log('bundesratctl page --url "https://www.bundesrat.de/..." --grep "term"');
    else
        printRootHelp();
}
function printExamples() {
    console.log(`bundesratctl examples

1. bundesratctl doctor
2. bundesratctl startlist --limit 12
3. bundesratctl news --limit 5
4. bundesratctl news search --term "Bovenschulte" --limit 3
5. bundesratctl dates --limit 5
6. bundesratctl members search --name "Özdemir" --limit 3
7. bundesratctl members dossier --name "Özdemir" --grep "Bundesrat"
8. bundesratctl plenum compact --limit 1 --top-limit 3
9. bundesratctl plenum current --limit 1 --top-limit 5
10. bundesratctl plenum compact --raw
`);
}
async function runDoctor(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, 5, 10);
    const checks = ["startlist", "news", "dates", "plenum compact", "members", "votes"].slice(0, limit);
    const summary = {
        authRequired: false,
        publishedRateLimit: "No exact public request quota was found in the OpenAPI wrapper or Bundesrat website material. The site robots.txt currently publishes Crawl-delay: 30; use small limits, cache repeated feed calls, and back off on 429/5xx responses.",
        fairUseHints: ["Prefer search and dossier commands before broad source-page expansion.", "Respect robots.txt Crawl-delay: 30 for crawling-like workflows.", "Use --limit and --top-limit on broad feeds.", "Preserve source URLs, retrieval timestamps, and image/media copyright fields."],
        endpoints: []
    };
    let status = "ok";
    for (const name of checks) {
        const requestUrl = withDefaultView(ENDPOINTS[name], {});
        const raw = await fetchRaw(requestUrl);
        const ok = raw.status >= 200 && raw.status < 300;
        if (!ok)
            status = "degraded";
        summary.endpoints.push({ name, url: ENDPOINTS[name], statusCode: raw.status, contentType: raw.contentType, ok, bodyPreview: truncate(stripSpace(raw.body), 180) });
    }
    const payload = envelope("doctor", BASE_URL, { limit });
    payload.status = status;
    payload.summary = summary;
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ["bundesratctl news --limit 5", 'bundesratctl members search --name "Özdemir" --limit 3', "bundesratctl plenum compact --limit 1 --top-limit 3"];
    emit(payload);
}
async function runFeed(key, argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchEndpoint(key, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const grep = firstNonEmpty(parsed.flags.grep, parsed.flags.term, parsed.flags.q);
    const items = compactItems(body, key, limit, grep, flagBool(parsed, "include-raw"));
    const payload = envelope(key, requestUrl, { limit, grep });
    payload.summary = { totalItems: countItemLike(body), returned: items.length, grep };
    payload.items = items;
    payload.sources = source(`Bundesrat ${key} XML feed`, requestUrl, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsFromItems(items, key);
    emit(payload);
}
async function runFeedSearch(key, argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.flags.name, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", `${key} search requires --term`);
    parsed.flags.grep = term;
    await runFeed(key, rebuildArgs(parsed));
}
async function runMembers(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchEndpoint("members", parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const items = compactEmployees(body, limit, "", flagBool(parsed, "include-raw"));
    const payload = envelope("members", requestUrl, { limit });
    payload.summary = { totalMembers: blocks(body, "employee").length, returned: items.length };
    payload.items = items;
    payload.sources = source("Bundesrat member XML feed", requestUrl, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = ['bundesratctl members search --name "Özdemir" --limit 3'];
    emit(payload);
}
async function runMembersSearch(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.name, parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", "members search requires --name or --term");
    const { body, requestUrl } = await fetchEndpoint("members", {});
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const items = compactEmployees(body, limit, term, flagBool(parsed, "include-raw"));
    const payload = envelope("members search", requestUrl, { term, limit });
    payload.summary = { term, totalMembers: blocks(body, "employee").length, matchesReturned: items.length };
    payload.items = items;
    payload.sources = source("Bundesrat member XML feed", requestUrl, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsFromEmployees(items);
    emit(payload);
}
async function runMemberDossier(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.name, parsed.flags.url, parsed.flags.term, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_member", "members dossier requires --name or --url");
    const { body, requestUrl } = await fetchEndpoint("members", {});
    const grep = parsed.flags.grep ?? "";
    const matchesFound = compactEmployees(body, SAFE_LIMIT, term, flagBool(parsed, "include-raw"));
    if (!matchesFound.length)
        throw new CLIError(2, "member_not_found", `member not found in current Bundesrat feed: ${term}`);
    const item = matchesFound[0];
    const text = String(item.evidenceText ?? "");
    const payload = envelope("members dossier", requestUrl, { term, grep });
    payload.summary = { name: item.name, party: item.party, state: item.state, profileUrl: item.url, snippetCount: grepSnippets(text, grep, 8, 650).length };
    payload.items = [{ profile: item, snippets: grepSnippets(text, grep, 8, 650) }];
    payload.sources = [{ title: "Bundesrat member XML feed", url: requestUrl, kind: "api_endpoint" }];
    if (item.url) {
        payload.sources.push({ title: "Bundesrat member profile", url: item.url, kind: "public_profile" });
        payload.nextActions = [`bundesratctl page --url "${item.url}" --grep "${firstNonEmpty(grep, "Bundesrat")}"`];
    }
    payload.warnings = defaultWarnings();
    emit(payload);
}
async function runPlenum(key, argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchEndpoint(key, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const topLimit = limitFlagName(parsed, "top-limit", DEFAULT_LIMIT, SAFE_LIMIT);
    const grep = firstNonEmpty(parsed.flags.grep, parsed.flags.term, parsed.flags.q);
    const items = compactPlenum(body, key, limit, topLimit, grep, flagBool(parsed, "include-raw"));
    const payload = envelope(key, requestUrl, { limit, topLimit, grep });
    payload.summary = plenumSummary(body, key, items.length, grep);
    payload.items = items;
    payload.sources = source(`Bundesrat ${key} XML feed`, requestUrl, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsFromItems(items, key);
    emit(payload);
}
async function runPlenumNext(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchEndpoint("plenum next", parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const itemBlocks = blocks(body, "item");
    const items = compactItems(body, "plenum next", limit, parsed.flags.grep ?? "", flagBool(parsed, "include-raw"));
    const sessions = itemBlocks.length ? tableRows(tag(itemBlocks[0], "detail")) : [];
    const payload = envelope("plenum next", requestUrl, { limit });
    payload.summary = { returned: items.length, upcomingSessions: sessions };
    payload.items = items;
    payload.sources = source("Bundesrat next plenary sessions XML feed", requestUrl, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = ["bundesratctl plenum current --limit 1 --top-limit 5", "bundesratctl plenum compact --limit 1 --top-limit 5"];
    emit(payload);
}
async function runPage(command, argv) {
    const parsed = parseArgs(argv);
    const sourceUrl = firstNonEmpty(parsed.flags.url, parsed.flags["source-url"], parsed.positionals.join(" "));
    if (!sourceUrl)
        throw new CLIError(2, "missing_url", `${command} requires --url`);
    if (!sourceUrl.startsWith(`${BASE_URL}/`))
        throw new CLIError(2, "unsafe_url", "page only accepts https://www.bundesrat.de URLs");
    const raw = await fetchRaw(sourceUrl);
    const grep = parsed.flags.grep ?? "";
    const text = stripHtml(raw.body);
    const payload = envelope(command, sourceUrl, { url: sourceUrl, grep });
    payload.summary = { url: sourceUrl, statusCode: raw.status, contentType: raw.contentType, title: htmlTitle(raw.body), textLength: text.length, snippetCount: grepSnippets(text, grep, 8, 650).length };
    payload.items = grepSnippets(text, grep, 8, 650);
    payload.sources = source("Bundesrat public source page", sourceUrl, "public_page");
    payload.warnings = [...defaultWarnings(), "Public HTML extraction is best-effort; prefer XML feed fields for structured metadata."];
    payload.nextActions = [`bundesratctl source --url "${sourceUrl}"`];
    emit(payload);
}
function runSource(argv) {
    const parsed = parseArgs(argv);
    const sourceUrl = firstNonEmpty(parsed.flags.url, parsed.positionals.join(" "));
    if (!sourceUrl)
        throw new CLIError(2, "missing_url", "source requires --url");
    const payload = envelope("source", sourceUrl, { url: sourceUrl });
    payload.summary = { url: sourceUrl, kind: sourceKind(sourceUrl), citation: `Bundesrat, ${sourceUrl}` };
    payload.sources = source("Bundesrat source", sourceUrl, sourceKind(sourceUrl));
    payload.warnings = defaultWarnings();
    if (sourceUrl.startsWith(`${BASE_URL}/`))
        payload.nextActions = [`bundesratctl page --url "${sourceUrl}"`];
    emit(payload);
}
async function fetchEndpoint(key, params) {
    if (!ENDPOINTS[key])
        throw new CLIError(2, "unknown_endpoint", `unknown endpoint: ${key}`);
    const requestUrl = withDefaultView(ENDPOINTS[key], params ?? {});
    const raw = await fetchRaw(requestUrl);
    if (raw.status < 200 || raw.status >= 300)
        throw new Error(`upstream status ${raw.status} from ${requestUrl}: ${truncate(stripSpace(raw.body), 300)}`);
    return { body: raw.body, requestUrl };
}
async function fetchRaw(requestUrl) {
    const response = await fetch(requestUrl, { headers: { "User-Agent": "democracy-researcher/bundesratctl-node-2.0" }, signal: AbortSignal.timeout(45000) });
    return { status: response.status, contentType: response.headers.get("content-type") ?? "", body: await response.text() };
}
function withDefaultView(base, params) {
    const query = new URLSearchParams({ ...(params ?? {}) });
    if (!query.has("view"))
        query.set("view", "renderXml");
    return `${base}?${query.toString()}`;
}
function compactItems(raw, key, limit, grep, includeRaw) {
    const out = [];
    for (const item of blocks(raw, "item")) {
        if (grep && !itemSearchText(item).toLowerCase().includes(grep.toLowerCase()))
            continue;
        out.push(compactItem(item, key, grep, includeRaw));
        if (out.length >= limit)
            break;
    }
    return out;
}
function compactItem(item, key, grep, includeRaw) {
    const detail = tag(item, "detail");
    const text = firstNonEmpty(tag(item, "bodyText"), tag(item, "description"), tag(item, "abstract"), stripHtml(detail));
    const sourceUrl = tag(item, "url");
    const out = {
        type: tag(item, "type"),
        id: tag(item, "id"),
        name: tag(item, "name"),
        title: firstNonEmpty(tag(item, "title"), tag(item, "name")),
        url: sourceUrl,
        date: tag(item, "date"),
        dateOfIssue: tag(item, "dateOfIssue"),
        startDate: tag(item, "startdate"),
        stopDate: tag(item, "stopdate"),
        summary: truncate(stripHtml(text), 700),
        imageUrl: tag(item, "imagePath"),
        imageDate: tag(item, "imageDate"),
        imageCaption: tag(item, "imageCaption"),
        sources: source("Bundesrat source", sourceUrl, sourceKind(sourceUrl)),
        links: extractLinks(detail, 10),
        snippets: grepSnippets(stripHtml(`${detail} ${text}`), grep, 4, 650),
        nextActions: nextActionsForUrl(sourceUrl, key)
    };
    if (includeRaw)
        out.raw = item;
    return out;
}
function compactEmployees(raw, limit, term, includeRaw) {
    const out = [];
    for (const employee of blocks(raw, "employee")) {
        if (term && !employeeSearchText(employee).toLowerCase().includes(term.toLowerCase()))
            continue;
        const firstName = tag(employee, "firstname");
        const lastName = tag(employee, "name");
        const sourceUrl = tag(employee, "url");
        const evidence = stripHtml([tag(employee, "detail1"), tag(employee, "detail2"), tag(employee, "detail3")].join(" "));
        const item = {
            name: stripSpace(`${firstName} ${lastName}`),
            firstName,
            lastName,
            party: tag(employee, "party"),
            state: tag(employee, "state"),
            isBundesratMember: tag(employee, "brmitglied"),
            isMember: tag(employee, "mitglied"),
            isBevollmaechtigt: tag(employee, "bv"),
            url: sourceUrl,
            imageUrl: tag(employee, "imagePath"),
            roles: truncate(stripHtml(tag(employee, "detail1")), 1000),
            biography: truncate(stripHtml(tag(employee, "detail2")), 1000),
            contact: truncate(stripHtml(tag(employee, "detail3")), 1000),
            evidenceText: evidence,
            sources: source("Bundesrat member profile", sourceUrl, "public_profile"),
            nextActions: nextActionsForUrl(sourceUrl, "members")
        };
        if (includeRaw)
            item.raw = employee;
        out.push(item);
        if (out.length >= limit)
            break;
    }
    return out;
}
function compactPlenum(raw, key, limit, topLimit, grep, includeRaw) {
    const out = [];
    const header = block(raw, "header");
    if (header && header.includes("<url>")) {
        out.push({ kind: "header", url: tag(header, "url"), title: firstNonEmpty(tag(header, "titel2"), tag(header, "title")), subtitle: stripSpace([tag(header, "titel1"), tag(header, "titel3"), tag(header, "titelAlt")].join(" ")), detailType: tag(header, "detailTyp"), summary: truncate(stripHtml(firstNonEmpty(tag(header, "vorschautext"), tag(header, "detail"))), 1000), sources: source("Bundesrat plenary page", tag(header, "url"), "public_page"), links: extractLinks(tag(header, "detail"), 10), snippets: grepSnippets(stripHtml(tag(header, "detail")), grep, 4, 650), nextActions: nextActionsForUrl(tag(header, "url"), key) });
    }
    for (const top of blocks(raw, "top")) {
        if (grep && !stripHtml(top).toLowerCase().includes(grep.toLowerCase()))
            continue;
        const detail = firstNonEmpty(tag(top, "detail"), tag(top, "topdetail"));
        const sourceUrl = tag(top, "url");
        const item = { kind: "top", top: firstNonEmpty(tag(top, "nr"), tag(top, "toptitle")), printMatter: tag(top, "topdrucksache"), filter: tag(top, "filter"), title: firstNonEmpty(tag(top, "title"), tag(top, "topheader")), url: sourceUrl, summary: truncate(stripHtml(firstNonEmpty(tag(top, "topheader"), detail)), 900), links: extractLinks(detail, 14), snippets: grepSnippets(stripHtml(detail), grep, 5, 650), sources: source("Bundesrat plenary TOP", sourceUrl, sourceKind(sourceUrl)), nextActions: nextActionsForUrl(sourceUrl, key) };
        if (includeRaw)
            item.raw = top;
        out.push(item);
        if (out.length >= limit + topLimit)
            break;
    }
    return (out.length ? out : compactItems(raw, key, limit, grep, includeRaw)).slice(0, limit + topLimit);
}
function plenumSummary(raw, key, returned, grep) {
    return { title: tag(raw, "title"), header: truncate(stripHtml(block(raw, "header")), 900), topCount: blocks(raw, "top").length, itemCount: blocks(raw, "item").length, returned, grep, sourceUrl: firstNonEmpty(tag(block(raw, "header"), "url"), ENDPOINTS[key]) };
}
function itemSearchText(item) {
    return stripHtml([tag(item, "type"), tag(item, "id"), tag(item, "name"), tag(item, "title"), tag(item, "url"), tag(item, "date"), tag(item, "dateOfIssue"), tag(item, "bodyText"), tag(item, "description"), tag(item, "abstract"), tag(item, "detail")].join(" "));
}
function employeeSearchText(employee) {
    return stripHtml([tag(employee, "firstname"), tag(employee, "name"), tag(employee, "party"), tag(employee, "state"), tag(employee, "url"), tag(employee, "detail1"), tag(employee, "detail2"), tag(employee, "detail3")].join(" "));
}
function nextActionsFromItems(items, key) {
    const actions = [];
    for (const item of items) {
        actions.push(...nextActionsForUrl(String(item.url ?? ""), key));
        if (actions.length >= 4)
            return actions;
    }
    if (key === "news")
        return ['bundesratctl news search --term "Bovenschulte" --limit 3'];
    if (key === "dates")
        return ['bundesratctl dates search --term "Ausschuss" --limit 5'];
    return ["bundesratctl plenum compact --limit 1 --top-limit 3"];
}
function nextActionsFromEmployees(items) {
    const actions = items.slice(0, 3).filter((item) => item.name).map((item) => `bundesratctl members dossier --name "${item.name}"`);
    return actions.length ? actions : ['bundesratctl members search --name "Özdemir" --limit 3'];
}
function nextActionsForUrl(sourceUrl, key) {
    if (!sourceUrl)
        return [];
    const actions = [];
    if (sourceUrl.startsWith(`${BASE_URL}/`))
        actions.push(`bundesratctl page --url "${sourceUrl}"`);
    if (key === "news")
        actions.push(`bundesratctl news page --url "${sourceUrl}"`);
    if (key === "dates")
        actions.push(`bundesratctl dates page --url "${sourceUrl}"`);
    return actions;
}
function defaultSources() {
    return [{ title: "bundesAPI Bundesrat OpenAPI wrapper", url: OPENAPI_URL, kind: "openapi_reference" }, { title: "service.bund.de Bundesrat profile", url: SERVICE_BUND_URL, kind: "official_context" }, { title: "Bundesrat robots.txt", url: ROBOTS_URL, kind: "fair_use" }, { title: "Bundesrat Impressum", url: IMPRINT_URL, kind: "terms" }, { title: "Bundesrat Datenschutzerklärung", url: PRIVACY_URL, kind: "privacy" }];
}
function defaultWarnings() {
    return ["No exact public rate limit for these Bundesrat XML feeds was found; robots.txt publishes Crawl-delay: 30, so avoid crawling-style rapid page expansion.", "This app/live XML surface is current-publication oriented, not a complete historical archive.", "Bundesrat public pages can include image/media copyright notices; preserve source URLs and copyright fields in final artifacts.", "Votes by individual Land are generally not always recorded by the Bundesrat itself; inspect plenary records and linked state pages where the distinction matters."];
}
function parseArgs(args) {
    const parsed = { flags: {}, params: {}, positionals: [] };
    for (let i = 0; i < args.length; i += 1) {
        const arg = args[i];
        if (!arg.startsWith("--")) {
            parsed.positionals.push(arg);
            continue;
        }
        let key = arg.slice(2);
        let value = "true";
        if (key.includes("=")) {
            const splitAt = key.indexOf("=");
            value = key.slice(splitAt + 1);
            key = key.slice(0, splitAt);
        }
        else if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
            value = args[i + 1];
            i += 1;
        }
        key = key.toLowerCase().trim();
        if (key === "param" && value.includes("=")) {
            const splitAt = value.indexOf("=");
            parsed.params[value.slice(0, splitAt)] = value.slice(splitAt + 1);
        }
        else {
            parsed.flags[key] = value;
        }
    }
    return parsed;
}
function rebuildArgs(parsed) {
    const args = [];
    for (const [key, value] of Object.entries(parsed.flags))
        args.push(`--${key}`, value);
    for (const [key, value] of Object.entries(parsed.params))
        args.push("--param", `${key}=${value}`);
    return [...args, ...parsed.positionals];
}
function limitFlag(parsed, fallback, maxValue) {
    return limitFlagName(parsed, "limit", fallback, maxValue);
}
function limitFlagName(parsed, name, fallback, maxValue) {
    const parsedValue = Number.parseInt(String(parsed.flags[name] ?? fallback), 10);
    const value = Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : fallback;
    if (value > maxValue && !flagBool(parsed, "allow-large-output"))
        throw new CLIError(2, "limit_exceeds_safe_max", `${name} ${value} exceeds safe max ${maxValue}; pass --allow-large-output to override`);
    return value;
}
function flagBool(parsed, key) {
    return ["true", "1", "yes", "y"].includes(String(parsed.flags[key] ?? "").toLowerCase());
}
function envelope(command, requestUrl, request) {
    return { status: "ok", tool: APP_NAME, command, retrievedAt: new Date().toISOString(), request: { method: "GET", url: requestUrl, params: request }, summary: {}, items: [], sources: [], warnings: [], nextActions: [] };
}
function emit(value) {
    console.log(JSON.stringify(value, null, 2));
}
function fail(exitCode, code, message) {
    emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message } });
    process.exitCode = exitCode;
}
function printRaw(value) {
    process.stdout.write(value);
}
function isHelp(value) {
    return value === "--help" || value === "-h" || value === "help";
}
function matches(argv, ...expected) {
    return expected.every((value, index) => argv[index] === value);
}
function firstNonEmpty(...values) {
    for (const value of values)
        if (value !== undefined && value !== null && String(value).trim())
            return String(value).trim();
    return "";
}
function countItemLike(raw) {
    const count = blocks(raw, "item").length;
    return count || blocks(raw, "employee").length + blocks(raw, "top").length;
}
function sourceKind(sourceUrl) {
    if (!sourceUrl)
        return "unknown";
    if (sourceUrl.includes("dip.bundestag.de"))
        return "dip_reference";
    if (sourceUrl.includes("/SharedDocs/personen/"))
        return "public_profile";
    if (sourceUrl.includes("/SharedDocs/drucksachen/") || sourceUrl.includes("/drs.html"))
        return "official_document";
    if (sourceUrl.includes("/DE/plenum/"))
        return "plenary_page";
    if (sourceUrl.includes("/SharedDocs/pm/"))
        return "press_release";
    return "public_page";
}
function source(title, url, kind) {
    return url ? [{ title, url, kind }] : [];
}
function block(xml, name) {
    const match = new RegExp(`<${escapeRegExp(name)}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${escapeRegExp(name)}>`, "i").exec(xml || "");
    return match ? match[1] : "";
}
function blocks(xml, name) {
    return [...String(xml || "").matchAll(new RegExp(`<${escapeRegExp(name)}(?:\\s[^>]*)?>[\\s\\S]*?<\\/${escapeRegExp(name)}>`, "gi"))].map((match) => match[0]);
}
function tag(xml, name) {
    return decodeEntities(block(xml, name).replace(/^<!\[CDATA\[/, "").replace(/\]\]>$/, "")).trim();
}
function stripHtml(value) {
    return stripSpace(decodeEntities(String(value ?? "").replace(/^<!\[CDATA\[/, "").replace(/\]\]>$/, "").replace(/<(script|style)[^>]*>.*?<\/(script|style)>/gis, " ").replace(/<br\s*\/?>/gi, " ").replace(/<[^>]+>/g, " ")));
}
function stripSpace(value) {
    return String(value ?? "").replace(/\s+/g, " ").trim();
}
function truncate(value, maxLen) {
    value = stripSpace(value);
    return value.length <= maxLen ? value : `${value.slice(0, maxLen)}...`;
}
function grepSnippets(text, grep, limit, maxLen) {
    text = stripSpace(text);
    if (!text)
        return [];
    if (!grep)
        return [{ text: truncate(text, maxLen) }];
    const lower = text.toLowerCase();
    const needle = grep.toLowerCase().trim();
    const out = [];
    const seen = new Set();
    let startFrom = 0;
    while (out.length < limit) {
        const idx = lower.indexOf(needle, startFrom);
        if (idx < 0)
            break;
        const start = Math.max(0, idx - Math.floor(maxLen / 2));
        const end = Math.min(text.length, start + maxLen);
        const snippet = text.slice(start, end).trim();
        const key = snippet.slice(0, 180);
        if (!seen.has(key)) {
            out.push({ grep, text: snippet });
            seen.add(key);
        }
        startFrom = idx + needle.length;
    }
    return out;
}
function extractLinks(value, limit) {
    const out = [];
    const seen = new Set();
    for (const match of String(value || "").matchAll(/<a\s+[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi)) {
        let rawUrl = decodeEntities(match[1]).trim();
        if (!rawUrl || rawUrl.startsWith("mailto:") || rawUrl.startsWith("tel:"))
            continue;
        if (rawUrl.startsWith("/"))
            rawUrl = BASE_URL + rawUrl;
        if (!rawUrl.startsWith("http"))
            rawUrl = `${BASE_URL}/${rawUrl.replace(/^\.\//, "")}`;
        if (seen.has(rawUrl))
            continue;
        seen.add(rawUrl);
        out.push({ title: truncate(stripHtml(match[2]), 160), url: rawUrl, kind: sourceKind(rawUrl) });
        if (out.length >= limit)
            break;
    }
    return out;
}
function tableRows(value) {
    return [...String(value || "").matchAll(/<tr>\s*<td[^>]*>(.*?)<\/td>\s*<td[^>]*>(.*?)<\/td>\s*<\/tr>/gis)].map((match) => ({ date: stripHtml(match[1]), time: stripHtml(match[2]) }));
}
function htmlTitle(value) {
    return stripHtml(/<title[^>]*>([\s\S]*?)<\/title>/i.exec(value ?? "")?.[1] ?? "");
}
function decodeEntities(value) {
    return String(value ?? "")
        .replace(/&nbsp;/g, " ")
        .replace(/&amp;/g, "&")
        .replace(/&lt;/g, "<")
        .replace(/&gt;/g, ">")
        .replace(/&quot;/g, '"')
        .replace(/&#(\d+);/g, (_, n) => String.fromCodePoint(Number.parseInt(n, 10)))
        .replace(/&#x([0-9a-f]+);/gi, (_, n) => String.fromCodePoint(Number.parseInt(n, 16)));
}
function escapeRegExp(value) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
main(process.argv.slice(2)).then((code) => {
    process.exitCode = code;
});
