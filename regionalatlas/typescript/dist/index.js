"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const node_fs_1 = require("node:fs");
const APP_NAME = "regionalatlasctl";
const MAP_SERVER_URL = "https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer";
const QUERY_ENDPOINT = `${MAP_SERVER_URL}/dynamicLayer/query`;
const CATALOG_URL = "https://regionalatlas.statistikportal.de/taskrunner/services.json";
const THESAURUS_URL = "https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv";
const APP_URL = "https://regionalatlas.statistikportal.de/";
const STATISTIKPORTAL_URL = "https://www.statistikportal.de/de/karten/regionalatlas-deutschland";
const DESTATIS_URL = "https://www.destatis.de/DE/Service/Statistik-Visualisiert/RegionalatlasAktuell.html";
const OPEN_DATA_URL = "https://www.statistikportal.de/de/open-data";
const MAPS_GEODATA_URL = "https://www.destatis.de/DE/Service/OpenData/karten-geodaten.html";
const OPENAPI_REPO_URL = "https://github.com/bundesAPI/regionalatlas-api";
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
        else if (matches(argv, "indicators", "list"))
            await runIndicatorsList(argv.slice(2));
        else if (matches(argv, "indicators", "search"))
            await runIndicatorsSearch(argv.slice(2));
        else if (matches(argv, "indicator", "get"))
            await runIndicatorGet(argv.slice(2));
        else if (argv[0] === "fields")
            await runFields(argv.slice(1));
        else if (argv[0] === "sample")
            await runSample(argv.slice(1));
        else if (argv[0] === "source")
            await runSource(argv.slice(1));
        else if (argv[0] === "dossier")
            await runDossier(argv.slice(1));
        else if (argv[0] === "query-builder")
            await runQueryBuilder(argv.slice(1));
        else if (argv[0] === "explain-field")
            await runExplainField(argv.slice(1));
        else if (argv[0] === "query")
            await runRawQuery(argv.slice(1));
        else
            throw new CLIError(2, "unknown_command", "unknown command; run regionalatlasctl --help");
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
    console.log(`regionalatlasctl -- Regionalatlas Deutschland research CLI

Purpose
  Discover and query official Regionalatlas indicators from the statistical
  offices of the German federation and states.

Fast paths
  regionalatlasctl doctor
  regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5
  regionalatlasctl fields --indicator AI008-1-5
  regionalatlasctl sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 5
  regionalatlasctl dossier --indicator AI008-1-5 --field AI0801 --year 2024

Legacy-compatible command
  query --layer <dynamic-layer-json> [--param key=value]

Research commands
  doctor
  indicators list
  indicators search
  indicator get
  fields
  sample
  source
  dossier
  query-builder
  explain-field
  query
`);
}
function printHelp(scope) {
    const topic = scope.join(" ");
    if (topic === "sample") {
        console.log(`regionalatlasctl sample --indicator <code> [--field <field>] [--year <yyyy>] [--region-level 1|2|3|5]

Fetches a small ArcGIS dynamic-layer sample. Defaults:
  --region-level 1
  --limit 10
  --geometry false

Useful flags
  --indicator AI008-1-5
  --field AI0801
  --year 2024
  --region-level 1
  --ags 11
  --fields ags,gen,ai0801
  --include-raw
`);
        return;
    }
    if (topic === "query") {
        console.log(`regionalatlasctl query --layer <dynamic-layer-json> [--param key=value]

Raw compatibility wrapper around the ArcGIS dynamicLayer/query endpoint.
Use query-builder first if you need a safe generated layer payload.

Examples
  regionalatlasctl query --layer-file layer.json --param outFields=ags,gen,ai0801
  regionalatlasctl query --layer <json> --param resultRecordCount=5

On Windows shells, prefer --layer-file because raw JSON quoting is fragile.
`);
        return;
    }
    printRootHelp();
}
async function runDoctor(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, 1, 10);
    const payload = envelope("doctor", `${MAP_SERVER_URL}?f=json`, null);
    const warnings = defaultWarnings();
    const summary = {
        authRequired: false,
        catalogUrl: CATALOG_URL,
        mapServerUrl: MAP_SERVER_URL,
        publishedRateLimit: "No exact public API rate limit found in reviewed Regionalatlas/API materials. Use small limits, cache catalog metadata, and avoid parallel broad ArcGIS queries.",
        fairUseHints: [
            "Use indicators search/list and fields before sample/query.",
            "Do not request geometry unless map shapes are required.",
            "Avoid municipality-level full pulls unless explicitly exporting with a plan.",
            "Back off on 429, 5xx, or slow responses."
        ]
    };
    try {
        summary.mapServerReachable = true;
        summary.mapServer = mapServerSummary(await fetchJson(`${MAP_SERVER_URL}?f=json`));
    }
    catch (error) {
        summary.mapServerReachable = false;
        warnings.push(`mapServer: ${error instanceof Error ? error.message : String(error)}`);
    }
    try {
        const catalog = await fetchCatalog();
        const flat = flattenCatalog(catalog);
        summary.catalogReachable = true;
        summary.topics = Array.isArray(catalog) ? catalog.length : 0;
        summary.indicators = flat.length;
        summary.sampleIndicators = compactIndicators(flat, limit);
    }
    catch (error) {
        summary.catalogReachable = false;
        warnings.push(`catalog: ${error instanceof Error ? error.message : String(error)}`);
    }
    payload.status = summary.mapServerReachable && summary.catalogReachable ? "ok" : "degraded";
    payload.summary = summary;
    payload.sources = defaultSources();
    payload.warnings = warnings;
    payload.nextActions = [
        'regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5',
        "regionalatlasctl fields --indicator AI008-1-5"
    ];
    emit(payload);
}
async function runIndicatorsList(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, 50);
    const topic = firstNonEmpty(parsed.flags.topic, parsed.flags.thema).toLowerCase();
    let flat = flattenCatalog(await fetchCatalog());
    if (topic)
        flat = flat.filter((item) => item.topic.toLowerCase().includes(topic));
    const payload = envelope("indicators list", CATALOG_URL, { limit, topic });
    payload.summary = { returned: Math.min(limit, flat.length), available: flat.length, topicFilter: topic };
    payload.items = compactIndicators(flat, limit);
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ['regionalatlasctl indicators search --term "Arbeitslosenquote" --limit 5'];
    emit(payload);
}
async function runIndicatorsSearch(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", "indicators search requires --term");
    const limit = limitFlag(parsed, 5, 50);
    const matchesFound = searchCatalog(flattenCatalog(await fetchCatalog()), term);
    const payload = envelope("indicators search", CATALOG_URL, { term, limit });
    payload.summary = { term, matches: matchesFound.length, returned: Math.min(limit, matchesFound.length) };
    payload.items = compactIndicators(matchesFound, limit);
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = nextActionsForIndicators(matchesFound);
    emit(payload);
}
async function runIndicatorGet(argv) {
    const parsed = parseArgs(argv);
    const item = await findIndicator(requiredIndicator(parsed));
    const field = firstAttributeCode(item.node);
    const year = latestYear(item.node);
    const payload = envelope("indicator get", CATALOG_URL, { indicator: item.node.code });
    payload.summary = indicatorSummary(item);
    payload.items = compactAttributes(item.node.attributes ?? [], 50);
    payload.sources = sourcesForIndicator(item.node, field, year);
    payload.warnings = defaultWarnings();
    payload.nextActions = [
        `regionalatlasctl fields --indicator ${item.node.code}`,
        `regionalatlasctl sample --indicator ${item.node.code} --field ${field} --year ${year} --region-level 1 --limit 5`
    ];
    emit(payload);
}
async function runFields(argv) {
    const parsed = parseArgs(argv);
    const item = await findIndicator(requiredIndicator(parsed));
    const node = item.node;
    const field = firstAttributeCode(node);
    const payload = envelope("fields", CATALOG_URL, { indicator: node.code });
    payload.summary = {
        indicator: node.code,
        title: node.title_short,
        topic: item.topic,
        availableYears: availableYears(node),
        latestYear: latestYear(node),
        regionLevels: regionLevelAvailability(),
        attributeCount: (node.attributes ?? []).length,
        regionalDbTable: node.code
    };
    payload.items = compactAttributes(node.attributes ?? [], 100);
    payload.sources = sourcesForIndicator(node, field, latestYear(node));
    payload.warnings = defaultWarnings();
    payload.nextActions = [
        `regionalatlasctl explain-field --indicator ${node.code} --field ${field}`,
        `regionalatlasctl sample --indicator ${node.code} --field ${field} --year ${latestYear(node)} --region-level 1 --limit 5`
    ];
    emit(payload);
}
async function runSample(argv) {
    const parsed = parseArgs(argv);
    const { item, field, year, regionLevel } = await resolveQueryInputs(parsed);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const params = buildQueryParams(item.node, field, year, regionLevel, limit, parsed);
    const requestUrl = `${QUERY_ENDPOINT}?${params.toString()}`;
    const data = await fetchJson(requestUrl);
    const warnings = defaultWarnings();
    if (data.exceededTransferLimit)
        warnings.push("ArcGIS reported exceededTransferLimit=true; the returned sample is not a complete extract.");
    if (flagBool(parsed, "geometry"))
        warnings.push("Geometry was requested intentionally; municipality-level geometry can be very large.");
    const items = compactFeatures(data, flagBool(parsed, "geometry"));
    const payload = envelope("sample", requestUrl, { indicator: item.node.code, field: field.toUpperCase(), year, regionLevel, limit });
    payload.summary = {
        indicator: item.node.code,
        field: field.toUpperCase(),
        fieldTitle: attributeTitle(item.node, field),
        unit: attributeUnit(item.node, field),
        year,
        regionLevel,
        regionLevelLabel: regionLevelLabel(regionLevel),
        returned: items.length,
        limitApplied: limit,
        returnGeometry: flagBool(parsed, "geometry"),
        exceededTransferLimit: Boolean(data.exceededTransferLimit)
    };
    payload.items = items;
    payload.sources = sourcesForIndicator(item.node, field, year);
    payload.warnings = warnings;
    payload.nextActions = [
        `regionalatlasctl query-builder --indicator ${item.node.code} --field ${field.toUpperCase()} --year ${year} --region-level ${regionLevel} --limit ${limit}`,
        `regionalatlasctl explain-field --indicator ${item.node.code} --field ${field.toUpperCase()}`
    ];
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    emit(payload);
}
async function runSource(argv) {
    const parsed = parseArgs(argv);
    const item = await findIndicator(requiredIndicator(parsed));
    const node = item.node;
    const field = firstNonEmpty(parsed.flags.field, firstAttributeCode(node));
    const year = intFlag(parsed, "year", latestYear(node));
    const payload = envelope("source", CATALOG_URL, { indicator: node.code, field, year });
    payload.summary = { indicator: node.code, title: node.title_short, field: field.toUpperCase(), year, regionalDbTable: node.code, authRequired: false };
    payload.sources = sourcesForIndicator(node, field, year);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`regionalatlasctl dossier --indicator ${node.code} --field ${field.toUpperCase()} --year ${year}`];
    emit(payload);
}
async function runDossier(argv) {
    const parsed = parseArgs(argv);
    const { item, field, year, regionLevel } = await resolveQueryInputs(parsed);
    const limit = limitFlag(parsed, 5, 25);
    const params = buildQueryParams(item.node, field, year, regionLevel, limit, parsed);
    const requestUrl = `${QUERY_ENDPOINT}?${params.toString()}`;
    const warnings = defaultWarnings();
    let sampleData;
    try {
        sampleData = await fetchJson(requestUrl);
    }
    catch (error) {
        warnings.push(`sampleQuery: ${error instanceof Error ? error.message : String(error)}`);
    }
    const payload = envelope("dossier", requestUrl, { indicator: item.node.code, field: field.toUpperCase(), year, regionLevel, limit });
    payload.summary = {
        indicator: item.node.code,
        title: item.node.title_short,
        topic: item.topic,
        field: field.toUpperCase(),
        fieldTitle: attributeTitle(item.node, field),
        unit: attributeUnit(item.node, field),
        year,
        regionLevel,
        regionLevelLabel: regionLevelLabel(regionLevel),
        availableYears: availableYears(item.node),
        regionLevels: regionLevelAvailability()
    };
    payload.fields = compactAttributes(item.node.attributes ?? [], 100);
    payload.metadata = { indicatorTitleLong: item.node.title_long, fieldMetaSnippets: metaSnippets(attributeMeta(item.node, field), "", 6) };
    if (sampleData) {
        payload.sample = { items: compactFeatures(sampleData, flagBool(parsed, "geometry")), exceededTransferLimit: Boolean(sampleData.exceededTransferLimit) };
        if (sampleData.exceededTransferLimit)
            warnings.push("Sample query reports exceededTransferLimit=true; use pagination/filtering for complete extraction.");
    }
    payload.sources = sourcesForIndicator(item.node, field, year);
    payload.warnings = warnings;
    payload.nextActions = [
        `regionalatlasctl explain-field --indicator ${item.node.code} --field ${field.toUpperCase()} --grep Quelle`,
        `regionalatlasctl query-builder --indicator ${item.node.code} --field ${field.toUpperCase()} --year ${year} --region-level ${regionLevel}`
    ];
    emit(payload);
}
async function runQueryBuilder(argv) {
    const parsed = parseArgs(argv);
    const { item, field, year, regionLevel } = await resolveQueryInputs(parsed);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const params = buildQueryParams(item.node, field, year, regionLevel, limit, parsed);
    const requestUrl = `${QUERY_ENDPOINT}?${params.toString()}`;
    const payload = envelope("query-builder", requestUrl, { indicator: item.node.code, field: field.toUpperCase(), year, regionLevel, limit });
    payload.summary = {
        indicator: item.node.code,
        field: field.toUpperCase(),
        year,
        regionLevel,
        regionLevelLabel: regionLevelLabel(regionLevel),
        requestUrl,
        layerJson: params.get("layer"),
        doesNotFetch: true
    };
    payload.sources = sourcesForIndicator(item.node, field, year);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`regionalatlasctl sample --indicator ${item.node.code} --field ${field.toUpperCase()} --year ${year} --region-level ${regionLevel} --limit ${limit}`];
    emit(payload);
}
async function runExplainField(argv) {
    const parsed = parseArgs(argv);
    const item = await findIndicator(requiredIndicator(parsed));
    const node = item.node;
    const field = firstNonEmpty(parsed.flags.field, parsed.flags.name, firstPosition(parsed), firstAttributeCode(node));
    const attr = findAttribute(node, field);
    if (!attr)
        throw new CLIError(2, "field_not_found", "field not found in indicator attributes");
    const grep = firstNonEmpty(parsed.flags.grep);
    const payload = envelope("explain-field", CATALOG_URL, { indicator: node.code, field: field.toUpperCase(), grep });
    payload.summary = {
        indicator: node.code,
        field: String(attr.code ?? "").toUpperCase(),
        title: attr.title_short,
        titleLong: attr.title_long,
        unit: attr.unit
    };
    payload.items = metaSnippets(attr.meta ?? "", grep, 10);
    payload.sources = sourcesForIndicator(node, attr.code ?? "", latestYear(node));
    payload.warnings = defaultWarnings();
    payload.nextActions = [`regionalatlasctl sample --indicator ${node.code} --field ${String(attr.code ?? "").toUpperCase()} --year ${latestYear(node)} --region-level 1 --limit 5`];
    emit(payload);
}
async function runRawQuery(argv) {
    const parsed = parseArgs(argv);
    if (!parsed.params.layer && !parsed.flags.layer && !parsed.flags["layer-file"]) {
        throw new CLIError(2, "missing_layer", "raw query requires --param layer=<json>, --layer-file <path>, or use query-builder/sample");
    }
    const params = new URLSearchParams(parsed.params);
    if (parsed.flags["layer-file"]) {
        try {
            params.set("layer", (0, node_fs_1.readFileSync)(parsed.flags["layer-file"], "utf8").replace(/^\uFEFF/, "").trim());
        }
        catch (error) {
            throw new CLIError(2, "layer_file_read_failed", error instanceof Error ? error.message : String(error));
        }
    }
    if (parsed.flags.layer)
        params.set("layer", parsed.flags.layer);
    if (!params.get("f"))
        params.set("f", "json");
    if (!params.get("returnGeometry") && !params.get("returngeometry"))
        params.set("returnGeometry", "false");
    if (!params.get("where"))
        params.set("where", "1=1");
    if (!params.get("spatialRel"))
        params.set("spatialRel", "esriSpatialRelIntersects");
    if (!params.get("resultRecordCount") && !params.get("resultrecordcount"))
        params.set("resultRecordCount", String(limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT)));
    const rawLimit = intValue(firstNonEmpty(params.get("resultRecordCount"), params.get("resultrecordcount")));
    if (rawLimit > SAFE_LIMIT && !flagBool(parsed, "allow-large-output")) {
        throw new CLIError(2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override");
    }
    emit(await fetchJson(`${QUERY_ENDPOINT}?${params.toString()}`));
}
function parseArgs(args) {
    const parsed = { flags: {}, params: {}, positionals: [] };
    for (let i = 0; i < args.length; i += 1) {
        const arg = args[i];
        if (!arg.startsWith("--")) {
            parsed.positionals.push(arg);
            continue;
        }
        let keyValue = arg.slice(2);
        let key;
        let value;
        if (keyValue.includes("=")) {
            const splitAt = keyValue.indexOf("=");
            key = keyValue.slice(0, splitAt);
            value = keyValue.slice(splitAt + 1);
        }
        else if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
            key = keyValue;
            value = args[i + 1];
            i += 1;
        }
        else {
            key = keyValue;
            value = "true";
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
async function fetchCatalog() {
    const data = await fetchJson(CATALOG_URL);
    if (!Array.isArray(data))
        throw new CLIError(1, "catalog_shape_changed", "Regionalatlas catalog is not an array");
    return data;
}
async function fetchJson(requestUrl) {
    const response = await fetch(requestUrl, {
        headers: { "User-Agent": "germany-skills/regionalatlasctl-node-2.0" },
        signal: AbortSignal.timeout(45000)
    });
    const body = await response.text();
    if (!response.ok)
        throw new Error(`HTTP ${response.status} from ${requestUrl}: ${body.slice(0, 300)}`);
    const data = JSON.parse(body);
    if (data && typeof data === "object" && data.error)
        throw new Error(`upstream error ${JSON.stringify(data.error)}`);
    return data;
}
function requiredIndicator(parsed) {
    const code = firstNonEmpty(parsed.flags.indicator, parsed.flags.code, parsed.flags.table, firstPosition(parsed));
    if (!code)
        throw new CLIError(2, "missing_indicator", "command requires --indicator");
    return code.toUpperCase();
}
async function findIndicator(code) {
    const norm = normalizeCode(code);
    for (const item of flattenCatalog(await fetchCatalog())) {
        if (normalizeCode(String(item.node.code ?? "")) === norm)
            return item;
    }
    throw new CLIError(2, "indicator_not_found", "indicator not found in Regionalatlas catalog");
}
async function resolveQueryInputs(parsed) {
    const item = await findIndicator(requiredIndicator(parsed));
    const field = firstNonEmpty(parsed.flags.field, parsed.flags.icode, firstAttributeCode(item.node)).toLowerCase();
    if (!findAttribute(item.node, field))
        throw new CLIError(2, "field_not_found", "field not found for indicator");
    const year = intFlag(parsed, "year", latestYear(item.node));
    if (!year)
        throw new CLIError(2, "missing_year", "year could not be inferred; pass --year");
    const regionLevel = intFlag(parsed, "region-level", intFlag(parsed, "typ", 1));
    if (![1, 2, 3, 5].includes(regionLevel))
        throw new CLIError(2, "invalid_region_level", "region-level must be one of 1, 2, 3, or 5");
    return { item, field, year, regionLevel };
}
function buildQueryParams(node, field, year, regionLevel, limit, parsed) {
    const table = tableName(String(node.code ?? ""));
    const geoYear = intFlag(parsed, "geo-year", year);
    const sql = `SELECT * FROM verwaltungsgrenzen_gesamt LEFT OUTER JOIN ${table} ON ags = ags2 and jahr = jahr2 WHERE typ = ${regionLevel} AND jahr = ${geoYear} AND (jahr2 = ${year} OR jahr2 IS NULL)`;
    const layer = {
        source: {
            dataSource: {
                geometryType: "esriGeometryPolygon",
                workspaceId: "gdb",
                query: sql,
                oidFields: "id",
                spatialReference: { wkid: 25832 },
                type: "queryTable"
            },
            type: "dataLayer"
        }
    };
    let where = firstNonEmpty(parsed.flags.where, parsed.params.where, "1=1");
    if (parsed.flags.ags && where === "1=1")
        where = `ags = '${parsed.flags.ags.replace(/'/g, "''")}'`;
    const params = new URLSearchParams();
    params.set("layer", JSON.stringify(layer));
    params.set("f", "json");
    params.set("outFields", firstNonEmpty(parsed.flags.fields, parsed.params.outFields, `ags,gen,typ,jahr,jahr2,ags2,gen2,${field.toLowerCase()}`));
    params.set("returnGeometry", flagBool(parsed, "geometry") ? "true" : "false");
    params.set("spatialRel", "esriSpatialRelIntersects");
    params.set("where", where);
    params.set("resultRecordCount", String(limit));
    for (const [key, value] of Object.entries(parsed.params)) {
        if (key !== "layer")
            params.set(key, value);
    }
    return params;
}
function flattenCatalog(catalog) {
    const flat = [];
    const walk = (nodes, topic = "") => {
        for (const node of nodes ?? []) {
            let nextTopic = topic;
            if (node?.title && !node?.code)
                nextTopic = String(node.title);
            if (node?.code && node?.attributes)
                flat.push({ topic: nextTopic, node });
            if (Array.isArray(node?.children))
                walk(node.children, nextTopic);
        }
    };
    walk(catalog);
    return flat.sort((left, right) => String(left.node.code ?? "").localeCompare(String(right.node.code ?? "")));
}
function searchCatalog(flat, term) {
    const needle = term.toLowerCase();
    return flat.filter((item) => {
        const node = item.node;
        let hay = [item.topic, node.code, node.title_short, node.title_long].map((value) => String(value ?? "")).join(" ").toLowerCase();
        for (const attr of node.attributes ?? []) {
            hay += ` ${[attr.code, attr.title_short, attr.title_long, attr.unit, stripWiki(attr.meta ?? "")].map((value) => String(value ?? "")).join(" ").toLowerCase()}`;
        }
        return hay.includes(needle);
    });
}
function compactIndicators(items, limit) {
    return items.slice(0, limit).map((item) => {
        const node = item.node;
        const field = firstAttributeCode(node);
        const year = latestYear(node);
        return {
            code: node.code,
            table: tableName(String(node.code ?? "")),
            topic: item.topic,
            title: node.title_short,
            titleLong: node.title_long,
            latestYear: year,
            availableYears: availableYears(node),
            attributes: compactAttributes(node.attributes ?? [], 8),
            nextActions: [
                `regionalatlasctl fields --indicator ${node.code}`,
                `regionalatlasctl sample --indicator ${node.code} --field ${field} --year ${year} --region-level 1 --limit 5`
            ]
        };
    });
}
function compactAttributes(attrs, limit) {
    return attrs.slice(0, limit).map((attr) => ({
        code: String(attr.code ?? "").toUpperCase(),
        field: String(attr.code ?? "").toLowerCase(),
        title: attr.title_short,
        titleLong: attr.title_long,
        unit: attr.unit,
        metaPreview: truncate(stripWiki(String(attr.meta ?? "")), 500)
    }));
}
function compactFeatures(data, includeGeometry) {
    return (data.features ?? []).map((feature) => {
        const item = { attributes: normalizeAttributes(feature.attributes ?? {}) };
        if (includeGeometry)
            item.geometry = feature.geometry;
        return item;
    });
}
function normalizeAttributes(attrs) {
    const clean = {};
    for (const [key, value] of Object.entries(attrs)) {
        if (key.toLowerCase().endsWith(".shape"))
            continue;
        clean[key] = typeof value === "string" ? value.trim() : value;
    }
    return clean;
}
function indicatorSummary(item) {
    const node = item.node;
    return {
        code: node.code,
        table: tableName(String(node.code ?? "")),
        topic: item.topic,
        title: node.title_short,
        titleLong: node.title_long,
        timestamp: node.timestamp,
        latestYear: latestYear(node),
        availableYears: availableYears(node),
        regionLevels: regionLevelAvailability(),
        attributeCount: (node.attributes ?? []).length
    };
}
function availableYears(node) {
    return Object.keys(node.years ?? {})
        .map((key) => Number.parseInt(key, 10))
        .filter((value) => Number.isFinite(value))
        .sort((left, right) => left - right);
}
function latestYear(node) {
    const years = availableYears(node);
    return years.length ? years[years.length - 1] : 0;
}
function regionLevelAvailability() {
    return {
        "1": { label: regionLevelLabel(1), appearsAvailableLatestYear: true },
        "2": { label: regionLevelLabel(2), appearsAvailableLatestYear: true },
        "3": { label: regionLevelLabel(3), appearsAvailableLatestYear: true },
        "5": { label: regionLevelLabel(5), appearsAvailableLatestYear: true }
    };
}
function findAttribute(node, code) {
    return (node.attributes ?? []).find((attr) => String(attr.code ?? "").toLowerCase() === code.toLowerCase());
}
function firstAttributeCode(node) {
    const attrs = node.attributes ?? [];
    for (const attr of attrs) {
        const code = String(attr.code ?? "");
        if (code && !code.toLowerCase().endsWith("v"))
            return code.toUpperCase();
    }
    return attrs.length ? String(attrs[0].code ?? "").toUpperCase() : "";
}
function attributeTitle(node, code) {
    return String(findAttribute(node, code)?.title_short ?? "");
}
function attributeUnit(node, code) {
    return String(findAttribute(node, code)?.unit ?? "");
}
function attributeMeta(node, code) {
    return String(findAttribute(node, code)?.meta ?? "");
}
function metaSnippets(meta, grep, limit) {
    const needle = grep.toLowerCase();
    const lines = stripWiki(meta)
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line.length > 10);
    const snippets = [];
    for (const line of lines) {
        if (!needle || line.toLowerCase().includes(needle))
            snippets.push({ text: truncate(line, 700) });
        if (snippets.length >= limit)
            break;
    }
    return snippets;
}
function stripWiki(value) {
    return value
        .replace(/^wiki/, "")
        .replace(/===/g, "")
        .replace(/==/g, "")
        .replace(/'''/g, "")
        .replace(/''/g, "")
        .replace(/\*/g, "")
        .replace(/\s+\|/g, " |")
        .trim();
}
function sourcesForIndicator(node, field, year) {
    const deepLink = `${APP_URL}?${new URLSearchParams({ BL: "DE", TCode: String(node.code ?? ""), ICode: field.toUpperCase(), Jhr: String(year) }).toString()}`;
    return [
        { title: "Regionalatlas app", url: deepLink, kind: "interactive_atlas" },
        { title: "Regionalatlas Statistikportal page", url: STATISTIKPORTAL_URL, kind: "official_context" },
        { title: "Destatis Regionalatlas page", url: DESTATIS_URL, kind: "official_context" },
        { title: "Statistikportal Open Data", url: OPEN_DATA_URL, kind: "terms_and_downloads" },
        { title: "Destatis maps and geodata", url: MAPS_GEODATA_URL, kind: "terms_and_downloads" },
        { title: "Regionalatlas catalog JSON", url: CATALOG_URL, kind: "catalog" },
        { title: "Regionaldatenbank table", url: `https://www.regionalstatistik.de/genesis/online/data?operation=table&code=${String(node.code ?? "")}`, kind: "official_table" },
        { title: "ArcGIS dynamic-layer query endpoint", url: QUERY_ENDPOINT, kind: "api_endpoint" },
        { title: "bundesAPI Regionalatlas OpenAPI wrapper", url: OPENAPI_REPO_URL, kind: "openapi_reference" }
    ];
}
function defaultSources() {
    return [
        { title: "Regionalatlas Statistikportal page", url: STATISTIKPORTAL_URL, kind: "official_context" },
        { title: "Destatis Regionalatlas page", url: DESTATIS_URL, kind: "official_context" },
        { title: "Statistikportal Open Data", url: OPEN_DATA_URL, kind: "terms_and_downloads" },
        { title: "Regionalatlas catalog JSON", url: CATALOG_URL, kind: "catalog" },
        { title: "Regionalatlas thesaurus CSV", url: THESAURUS_URL, kind: "catalog" },
        { title: "ArcGIS MapServer metadata", url: `${MAP_SERVER_URL}?f=json`, kind: "api_metadata" },
        { title: "bundesAPI Regionalatlas OpenAPI wrapper", url: OPENAPI_REPO_URL, kind: "openapi_reference" }
    ];
}
function defaultWarnings() {
    return [
        "No exact published API rate limit was found in reviewed materials; keep requests small and cache catalog metadata.",
        "The ArcGIS service advertises a very high maxRecordCount; never run broad municipality-level pulls accidentally.",
        "Use field metadata for units, definitions, source statistics, and regional caveats before interpreting values.",
        "Statistikportal Open Data notes point to Datenlizenz Deutschland 2.0 for statistical data and atlas/imprint license hints for geodata."
    ];
}
function nextActionsForIndicators(items) {
    const actions = items.slice(0, 3).map((item) => {
        const node = item.node;
        return `regionalatlasctl dossier --indicator ${node.code} --field ${firstAttributeCode(node)} --year ${latestYear(node)} --region-level 1`;
    });
    return actions.length ? actions : ['regionalatlasctl indicators search --term "Bevoelkerung" --limit 5'];
}
function mapServerSummary(data) {
    return {
        mapName: data.mapName,
        supportsDynamicLayers: Boolean(data.supportsDynamicLayers),
        supportedQueryFormats: data.supportedQueryFormats,
        maxRecordCount: intValue(data.maxRecordCount),
        capabilities: data.capabilities,
        featureLayerCount: (data.layers ?? []).length,
        spatialReferenceLatest: intValue(data.spatialReference?.latestWkid)
    };
}
function envelope(command, requestUrl, request) {
    return {
        status: "ok",
        tool: APP_NAME,
        command,
        retrievedAt: new Date().toISOString(),
        request: { method: "GET", url: requestUrl, params: request },
        summary: {},
        items: [],
        sources: [],
        warnings: [],
        nextActions: []
    };
}
function emit(value) {
    console.log(JSON.stringify(value, null, 2));
}
function fail(exitCode, code, message) {
    emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message } });
    process.exitCode = exitCode;
}
function isHelp(value) {
    return value === "--help" || value === "-h" || value === "help";
}
function matches(argv, ...expected) {
    return expected.every((value, index) => argv[index] === value);
}
function firstNonEmpty(...values) {
    for (const value of values) {
        if (value !== undefined && value !== null && String(value).trim())
            return String(value).trim();
    }
    return "";
}
function firstPosition(parsed) {
    return parsed.positionals.length ? parsed.positionals[0] : "";
}
function flagBool(parsed, key) {
    return ["true", "1", "yes", "y"].includes(String(parsed.flags[key] ?? "").toLowerCase());
}
function limitFlag(parsed, fallback, maxValue) {
    const raw = firstNonEmpty(parsed.flags.limit, parsed.flags.resultrecordcount, parsed.params.resultRecordCount, parsed.params.resultrecordcount);
    let value = raw ? intValue(raw) : fallback;
    if (value < 1)
        value = fallback;
    if (value > maxValue && !flagBool(parsed, "allow-large-output")) {
        throw new CLIError(2, "limit_exceeds_safe_max", `limit ${value} exceeds safe max ${maxValue}; pass --allow-large-output to override`);
    }
    return value;
}
function intFlag(parsed, key, fallback) {
    return intValue(parsed.flags[key]) || fallback;
}
function intValue(value) {
    const parsed = Number.parseInt(String(value ?? ""), 10);
    return Number.isFinite(parsed) ? parsed : 0;
}
function tableName(code) {
    return code.replace(/-/g, "_").toLowerCase();
}
function normalizeCode(code) {
    return code.trim().replace(/_/g, "-").toUpperCase();
}
function regionLevelLabel(level) {
    const labels = {
        1: "Laender",
        2: "Regierungsbezirke/statistical regions",
        3: "Kreise and kreisfreie Staedte",
        5: "Gemeinden/Gemeindeverbaende"
    };
    return labels[level] ?? "unknown";
}
function truncate(value, maxLen) {
    return value.length <= maxLen ? value : `${value.slice(0, maxLen)}...`;
}
main(process.argv.slice(2)).then((code) => {
    process.exitCode = code;
});
