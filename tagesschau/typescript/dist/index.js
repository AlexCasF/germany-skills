#!/usr/bin/env node
"use strict";
const APP_NAME = "tagesschauctl";
const BASE_URL = "https://www.tagesschau.de";
const HOMEPAGE_URL = `${BASE_URL}/api2u/homepage`;
const NEWS_URL = `${BASE_URL}/api2u/news`;
const CHANNELS_URL = `${BASE_URL}/api2u/channels`;
const SEARCH_URL = `${BASE_URL}/api2u/search`;
const API_DOCS_URL = "https://github.com/bundesAPI/tagesschau-api";
const OPENAPI_URL = "https://github.com/bundesAPI/tagesschau-api/raw/refs/heads/main/openapi.yaml";
const CC_URL = "https://www.tagesschau.de/multimedia/video/creative-commons-index-100.html";
const RSS_INFO_URL = "https://www.tagesschau.de/infoservices/rssfeeds";
const USER_AGENT = "democracy-researcher/tagesschauctl-node-2.0";
const DEFAULT_LIMIT = 10;
const MAX_LIMIT = 30;
class CliError extends Error {
    code;
    exitCode;
    constructor(code, message, exitCode = 1) {
        super(message);
        this.code = code;
        this.exitCode = exitCode;
    }
}
main(process.argv.slice(2)).then((exitCode) => {
    process.exitCode = exitCode;
}, (error) => {
    emitError("unexpected_error", error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
});
async function main(argv) {
    try {
        if (argv.length === 0 || isHelp(argv[0])) {
            printRootHelp();
            return 0;
        }
        if (isHelp(argv[argv.length - 1])) {
            printHelp(argv.slice(0, -1));
            return 0;
        }
        if (argv[0] === "doctor")
            await runDoctor(argv.slice(1));
        else if (argv[0] === "examples")
            printExamples();
        else if (argv[0] === "source")
            runSource(argv.slice(1));
        else if (argv[0] === "fields")
            runFields(argv.slice(1));
        else if (argv[0] === "homepage")
            await runFeed("homepage", HOMEPAGE_URL, argv.slice(1));
        else if (argv[0] === "news")
            await runFeed("news", NEWS_URL, argv.slice(1));
        else if (argv[0] === "channels")
            await runFeed("channels", CHANNELS_URL, argv.slice(1));
        else if (argv[0] === "search")
            await runSearch(argv.slice(1));
        else if (match(argv, "article", "get"))
            await runArticle("article get", argv.slice(2), false);
        else if (match(argv, "article", "source"))
            runArticleSource(argv.slice(2));
        else if (match(argv, "article", "dossier"))
            await runArticle("article dossier", argv.slice(2), true);
        else
            throw new CliError("unknown_command", `unknown command: ${argv.join(" ")}`, 2);
        return 0;
    }
    catch (error) {
        if (error instanceof CliError) {
            emitError(error.code, error.message);
            return error.exitCode;
        }
        emitError("unexpected_error", error instanceof Error ? error.message : String(error));
        return 1;
    }
}
function printRootHelp() {
    console.log(`tagesschauctl 2.0 - Tagesschau public JSON feed research CLI

Usage:
  tagesschauctl doctor
  tagesschauctl homepage --limit 5
  tagesschauctl news --ressort inland --limit 5
  tagesschauctl channels --limit 5
  tagesschauctl search --text "Bundestag" --limit 5
  tagesschauctl article get --url "https://www.tagesschau.de/...-100.html" --grep "Bundestag"
  tagesschauctl article dossier --url "https://www.tagesschau.de/...-100.html"

Tagesschau is a current-news context source, not the sole official evidence for parliamentary, legal, fiscal, or statistical claims.`);
}
function printHelp(args) {
    if (args.length === 0)
        printRootHelp();
    else if (args[0] === "search")
        console.log("search flags: --text/--searchText TERM --limit 1-30 --result-page N --include-raw --raw --param key=value");
    else if (args[0] === "news")
        console.log("news flags: --ressort inland|ausland|wirtschaft|sport|video|investigativ|wissen --regions 1,2 --limit 1-30 --include-raw --raw --param key=value");
    else if (args[0] === "homepage")
        console.log("homepage flags: --limit 1-30 --include-regional --include-raw --raw");
    else if (match(args, "article", "get"))
        console.log("article get flags: --url URL --grep TERM --limit 1-30 --include-raw --raw");
    else if (match(args, "article", "source"))
        console.log("article source flags: --url URL");
    else if (match(args, "article", "dossier"))
        console.log("article dossier flags: --url URL --grep TERM --limit 1-30 --include-raw");
    else
        printRootHelp();
}
function printExamples() {
    console.log(`Examples:
  tagesschauctl doctor
  tagesschauctl homepage --limit 5
  tagesschauctl news --ressort inland --limit 5
  tagesschauctl search --text "Bundestag" --limit 5
  tagesschauctl search --param searchText=Bundestag --param pageSize=5
  tagesschauctl article get --url "https://www.tagesschau.de/inland/example-100.html" --grep "Bundestag"`);
}
async function runDoctor(_argv) {
    const checks = [];
    for (const [name, url] of [
        ["homepage", HOMEPAGE_URL],
        ["news", withParams(NEWS_URL, { ressort: "inland" })],
        ["channels", CHANNELS_URL],
        ["search", withParams(SEARCH_URL, { searchText: "Bundestag", pageSize: "1" })],
    ]) {
        const item = { name, url };
        try {
            const response = await fetchRaw(url);
            Object.assign(item, { ok: response.status >= 200 && response.status < 300, statusCode: response.status, contentType: response.contentType, bodyBytes: response.body.length });
        }
        catch (error) {
            Object.assign(item, { ok: false, error: error instanceof Error ? error.message : String(error) });
        }
        checks.push(item);
    }
    const payload = envelope("doctor", "GET", "multiple", {});
    payload.summary = {
        authRequired: false,
        documentedLimit: "The published API documentation states that more than 60 requests per hour are not allowed.",
        usageRestrictions: "Private, non-commercial use is allowed; publication is not allowed except for content explicitly released under a Creative Commons license.",
        recommendedRole: "Use as a current-news context layer, not as the sole official source for institutional or statistical claims.",
        endpointHealth: checks,
        copyrightSensitive: true,
    };
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ['tagesschauctl search --text "Bundestag" --limit 5', "tagesschauctl homepage --limit 5", "tagesschauctl source"];
    emit(payload);
}
function runSource(_argv) {
    const payload = envelope("source", "GET", API_DOCS_URL, {});
    payload.summary = {
        publisher: "Tagesschau / ARD-aktuell; API documentation mirrored by bundesAPI.",
        authRequired: false,
        documentedLimit: "No more than 60 requests per hour.",
        reuseRestriction: "Private, non-commercial use only; no publication except explicitly CC-licensed offers.",
        primaryEndpoints: [HOMEPAGE_URL, NEWS_URL, CHANNELS_URL, SEARCH_URL],
        articleURLPattern: "Public detailsweb URLs can be converted to /api2u/...json detail URLs.",
    };
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ["tagesschauctl fields", 'tagesschauctl search --text "Bundestag" --limit 5'];
    emit(payload);
}
function runFields(_argv) {
    const payload = envelope("fields", "GET", API_DOCS_URL, {});
    payload.summary = {
        feeds: [
            { command: "homepage", meaning: "Selected current and breaking items shown in the app homepage." },
            { command: "news", meaning: "Current news feed; filterable by ressort and region." },
            { command: "channels", meaning: "Current livestream/program channels." },
            { command: "search", meaning: "Search feed with searchText, resultPage, and pageSize." },
        ],
        ressorts: ["inland", "ausland", "wirtschaft", "sport", "video", "investigativ", "wissen"],
        regions: { "1": "Baden-Württemberg", "2": "Bayern", "3": "Berlin", "4": "Brandenburg", "5": "Bremen", "6": "Hamburg", "7": "Hessen", "8": "Mecklenburg-Vorpommern", "9": "Niedersachsen", "10": "Nordrhein-Westfalen", "11": "Rheinland-Pfalz", "12": "Saarland", "13": "Sachsen", "14": "Sachsen-Anhalt", "15": "Schleswig-Holstein", "16": "Thüringen" },
        coreArticleFields: ["title", "topline", "date", "details", "detailsweb", "shareURL", "firstSentence", "ressort", "type", "tags"],
    };
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ['tagesschauctl search --text "Bundestag" --limit 5', "tagesschauctl homepage --limit 5"];
    emit(payload);
}
async function runFeed(command, endpoint, argv) {
    const parsed = parseArgs(argv);
    const params = { ...parsed.params };
    for (const key of ["ressort", "regions"]) {
        if (parsed.flags[key])
            params[key] = parsed.flags[key];
    }
    const requestUrl = withParams(endpoint, params);
    const response = await fetchRaw(requestUrl);
    if (response.status < 200 || response.status >= 300)
        throw new CliError("upstream_http_error", `upstream status ${response.status} from ${requestUrl}: ${stripSpace(response.body).slice(0, 260)}`);
    if (flagBool(parsed, "raw")) {
        process.stdout.write(response.body);
        return;
    }
    const data = JSON.parse(response.body);
    const limit = limitFlag(parsed);
    const items = compactFeedItems(data, limit, flagBool(parsed, "include-regional"));
    const payload = envelope(command, "GET", requestUrl, params);
    payload.summary = { type: data.type, itemsReturned: items.length, rawCounts: feedCounts(data) };
    payload.items = items;
    payload.sources = [{ kind: "api_request", title: "Tagesschau API request", url: requestUrl }, ...defaultSources()];
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsFromItems(items);
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    emit(payload);
}
async function runSearch(argv) {
    const parsed = parseArgs(argv);
    const params = { ...parsed.params };
    const text = first(parsed.flags.text, parsed.flags.searchText, parsed.flags.q, parsed.positionals.join(" "));
    if (text)
        params.searchText = text;
    const limit = limitFlag(parsed);
    params.pageSize ??= String(limit);
    if (parsed.flags["page-size"] || parsed.flags.pageSize)
        params.pageSize = first(parsed.flags["page-size"], parsed.flags.pageSize);
    if (parsed.flags["result-page"] || parsed.flags.resultPage || parsed.flags.page)
        params.resultPage = first(parsed.flags["result-page"], parsed.flags.resultPage, parsed.flags.page);
    const requestUrl = withParams(SEARCH_URL, params);
    const response = await fetchRaw(requestUrl);
    if (response.status < 200 || response.status >= 300)
        throw new CliError("upstream_http_error", `upstream status ${response.status} from ${requestUrl}: ${stripSpace(response.body).slice(0, 260)}`);
    if (flagBool(parsed, "raw")) {
        process.stdout.write(response.body);
        return;
    }
    const data = JSON.parse(response.body);
    const items = compactArray(data.searchResults, limit);
    const payload = envelope("search", "GET", requestUrl, params);
    payload.summary = { searchText: data.searchText, totalItemCount: data.totalItemCount, pageSize: data.pageSize, resultPage: data.resultPage, itemsReturned: items.length, copyrightNotice: "Do not republish Tagesschau article text unless content is explicitly CC-licensed." };
    payload.items = items;
    payload.sources = [{ kind: "api_request", title: "Tagesschau API request", url: requestUrl }, ...defaultSources()];
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsFromItems(items);
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    emit(payload);
}
async function runArticle(command, argv, dossier) {
    const parsed = parseArgs(argv);
    const inputUrl = first(parsed.flags.url, parsed.params.url, parsed.positionals.join(" "));
    if (!inputUrl)
        throw new CliError("missing_url", `${command} requires --url`, 2);
    const urls = articleUrls(inputUrl);
    const response = await fetchRaw(urls.apiUrl);
    if (response.status < 200 || response.status >= 300)
        throw new CliError("upstream_http_error", `upstream status ${response.status} from ${urls.apiUrl}: ${stripSpace(response.body).slice(0, 260)}`);
    if (flagBool(parsed, "raw")) {
        process.stdout.write(response.body);
        return;
    }
    const data = JSON.parse(response.body);
    const grep = first(parsed.flags.grep, parsed.flags.term, parsed.flags.q);
    const limit = limitFlag(parsed);
    const snippets = articleSnippets(data, grep, limit);
    const summary = compactArticle(data);
    summary.snippetCount = snippets.length;
    summary.snippets = snippets;
    if (dossier)
        summary.dossierUse = "Use as current-news context; verify institutional/statistical claims against primary official sources.";
    const payload = envelope(command, "GET", urls.apiUrl, { url: inputUrl, grep, limit });
    payload.summary = summary;
    payload.items = snippets;
    payload.sources = [{ kind: "api_request", title: "Tagesschau article JSON", url: urls.apiUrl }, { kind: "public_article", title: "Tagesschau public article", url: urls.publicUrl }, ...defaultSources()];
    payload.warnings = defaultWarnings();
    payload.nextActions = [`tagesschauctl article source --url "${urls.publicUrl}"`];
    if (dossier)
        payload.nextActions.push("tagesschauctl source");
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    emit(payload);
}
function runArticleSource(argv) {
    const parsed = parseArgs(argv);
    const inputUrl = first(parsed.flags.url, parsed.params.url, parsed.positionals.join(" "));
    if (!inputUrl)
        throw new CliError("missing_url", "article source requires --url", 2);
    const urls = articleUrls(inputUrl);
    const payload = envelope("article source", "GET", urls.apiUrl, { url: inputUrl });
    payload.summary = {
        apiUrl: urls.apiUrl,
        publicUrl: urls.publicUrl,
        sourceType: "news_context",
        reuseRestriction: "Do not republish article text except where explicitly CC-licensed.",
        recommendedUse: "Cite headline/date/public URL; use short snippets only as needed for analysis.",
        primaryEvidenceUse: false,
    };
    payload.sources = [{ kind: "api_request", title: "Tagesschau article JSON", url: urls.apiUrl }, { kind: "public_article", title: "Tagesschau public article", url: urls.publicUrl }, ...defaultSources()];
    payload.warnings = defaultWarnings();
    payload.nextActions = [`tagesschauctl article get --url "${urls.publicUrl}" --limit 5`];
    emit(payload);
}
function compactFeedItems(data, limit, includeRegional) {
    const items = compactArray(data.news, limit);
    if (includeRegional && items.length < limit)
        items.push(...compactArray(data.regional, limit - items.length));
    if (items.length === 0)
        items.push(...compactArray(data.channels, limit));
    return items.slice(0, limit);
}
function compactArray(value, limit) {
    return Array.isArray(value) ? value.filter(isObject).slice(0, limit).map(compactArticle) : [];
}
function compactArticle(obj) {
    const details = stringValue(obj.details);
    let publicUrl = first(stringValue(obj.detailsweb), stringValue(obj.detailsWeb), stringValue(obj.shareURL));
    if (!publicUrl && details)
        publicUrl = articleUrls(details).publicUrl;
    const item = {
        title: stringValue(obj.title),
        topline: stringValue(obj.topline),
        date: stringValue(obj.date),
        type: stringValue(obj.type),
        firstSentence: stripHtml(stringValue(obj.firstSentence)),
        sophoraId: stringValue(obj.sophoraId),
        externalId: stringValue(obj.externalId),
        details,
        detailsweb: publicUrl,
        shareURL: stringValue(obj.shareURL),
        ressort: stringValue(obj.ressort),
        tags: tagStrings(obj.tags),
    };
    if (publicUrl) {
        item.sourceUrl = publicUrl;
        item.nextActions = [`tagesschauctl article get --url "${publicUrl}" --limit 5`, `tagesschauctl article source --url "${publicUrl}"`];
    }
    return item;
}
function articleSnippets(data, grep, limit) {
    if (!Array.isArray(data.content))
        return [];
    const needle = grep.toLowerCase();
    const snippets = [];
    data.content.forEach((raw, index) => {
        if (snippets.length >= limit || !isObject(raw))
            return;
        const type = stringValue(raw.type);
        if (type !== "text" && type !== "headline")
            return;
        const text = stripHtml(stringValue(raw.value));
        if (!text || (needle && !text.toLowerCase().includes(needle)))
            return;
        snippets.push({ index, type, text: truncate(text, 520), matched: !needle || text.toLowerCase().includes(needle) });
    });
    return snippets;
}
function articleUrls(inputUrl) {
    let parsed;
    try {
        parsed = new URL(inputUrl.trim());
    }
    catch {
        throw new CliError("invalid_url", "expected an absolute Tagesschau URL", 2);
    }
    if (!parsed.hostname.endsWith("tagesschau.de"))
        throw new CliError("invalid_url", "expected a tagesschau.de URL", 2);
    if (parsed.pathname.startsWith("/api2u/")) {
        const publicPath = parsed.pathname.replace(/^\/api2u/, "").replace(/\.json$/, ".html");
        return { apiUrl: `https://www.tagesschau.de${parsed.pathname}`, publicUrl: `https://www.tagesschau.de${publicPath}` };
    }
    const apiPath = `/api2u${parsed.pathname.replace(/\.html$/, ".json")}`;
    return { apiUrl: `https://www.tagesschau.de${apiPath}`, publicUrl: `https://www.tagesschau.de${parsed.pathname}` };
}
async function fetchRaw(requestUrl) {
    let lastStatus = 0;
    let lastBody = "";
    let lastContentType = "";
    let lastError;
    for (let attempt = 0; attempt < 2; attempt += 1) {
        if (attempt > 0)
            await sleep(750);
        try {
            const response = await fetch(requestUrl, { headers: { "User-Agent": USER_AGENT, Accept: "application/json" } });
            const body = await response.text();
            if (![429, 502, 503, 504].includes(response.status))
                return { status: response.status, body, contentType: response.headers.get("content-type") ?? "" };
            lastStatus = response.status;
            lastBody = body;
            lastContentType = response.headers.get("content-type") ?? "";
        }
        catch (error) {
            lastError = error;
        }
    }
    if (lastStatus)
        return { status: lastStatus, body: lastBody, contentType: lastContentType };
    throw lastError instanceof Error ? lastError : new Error(String(lastError));
}
function parseArgs(argv) {
    const flags = {};
    const params = {};
    const positionals = [];
    for (let index = 0; index < argv.length; index += 1) {
        const token = argv[index];
        if (token.startsWith("--")) {
            const key = token.slice(2);
            if (key === "param") {
                const value = argv[++index] ?? "";
                const equals = value.indexOf("=");
                if (equals < 1)
                    throw new CliError("invalid_param", "--param requires key=value", 2);
                params[value.slice(0, equals)] = value.slice(equals + 1);
            }
            else if (argv[index + 1] && !argv[index + 1].startsWith("--")) {
                flags[key] = argv[++index];
            }
            else {
                flags[key] = "true";
            }
        }
        else {
            positionals.push(token);
        }
    }
    return { flags, params, positionals };
}
function envelope(command, method, requestUrl, params) {
    return { status: "ok", tool: APP_NAME, command, retrievedAt: new Date().toISOString(), request: { method, url: requestUrl, params }, summary: {}, items: [], sources: [], warnings: [], nextActions: [] };
}
function defaultSources() {
    return [
        { kind: "api_docs", title: "bundesAPI Tagesschau API documentation", url: API_DOCS_URL },
        { kind: "openapi", title: "Tagesschau OpenAPI YAML", url: OPENAPI_URL },
        { kind: "public_service", title: "tagesschau.de", url: `${BASE_URL}/` },
        { kind: "usage", title: "Tagesschau RSS and reuse notice", url: RSS_INFO_URL },
        { kind: "license", title: "Creative Commons videos", url: CC_URL },
    ];
}
function defaultWarnings() {
    return [
        "Published API documentation says not to make more than 60 requests per hour.",
        "Tagesschau content use is private/non-commercial; publication is not allowed except for content explicitly under Creative Commons.",
        "Use this as current-news context, not as the only evidence for official parliamentary, legal, fiscal, or statistical claims.",
        "Avoid reproducing long article text; cite the public article URL and use short snippets only when needed.",
    ];
}
function feedCounts(data) {
    const counts = {};
    for (const key of ["news", "regional", "channels", "searchResults"]) {
        if (Array.isArray(data[key]))
            counts[key] = data[key].length;
    }
    return counts;
}
function nextActionsFromItems(items) {
    const actions = items.slice(0, 3).filter((item) => item.sourceUrl).map((item) => `tagesschauctl article get --url "${item.sourceUrl}" --limit 5`);
    return actions.length ? actions : ["tagesschauctl source"];
}
function withParams(base, params) {
    const url = new URL(base);
    for (const [key, value] of Object.entries(params)) {
        if (value.trim())
            url.searchParams.set(key, value);
    }
    return url.toString();
}
function tagStrings(value) {
    if (!Array.isArray(value))
        return [];
    return value.filter(isObject).map((item) => stringValue(item.tag)).filter(Boolean).sort();
}
const tagPattern = /<[^>]+>/g;
const spacePattern = /\s+/g;
function stripHtml(value) {
    return decodeEntities(value.replaceAll("<br />", " ").replaceAll("<br/>", " ").replace(tagPattern, " ")).replace(spacePattern, " ").trim();
}
function decodeEntities(value) {
    return value.replace(/&nbsp;/g, " ").replace(/&amp;/g, "&").replace(/&quot;/g, "\"").replace(/&#39;/g, "'").replace(/&lt;/g, "<").replace(/&gt;/g, ">");
}
function stripSpace(value) {
    return value.replace(spacePattern, " ").trim();
}
function truncate(value, max) {
    return Array.from(value).length <= max ? value : `${Array.from(value).slice(0, max).join("")}...`;
}
function first(...values) {
    for (const value of values) {
        if (value && value.trim())
            return value.trim();
    }
    return "";
}
function flagBool(parsed, key) {
    return ["1", "true", "yes", "on"].includes((parsed.flags[key] ?? "").toLowerCase());
}
function limitFlag(parsed) {
    if (!parsed.flags.limit)
        return DEFAULT_LIMIT;
    const value = Number(parsed.flags.limit);
    if (!Number.isInteger(value))
        throw new CliError("invalid_limit", "--limit must be an integer", 2);
    return Math.max(0, Math.min(value, MAX_LIMIT));
}
function stringValue(value) {
    return typeof value === "string" ? value : "";
}
function isObject(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}
function isHelp(value) {
    return ["--help", "-h", "help"].includes(value);
}
function match(args, ...expected) {
    return args.length >= expected.length && expected.every((value, index) => args[index] === value);
}
function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
function emit(payload) {
    console.log(JSON.stringify(payload, null, 2));
}
function emitError(code, message) {
    emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message } });
}
