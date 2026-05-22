"use strict";
const APP_NAME = "dashboardctl";
const BASE_URL = "https://www.dashboard-deutschland.de";
const DASHBOARDS_URL = `${BASE_URL}/api/dashboard/get`;
const INDICATORS_URL = `${BASE_URL}/api/tile/indicators`;
const GEO_URL = `${BASE_URL}/geojson/de-all.geo.json`;
const DESTATIS_URL = "https://www.destatis.de/DE/Ueber-uns/Aufgaben/dashboards.html";
const BMWE_URL = "https://www.bundeswirtschaftsministerium.de/Redaktion/DE/Dossier/WirtschaftlicheEntwicklung/dashboard-deutschland.html";
const PYPI_URL = "https://pypi.org/project/de-dashboarddeutschland/";
const OPENAPI_REPO_URL = "https://github.com/AndreasFischer1985/dashboard-deutschland-api";
const DEFAULT_LIMIT = 10;
const SAFE_LIMIT = 100;
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
    if (argv.length === 0 || isHelp(argv[0])) {
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
        else if (matches(argv, "dashboard", "get"))
            emit(await fetchJson(withParams(DASHBOARDS_URL, parseArgs(argv.slice(2)).params)));
        else if (matches(argv, "dashboards", "list"))
            await runDashboardsList(argv.slice(2));
        else if (matches(argv, "dashboard", "dossier"))
            await runDashboardDossier(argv.slice(2));
        else if (argv[0] === "indicators")
            await runIndicatorsRaw(argv.slice(1));
        else if (matches(argv, "indicator", "search"))
            await runIndicatorSearch(argv.slice(2));
        else if (matches(argv, "indicator", "get"))
            await runIndicatorGet(argv.slice(2));
        else if (matches(argv, "indicator", "data"))
            await runIndicatorData(argv.slice(2));
        else if (matches(argv, "indicator", "source"))
            await runIndicatorSource(argv.slice(2));
        else if (argv[0] === "source")
            await runIndicatorSource(argv.slice(1));
        else if (argv[0] === "geo")
            await runGeo(argv.slice(1));
        else
            throw new CLIError(2, "unknown_command", "unknown command; run dashboardctl --help");
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
    console.log(`dashboardctl -- Dashboard Deutschland research CLI

Purpose
  Discover and normalize curated Dashboard Deutschland indicators.

Fast paths
  dashboardctl doctor
  dashboardctl dashboards list --limit 5
  dashboardctl indicator search --term "Arbeitslosigkeit" --limit 5
  dashboardctl indicator get --id tile_1666958835081
  dashboardctl indicator data --id tile_1666958835081 --limit 5
  dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3

Legacy-compatible commands
  dashboard get [--param key=value]
  indicators --param ids=tile_1666958835081
  geo
`);
}
function printHelp(path) {
    const joined = path.join(" ");
    if (joined === "indicator data") {
        console.log(`dashboardctl indicator data --id <indicator-id> [--limit n]

Extract chart-ready series from an indicator tile. Use --series to filter
series names and --from-start for earliest points.`);
    }
    else if (joined === "dashboard dossier") {
        console.log(`dashboardctl dashboard dossier --id <dashboard-id> [--indicator-limit n]

Bundle dashboard metadata and a small set of normalized indicator summaries.`);
    }
    else if (joined === "geo") {
        console.log(`dashboardctl geo

Legacy GeoJSON endpoint wrapper. The endpoint returned 403 AccessDenied in live tests.`);
    }
    else {
        printRootHelp();
    }
}
async function runDoctor(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, 2, 10);
    const payload = envelope("doctor", DASHBOARDS_URL, null);
    const warnings = defaultWarnings();
    const summary = {
        authRequired: false,
        publishedRateLimit: "No exact public rate limit was found in reviewed materials. Use small batches and avoid repeated all-indicator pulls.",
        fairUseHints: [
            "Use dashboards list or indicator search before fetching indicator data.",
            "Fetch indicator data by explicit ID.",
            "Use small --limit values for chart points.",
            "Back off on 429, 5xx, or CloudFront/S3 errors."
        ]
    };
    try {
        const dashboards = await fetchDashboards();
        const ids = uniqueIndicatorIds(dashboards);
        summary.dashboardEndpoint = { ok: true, dashboards: dashboards.length, uniqueIndicatorIds: ids.length, sampleDashboards: compactDashboards(dashboards, limit) };
        const indicators = ids.length ? await fetchIndicators(ids.slice(0, 1)) : [];
        summary.indicatorEndpoint = { ok: true, sample: compactIndicators(indicators, 1) };
    }
    catch (error) {
        summary.dashboardEndpoint = { ok: false, error: error instanceof Error ? error.message : String(error) };
        payload.status = "degraded";
    }
    const raw = await fetchRaw(GEO_URL);
    const geoOk = raw.status >= 200 && raw.status < 300;
    summary.geoEndpoint = { url: GEO_URL, statusCode: raw.status, ok: geoOk, contentType: raw.contentType, bodyPreview: truncate(stripSpace(raw.body), 180) };
    if (!geoOk) {
        payload.status = "degraded";
        warnings.push("The documented GeoJSON endpoint currently returns 403 AccessDenied; use geo as a legacy diagnostic command.");
    }
    payload.summary = summary;
    payload.sources = defaultSources();
    payload.warnings = warnings;
    payload.nextActions = ['dashboardctl indicator search --term "Arbeitslosigkeit" --limit 5', "dashboardctl dashboards list --limit 5"];
    emit(payload);
}
async function runIndicatorsRaw(argv) {
    const parsed = parseArgs(argv);
    const params = { ...parsed.params };
    if (parsed.flags.id)
        params.ids = parsed.flags.id;
    if (parsed.flags.ids)
        params.ids = parsed.flags.ids;
    emit(await fetchJson(withParams(INDICATORS_URL, params)));
}
async function runGeo(_argv) {
    const raw = await fetchRaw(GEO_URL);
    if (raw.status < 200 || raw.status >= 300) {
        throw new CLIError(1, "geo_endpoint_failed", `geo endpoint status ${raw.status} content-type ${raw.contentType} body: ${truncate(stripSpace(raw.body), 220)}`);
    }
    try {
        emit(JSON.parse(raw.body));
    }
    catch {
        console.log(raw.body);
    }
}
async function runDashboardsList(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, 50);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.positionals.join(" ")).toLowerCase();
    const dashboards = await fetchDashboards();
    const filtered = dashboards.filter((item) => !term || dashboardSearchText(item).toLowerCase().includes(term));
    const payload = envelope("dashboards list", DASHBOARDS_URL, { term, limit });
    payload.summary = { available: filtered.length, returned: Math.min(limit, filtered.length), totalDashboards: dashboards.length, uniqueIndicatorIds: uniqueIndicatorIds(dashboards).length };
    payload.items = compactDashboards(filtered, limit);
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ["dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3"];
    emit(payload);
}
async function runIndicatorSearch(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", "indicator search requires --term");
    const limit = limitFlag(parsed, 5, 50);
    const dashboards = await fetchDashboards();
    const ids = uniqueIndicatorIds(dashboards);
    const indicators = await fetchIndicators(ids);
    const needle = term.toLowerCase();
    const found = indicators.filter((item) => indicatorSearchText(item).toLowerCase().includes(needle));
    const payload = envelope("indicator search", INDICATORS_URL, { term, limit });
    payload.summary = { term, matches: found.length, searchedIndicatorIds: ids.length, returned: Math.min(limit, found.length) };
    payload.items = compactIndicators(found, limit);
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsForIndicators(found);
    emit(payload);
}
async function runIndicatorGet(argv) {
    const parsed = parseArgs(argv);
    const id = requiredId(parsed);
    const indicator = await fetchOneIndicator(id);
    const config = parseTileConfig(indicator);
    const payload = envelope("indicator get", `${INDICATORS_URL}?ids=${encodeURIComponent(id)}`, { id });
    payload.summary = indicatorSummary(indicator, config);
    payload.items = [{ summary: indicatorSummary(indicator, config), textSnippets: textSnippets(config, "", 5), widgets: widgets(config), chartSeries: seriesSummaries(config) }];
    payload.sources = sourcesForIndicator(indicator, config);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`dashboardctl indicator data --id ${id} --limit 10`, `dashboardctl indicator source --id ${id}`];
    if (flagBool(parsed, "include-raw"))
        payload.raw = { indicator, config };
    emit(payload);
}
async function runIndicatorData(argv) {
    const parsed = parseArgs(argv);
    const id = requiredId(parsed);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const seriesTerm = firstNonEmpty(parsed.flags.series, parsed.flags.grep).toLowerCase();
    const indicator = await fetchOneIndicator(id);
    const config = parseTileConfig(indicator);
    const series = extractSeries(config, limit, flagBool(parsed, "from-start"), seriesTerm);
    const payload = envelope("indicator data", `${INDICATORS_URL}?ids=${encodeURIComponent(id)}`, { id, limit, series: seriesTerm });
    payload.summary = { id, title: firstNonEmpty(config.title, indicator.title), seriesReturned: series.length, pointsPerSeries: limit, dataVersionDate: config.dataVersionDate, lastUpdated: millisSummary(config.lastUpdated) };
    payload.items = series;
    payload.sources = sourcesForIndicator(indicator, config);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`dashboardctl indicator source --id ${id}`];
    if (flagBool(parsed, "include-raw"))
        payload.raw = config;
    emit(payload);
}
async function runIndicatorSource(argv) {
    const parsed = parseArgs(argv);
    const id = requiredId(parsed);
    const indicator = await fetchOneIndicator(id);
    const config = parseTileConfig(indicator);
    const payload = envelope("indicator source", `${INDICATORS_URL}?ids=${encodeURIComponent(id)}`, { id });
    payload.summary = { id, title: firstNonEmpty(config.title, indicator.title), sourceCount: sourcesForIndicator(indicator, config).length };
    payload.sources = sourcesForIndicator(indicator, config);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`dashboardctl indicator data --id ${id} --limit 10`];
    emit(payload);
}
async function runDashboardDossier(argv) {
    const parsed = parseArgs(argv);
    const indicatorLimit = limitFlagName(parsed, "indicator-limit", 3, 10);
    const dashboards = await fetchDashboards();
    const dashboard = findDashboard(dashboards, parsed);
    const ids = dashboardIndicatorIds(dashboard).slice(0, indicatorLimit);
    const indicators = ids.length ? await fetchIndicators(ids) : [];
    const payload = envelope("dashboard dossier", DASHBOARDS_URL, { id: dashboard.id, indicatorLimit });
    payload.summary = compactDashboard(dashboard);
    payload.items = compactIndicators(indicators, indicators.length);
    payload.sources = sourcesForDashboard();
    payload.warnings = defaultWarnings();
    payload.nextActions = ids.slice(0, 3).map((id) => `dashboardctl indicator data --id ${id} --limit 10`);
    emit(payload);
}
async function fetchDashboards() {
    const data = await fetchJson(DASHBOARDS_URL);
    return Array.isArray(data) ? data : [];
}
async function fetchOneIndicator(id) {
    const items = await fetchIndicators([id]);
    if (!items.length)
        throw new CLIError(2, "indicator_not_found", `indicator not found: ${id}`);
    return items[0];
}
async function fetchIndicators(ids) {
    if (!ids.length)
        throw new CLIError(2, "missing_ids", "indicator IDs required");
    const all = [];
    for (let start = 0; start < ids.length; start += 20) {
        const chunk = ids.slice(start, start + 20);
        const data = await fetchJson(withParams(INDICATORS_URL, { ids: chunk.join(";") }));
        if (Array.isArray(data))
            all.push(...data);
    }
    return all;
}
async function fetchJson(requestUrl) {
    const raw = await fetchRaw(requestUrl);
    if (raw.status < 200 || raw.status >= 300)
        throw new Error(`upstream status ${raw.status} from ${requestUrl}: ${truncate(stripSpace(raw.body), 300)}`);
    return JSON.parse(raw.body);
}
async function fetchRaw(requestUrl) {
    const response = await fetch(requestUrl, { headers: { "User-Agent": "germany-skills/dashboardctl-node-2.0" }, signal: AbortSignal.timeout(45000) });
    return { status: response.status, contentType: response.headers.get("content-type") ?? "", body: await response.text() };
}
function parseTileConfig(indicator) {
    if (!indicator.json)
        throw new CLIError(2, "missing_embedded_json", "indicator has no embedded json field");
    return JSON.parse(String(indicator.json));
}
function compactDashboards(dashboards, limit) {
    return dashboards.slice(0, limit).map(compactDashboard);
}
function compactDashboard(dashboard) {
    const ids = dashboardIndicatorIds(dashboard);
    return { id: dashboard.id, name: dashboard.name, nameEn: dashboard.nameEn, description: truncate(stripHtml(dashboard.description ?? ""), 420), category: compactCategory(dashboard.category ?? {}), tags: dashboard.tags ?? [], indicatorCount: ids.length, indicatorIds: ids.slice(0, 12), nextActions: [`dashboardctl dashboard dossier --id ${dashboard.id} --indicator-limit 3`] };
}
function compactIndicators(indicators, limit) {
    return indicators.slice(0, limit).map((indicator) => {
        let config = {};
        try {
            config = parseTileConfig(indicator);
        }
        catch { }
        const summary = indicatorSummary(indicator, config);
        summary.nextActions = [`dashboardctl indicator data --id ${indicator.id} --limit 10`, `dashboardctl indicator source --id ${indicator.id}`];
        return summary;
    });
}
function indicatorSummary(indicator, config) {
    return { id: indicator.id, title: firstNonEmpty(config.title, indicator.title), apiTitle: indicator.title, category: config.category, tags: config.tags ?? [], sourceCount: sourceEntries(config).length, sources: sourceEntries(config), componentCount: (config.components ?? []).length, seriesCount: seriesSummaries(config).length, widgetCount: widgets(config).length, dataVersionDate: config.dataVersionDate, dateUpload: config.dateUpload, lastUpdated: millisSummary(config.lastUpdated) };
}
function extractSeries(config, limit, fromStart, seriesTerm) {
    const out = [];
    for (const component of config.components ?? []) {
        for (const series of component.chart?.series ?? []) {
            const name = firstNonEmpty(series.custom?.name, series.name);
            const id = series.id ?? "";
            if (seriesTerm && !`${name} ${id}`.toLowerCase().includes(seriesTerm))
                continue;
            const points = series.data ?? [];
            out.push({ id, name, color: series.color, pointCount: points.length, points: fromStart ? points.slice(0, limit) : points.slice(Math.max(0, points.length - limit)), firstPoint: points[0] ?? null, lastPoint: points[points.length - 1] ?? null });
        }
    }
    return out;
}
function seriesSummaries(config) {
    const out = [];
    for (const component of config.components ?? []) {
        for (const series of component.chart?.series ?? []) {
            const points = series.data ?? [];
            out.push({ id: series.id ?? "", name: firstNonEmpty(series.custom?.name, series.name), pointCount: points.length, firstPoint: points[0] ?? null, lastPoint: points[points.length - 1] ?? null });
        }
    }
    return out;
}
function widgets(config) {
    const out = [];
    for (const component of config.components ?? []) {
        for (const widget of component.widgets ?? [])
            out.push({ num: widget.num, desc: stripHtml(widget.desc ?? ""), icon: widget.icon });
    }
    return out;
}
function textSnippets(config, grep, limit) {
    const needle = grep.toLowerCase();
    const out = [];
    for (const component of config.components ?? []) {
        const text = stripHtml(firstNonEmpty(component.text, component.infoButtonText, component.description));
        if (text.length > 20 && (!needle || text.toLowerCase().includes(needle)))
            out.push({ text: truncate(text, 700), type: component.type });
        if (out.length >= limit)
            break;
    }
    return out;
}
function sourceEntries(config) {
    const out = (config.sources ?? []).map((source) => ({ title: firstNonEmpty(source.name, "Dashboard Deutschland source"), url: source.link ?? "", kind: "indicator_source", quality: source.quality }));
    if (!out.length && config.source)
        out.push({ title: "Dashboard source field", url: "", kind: "source_text", text: stripHtml(config.source) });
    return out;
}
function sourcesForIndicator(indicator, config) {
    return [{ title: "Dashboard Deutschland indicator API", url: `${INDICATORS_URL}?ids=${encodeURIComponent(indicator.id ?? "")}`, kind: "api_endpoint" }, { title: "Dashboard Deutschland", url: BASE_URL, kind: "official_dashboard" }, ...sourceEntries(config)];
}
function sourcesForDashboard() {
    return [{ title: "Dashboard Deutschland dashboard API", url: DASHBOARDS_URL, kind: "api_endpoint" }, { title: "Dashboard Deutschland", url: BASE_URL, kind: "official_dashboard" }, { title: "Destatis dashboard page", url: DESTATIS_URL, kind: "official_context" }];
}
function defaultSources() {
    return [{ title: "Dashboard Deutschland", url: BASE_URL, kind: "official_dashboard" }, { title: "Dashboard Deutschland dashboard API", url: DASHBOARDS_URL, kind: "api_endpoint" }, { title: "Dashboard Deutschland indicator API", url: INDICATORS_URL, kind: "api_endpoint" }, { title: "Dashboard Deutschland GeoJSON endpoint", url: GEO_URL, kind: "api_endpoint" }, { title: "Destatis dashboards page", url: DESTATIS_URL, kind: "official_context" }, { title: "BMWE Dashboard Deutschland page", url: BMWE_URL, kind: "official_context" }, { title: "PyPI generated DashboardDeutschland package", url: PYPI_URL, kind: "openapi_reference" }, { title: "Dashboard Deutschland OpenAPI wrapper", url: OPENAPI_REPO_URL, kind: "openapi_reference" }];
}
function defaultWarnings() {
    return ["No exact published API rate limit was found in reviewed materials; use small batches and avoid repeated all-indicator pulls.", "Indicator tiles contain an embedded JSON string; parse it before interpreting chart data, sources, widgets, or update dates.", "The documented GeoJSON endpoint returned 403 AccessDenied in live tests.", "Dashboard Deutschland is curated and mixed-source; for deep statistical table work use Destatis/GENESIS where appropriate."];
}
function uniqueIndicatorIds(dashboards) {
    return [...new Set(dashboards.flatMap(dashboardIndicatorIds))].sort();
}
function dashboardIndicatorIds(dashboard) {
    return (dashboard.layoutTiles ?? []).map((tile) => firstNonEmpty(tile.indicatorid, tile.indicatorId)).filter(Boolean);
}
function findDashboard(dashboards, parsed) {
    const wanted = firstNonEmpty(parsed.flags.id, parsed.flags.name, parsed.positionals.join(" ")).toLowerCase();
    if (!wanted)
        throw new CLIError(2, "missing_dashboard", "dashboard dossier requires --id or --name");
    const found = dashboards.find((dashboard) => String(dashboard.id ?? "").toLowerCase() === wanted || String(dashboard.name ?? "").toLowerCase().includes(wanted));
    if (!found)
        throw new CLIError(2, "dashboard_not_found", `dashboard not found: ${wanted}`);
    return found;
}
function dashboardSearchText(dashboard) {
    return [dashboard.id, dashboard.name, dashboard.nameEn, dashboard.description, dashboard.category?.name, ...(dashboard.tags ?? []), ...dashboardIndicatorIds(dashboard)].join(" ");
}
function indicatorSearchText(indicator) {
    let config = {};
    try {
        config = parseTileConfig(indicator);
    }
    catch { }
    return [indicator.id, indicator.title, config.title, config.category, config.source, config.dataVersionDate, config.dateUpload, ...(config.tags ?? []), ...sourceEntries(config).flatMap((source) => [source.title, source.url]), ...textSnippets(config, "", 8).map((snippet) => snippet.text)].join(" ");
}
function nextActionsForIndicators(items) {
    const actions = items.slice(0, 3).flatMap((item) => [`dashboardctl indicator get --id ${item.id}`, `dashboardctl indicator data --id ${item.id} --limit 10`]);
    return actions.length ? actions : ['dashboardctl indicator search --term "Arbeitsmarkt" --limit 5'];
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
function requiredId(parsed) {
    const id = firstNonEmpty(parsed.flags.id, parsed.flags.ids, parsed.positionals[0]);
    if (!id)
        throw new CLIError(2, "missing_id", "command requires --id");
    return id;
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
function withParams(base, params) {
    const query = new URLSearchParams(params).toString();
    return query ? `${base}?${query}` : base;
}
function compactCategory(category) {
    return { id: category.id, name: category.name, nameEn: category.nameEn, description: truncate(stripHtml(category.description ?? ""), 300) };
}
function millisSummary(value) {
    const ms = Number.parseInt(String(value ?? ""), 10);
    if (!Number.isFinite(ms) || ms <= 0)
        return {};
    return { epochMs: ms, iso: new Date(ms).toISOString() };
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
function firstNonEmpty(...values) {
    for (const value of values)
        if (value !== undefined && value !== null && String(value).trim())
            return String(value).trim();
    return "";
}
function isHelp(value) {
    return value === "--help" || value === "-h" || value === "help";
}
function matches(argv, ...expected) {
    return expected.every((value, index) => argv[index] === value);
}
function stripHtml(value) {
    return stripSpace(String(value).replace(/&nbsp;/g, " ").replace(/\u00a0/g, " ").replace(/<[^>]+>/g, " "));
}
function stripSpace(value) {
    return String(value).replace(/\s+/g, " ").trim();
}
function truncate(value, maxLen) {
    return value.length <= maxLen ? value : `${value.slice(0, maxLen)}...`;
}
main(process.argv.slice(2)).then((code) => {
    process.exitCode = code;
});
