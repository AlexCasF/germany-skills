const APP_NAME = "deutschlandatlasctl";
const PORTAL_SEARCH_BASE = "https://www.karto365.de/portal/sharing/rest/search";
const HOSTING_BASE = "https://www.karto365.de/hosting/rest/services";
const OFFICIAL_HOME_URL = "https://www.deutschlandatlas.bund.de/DE/Home/home_node.html";
const OFFICIAL_DOWNLOADS_URL = "https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html";
const GITHUB_SPEC_URL = "https://github.com/bundesAPI/deutschlandatlas-api";
const DEFAULT_LIMIT = 10;
const SAFE_LIMIT = 100;

type JsonObject = Record<string, any>;
type ParsedArgs = { flags: Record<string, string>; params: Record<string, string>; positionals: string[] };

class CLIError extends Error {
  exitCode: number;
  code: string;
  constructor(exitCode: number, code: string, message: string) {
    super(message);
    this.exitCode = exitCode;
    this.code = code;
  }
}

async function main(argv: string[]): Promise<number> {
  if (argv.length === 0 || isHelp(argv[0])) {
    printRootHelp();
    return 0;
  }
  if (isHelp(argv[argv.length - 1])) {
    printHelp(argv.slice(0, -1));
    return 0;
  }
  try {
    if (argv[0] === "doctor") await runDoctor(argv.slice(1));
    else if (matches(argv, "tables", "search")) await runTablesSearch(argv.slice(2));
    else if (matches(argv, "table", "query")) await runTableQuery(argv.slice(2));
    else if (matches(argv, "table", "fields")) await runTableFields(argv.slice(2));
    else if (matches(argv, "table", "sample")) await runTableSample(argv.slice(2));
    else if (matches(argv, "table", "source")) await runTableSource(argv.slice(2));
    else if (matches(argv, "indicator", "dossier")) await runIndicatorDossier(argv.slice(2));
    else if (argv[0] === "query-builder") await runQueryBuilder(argv.slice(1));
    else if (argv[0] === "explain-field") await runExplainField(argv.slice(1));
    else throw new CLIError(2, "unknown_command", "unknown command; run deutschlandatlasctl --help");
  } catch (error) {
    if (error instanceof CLIError) {
      fail(error.exitCode, error.code, error.message);
      return error.exitCode;
    }
    fail(1, "unexpected_error", error instanceof Error ? error.message : String(error));
    return 1;
  }
  return 0;
}

function printRootHelp(): void {
  console.log(`deutschlandatlasctl -- Deutschlandatlas ArcGIS research CLI

Purpose
  Discover and query public Deutschlandatlas indicator map services for
  regional living-condition indicators in Germany.

Fast paths
  deutschlandatlasctl doctor
  deutschlandatlasctl tables search --term "Arbeitslosenquote" --limit 5
  deutschlandatlasctl table fields --table alq_HA2023
  deutschlandatlasctl table sample --table alq_HA2023 --limit 5
  deutschlandatlasctl indicator dossier --table alq_HA2023

Legacy-compatible command
  table query --table <table> [--param key=value] [--layer auto|0|5]

Research commands
  doctor
  tables search
  table fields
  table sample
  table source
  indicator dossier
  query-builder
  explain-field
`);
}

function printHelp(path: string[]): void {
  const joined = path.join(" ");
  if (joined === "table sample") {
    console.log(`deutschlandatlasctl table sample

Fetch a small bounded sample from one Deutschlandatlas ArcGIS table.

Examples
  deutschlandatlasctl table sample --table alq_HA2023 --limit 5
  deutschlandatlasctl table sample --table alq_HA2023 --fields name,alq --where "alq > 10"
`);
    return;
  }
  if (joined === "indicator dossier") {
    console.log(`deutschlandatlasctl indicator dossier

Bundle metadata, selected layer, fields, source URLs, warnings, and a tiny sample.
`);
    return;
  }
  if (joined === "tables search") {
    console.log(`deutschlandatlasctl tables search

Search the public ArcGIS portal for Deutschlandatlas table services.
`);
    return;
  }
  printRootHelp();
}

async function runDoctor(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const limit = limitFlag(parsed, 1, 10);
  const searchUrl = portalSearchUrl("", limit, 1);
  const payload = envelope("doctor", searchUrl, undefined);
  const summary: JsonObject = {
    authRequired: false,
    publishedRateLimit: "No exact public rate limit found in reviewed Deutschlandatlas/API materials. Use small limits, cache metadata, and avoid parallel broad ArcGIS queries.",
    fairUseHints: [
      "Prefer tables search, fields, and small samples before broad queries.",
      "Do not request geometry unless map geometry is needed.",
      "Respect ArcGIS transfer limits and back off on 429, 5xx, or slow responses."
    ]
  };
  const warnings = defaultWarnings();
  try {
    const searchData = await fetchJson(searchUrl);
    summary.portalSearchReachable = true;
    summary.portalTotal = intValue(searchData.total);
  } catch (error) {
    summary.portalSearchReachable = false;
    warnings.push(`portalSearch: ${error instanceof Error ? error.message : String(error)}`);
  }
  try {
    const serviceData = await fetchJson(serviceUrl("alq_HA2023"));
    summary.sampleServiceReachable = true;
    summary.sampleService = serviceSummary(serviceData);
  } catch (error) {
    summary.sampleServiceReachable = false;
    warnings.push(`sampleService: ${error instanceof Error ? error.message : String(error)}`);
  }
  payload.status = summary.portalSearchReachable && summary.sampleServiceReachable ? "ok" : "degraded";
  payload.summary = summary;
  payload.sources = defaultSources();
  payload.warnings = warnings;
  payload.nextActions = ['deutschlandatlasctl tables search --term "Arbeitslosenquote" --limit 5', "deutschlandatlasctl indicator dossier --table alq_HA2023"];
  emit(payload);
}

async function runTablesSearch(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
  if (!term) throw new CLIError(2, "missing_term", "tables search requires --term");
  const limit = limitFlag(parsed, 5, 25);
  const start = intFlag(parsed, "start", 1);
  const requestUrl = portalSearchUrl(term, limit, start);
  const data = await fetchJson(requestUrl);
  const items = compactPortalResults(data, limit);
  const payload = envelope("tables search", requestUrl, { term, limit, start });
  payload.summary = { term, total: intValue(data.total), returned: items.length, limitApplied: limit, nextStart: intValue(data.nextStart) };
  payload.items = items;
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = nextActionsForTables(items);
  emit(payload);
}

async function runTableQuery(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const [layer] = await resolveLayer(table, parsed);
  const params: Record<string, string> = {
    f: firstNonEmpty(parsed.params.f, "json"),
    where: firstNonEmpty(parsed.params.where, parsed.flags.where, "1=1"),
    outFields: firstNonEmpty(parsed.params.outFields, parsed.params.outfields, parsed.flags.fields, "*"),
    returnGeometry: firstNonEmpty(parsed.params.returnGeometry, parsed.params.returngeometry, boolString(flagBool(parsed, "geometry")))
  };
  Object.assign(params, parsed.params);
  if (!params.resultRecordCount && !params.resultrecordcount) params.resultRecordCount = String(limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT));
  if (!flagBool(parsed, "allow-large-output")) {
    const count = intValue(firstNonEmpty(params.resultRecordCount, params.resultrecordcount));
    if (count > SAFE_LIMIT) throw new CLIError(2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override");
  }
  emit(await fetchJson(queryUrl(table, layer, params)));
}

async function runTableFields(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const [layer, layerSource] = await resolveLayer(table, parsed);
  const requestUrl = layerUrl(table, layer);
  const layerData = await fetchJson(requestUrl);
  const fields = compactFields(layerData);
  const payload = envelope("table fields", requestUrl, { table, layer });
  payload.summary = {
    table,
    layer,
    layerSource,
    fieldCount: fields.length,
    displayField: layerData.displayField,
    objectIdField: layerData.objectIdField,
    geometryType: layerData.geometryType,
    maxRecordCount: intValue(layerData.maxRecordCount),
    likelyIndicatorFields: likelyIndicatorFields(fields)
  };
  payload.items = fields;
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = defaultWarnings();
  payload.nextActions = [`deutschlandatlasctl table sample --table ${table} --fields name,${firstLikelyIndicator(fields)} --limit 5`, `deutschlandatlasctl indicator dossier --table ${table}`];
  emit(payload);
}

async function runTableSample(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
  const [layer, layerSource] = await resolveLayer(table, parsed);
  const params: Record<string, string> = {
    f: "json",
    where: firstNonEmpty(parsed.params.where, parsed.flags.where, "1=1"),
    outFields: firstNonEmpty(parsed.params.outFields, parsed.params.outfields, parsed.flags.fields, "*"),
    returnGeometry: boolString(flagBool(parsed, "geometry")),
    resultRecordCount: String(limit)
  };
  Object.assign(params, parsed.params);
  const requestUrl = queryUrl(table, layer, params);
  const data = await fetchJson(requestUrl);
  const items = compactFeatures(data, flagBool(parsed, "geometry"));
  const warnings = defaultWarnings();
  if (data.exceededTransferLimit) warnings.push("ArcGIS reported exceededTransferLimit=true; narrow the where clause or paginate deliberately.");
  if (flagBool(parsed, "geometry")) warnings.push("Geometry was requested intentionally; outputs can grow quickly.");
  const payload = envelope("table sample", requestUrl, { table, layer, limit });
  payload.summary = { table, layer, layerSource, returned: items.length, limitApplied: limit, returnGeometry: flagBool(parsed, "geometry"), exceededTransferLimit: Boolean(data.exceededTransferLimit), displayField: data.displayFieldName };
  payload.items = items;
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = warnings;
  payload.nextActions = [`deutschlandatlasctl table fields --table ${table}`, `deutschlandatlasctl query-builder --table ${table} --where "name LIKE '%Berlin%'" --fields name,* --limit 10`];
  if (flagBool(parsed, "include-raw")) payload.raw = data;
  emit(payload);
}

async function runTableSource(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const [layer, layerSource] = flagBool(parsed, "skip-layer-discovery") ? [0, "legacy_default"] as [number, string] : await resolveLayer(table, parsed);
  const payload = envelope("table source", serviceUrl(table), { table, layer });
  payload.summary = { table, selectedLayer: layer, layerSource, authRequired: false, apiStyle: "ArcGIS REST MapServer query endpoint", rateLimitFound: false };
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = defaultWarnings();
  payload.nextActions = [`deutschlandatlasctl table fields --table ${table}`, `deutschlandatlasctl table sample --table ${table} --limit 5`];
  emit(payload);
}

async function runIndicatorDossier(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const limit = limitFlag(parsed, 5, 25);
  const [layer, layerSource] = await resolveLayer(table, parsed);
  const payload = envelope("indicator dossier", serviceUrl(table), { table, layer, limit });
  const warnings = defaultWarnings();
  let fields: JsonObject[] = [];
  payload.summary = { table, selectedLayer: layer, layerSource, limitApplied: limit, authRequired: false };
  try {
    payload.service = serviceSummary(await fetchJson(serviceUrl(table)));
  } catch (error) {
    warnings.push(`serviceMetadata: ${error instanceof Error ? error.message : String(error)}`);
  }
  try {
    fields = compactFields(await fetchJson(layerUrl(table, layer)));
    payload.fields = fields;
    payload.summary.likelyIndicatorFields = likelyIndicatorFields(fields);
  } catch (error) {
    warnings.push(`layerMetadata: ${error instanceof Error ? error.message : String(error)}`);
  }
  try {
    const sample = await fetchJson(queryUrl(table, layer, { f: "json", where: "1=1", outFields: "*", returnGeometry: "false", resultRecordCount: String(limit) }));
    payload.sample = { items: compactFeatures(sample, false), exceededTransferLimit: Boolean(sample.exceededTransferLimit) };
    if (sample.exceededTransferLimit) warnings.push("Sample query reports exceededTransferLimit=true; use pagination/filtering for full extraction.");
  } catch (error) {
    warnings.push(`sampleQuery: ${error instanceof Error ? error.message : String(error)}`);
  }
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = warnings;
  payload.nextActions = [`deutschlandatlasctl table fields --table ${table}`, `deutschlandatlasctl table sample --table ${table} --fields name,* --where "1=1" --limit 10`, `deutschlandatlasctl explain-field --table ${table} --field ${firstLikelyIndicator(fields)}`];
  emit(payload);
}

async function runQueryBuilder(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
  const [layer, layerSource] = await resolveLayer(table, parsed);
  let where = firstNonEmpty(parsed.flags.where, "1=1");
  if (parsed.flags.region && where === "1=1") where = `name LIKE '%${parsed.flags.region.replace(/'/g, "''")}%'`;
  const params: Record<string, string> = { f: "json", where, outFields: firstNonEmpty(parsed.flags.fields, "*"), returnGeometry: boolString(flagBool(parsed, "geometry")), resultRecordCount: String(limit) };
  Object.assign(params, parsed.params);
  const builtUrl = queryUrl(table, layer, params);
  const payload = envelope("query-builder", builtUrl, { table, layer, params });
  payload.summary = { table, layer, layerSource, requestUrl: builtUrl, doesNotFetch: true, limitApplied: limit, returnGeometry: flagBool(parsed, "geometry") };
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = defaultWarnings();
  if (parsed.flags.year) payload.warnings.push("Generic Deutschlandatlas services do not expose one standard year parameter; choose a year-specific table.");
  payload.nextActions = [`deutschlandatlasctl table query --table ${table} --layer ${layer} --param where=${JSON.stringify(where)} --param outFields=${JSON.stringify(params.outFields)} --limit ${limit}`];
  emit(payload);
}

async function runExplainField(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const table = requiredTable(parsed);
  const fieldName = firstNonEmpty(parsed.flags.field, parsed.flags.name, firstPosition(parsed));
  if (!fieldName) throw new CLIError(2, "missing_field", "explain-field requires --field");
  const [layer] = await resolveLayer(table, parsed);
  const fields = compactFields(await fetchJson(layerUrl(table, layer)));
  const match = fields.find((field) => String(field.name).toLowerCase() === fieldName.toLowerCase());
  if (!match) throw new CLIError(2, "field_not_found", "field not found in layer metadata");
  const payload = envelope("explain-field", layerUrl(table, layer), { table, field: fieldName, layer });
  payload.summary = { table, layer, field: match, interpretationHint: "Use the alias, table title/snippet from tables search, and official downloads/method notes for statistical meaning and units." };
  payload.sources = sourcesForTable(table, layer);
  payload.warnings = defaultWarnings();
  payload.nextActions = [`deutschlandatlasctl table sample --table ${table} --fields name,${fieldName} --limit 10`];
  emit(payload);
}

function parseArgs(args: string[]): ParsedArgs {
  const parsed: ParsedArgs = { flags: {}, params: {}, positionals: [] };
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (!arg.startsWith("--")) {
      parsed.positionals.push(arg);
      continue;
    }
    let keyValue = arg.slice(2);
    let key = keyValue;
    let value = "true";
    const equals = keyValue.indexOf("=");
    if (equals >= 0) {
      key = keyValue.slice(0, equals);
      value = keyValue.slice(equals + 1);
    } else if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
      value = args[++i];
    }
    key = key.toLowerCase().trim();
    if (key === "param") {
      const split = value.indexOf("=");
      if (split > 0) parsed.params[value.slice(0, split)] = value.slice(split + 1);
    } else {
      parsed.flags[key] = value;
    }
  }
  return parsed;
}

function requiredTable(parsed: ParsedArgs): string {
  const table = firstNonEmpty(parsed.flags.table, parsed.flags.name, firstPosition(parsed));
  if (!table) throw new CLIError(2, "missing_table", "command requires --table");
  return table;
}

async function resolveLayer(table: string, parsed: ParsedArgs): Promise<[number, string]> {
  if (flagBool(parsed, "legacy-layer-zero")) return [0, "legacy_layer_zero"];
  const layerFlag = firstNonEmpty(parsed.flags.layer, "auto");
  if (layerFlag && layerFlag !== "auto") {
    const layer = Number.parseInt(layerFlag, 10);
    if (!Number.isFinite(layer)) throw new CLIError(2, "invalid_layer", "--layer must be auto or an integer");
    return [layer, "explicit_flag"];
  }
  const data = await fetchJson(serviceUrl(table));
  for (const layer of asArray(data.layers)) {
    if (String(layer.type ?? "").toLowerCase().includes("feature")) return [intValue(layer.id), "service_metadata"];
  }
  if (asArray(data.layers).length > 0) return [intValue(asArray(data.layers)[0].id), "service_metadata"];
  throw new CLIError(1, "no_feature_layer", "service metadata did not expose a feature layer");
}

async function fetchJson(requestUrl: string): Promise<JsonObject> {
  const response = await fetch(requestUrl, { headers: { "User-Agent": "germany-skills/deutschlandatlasctl-node-2.0" } });
  const text = await response.text();
  if (!response.ok) throw new Error(`HTTP ${response.status} from ${requestUrl}: ${text.slice(0, 300)}`);
  const data = JSON.parse(text);
  if (data.error) throw new Error(`upstream error ${JSON.stringify(data.error)}`);
  return data;
}

function portalSearchUrl(term: string, limit: number, start: number): string {
  const query = `deutschlandatlas${term.trim() ? ` ${term.trim()}` : ""}`;
  return `${PORTAL_SEARCH_BASE}?${new URLSearchParams({ q: query, f: "json", num: String(limit), start: String(start) }).toString()}`;
}

function serviceUrl(table: string): string {
  return `${HOSTING_BASE}/${encodeURIComponent(table)}/MapServer?f=json`;
}

function layerUrl(table: string, layer: number): string {
  return `${HOSTING_BASE}/${encodeURIComponent(table)}/MapServer/${layer}?f=json`;
}

function queryUrl(table: string, layer: number, params: Record<string, string>): string {
  return `${HOSTING_BASE}/${encodeURIComponent(table)}/MapServer/${layer}/query?${new URLSearchParams(params).toString()}`;
}

function compactPortalResults(data: JsonObject, limit: number): JsonObject[] {
  return asArray(data.results).slice(0, limit).map((item: JsonObject) => {
    const service = String(item.url ?? "");
    const table = firstNonEmpty(item.title, tableFromUrl(service));
    return {
      table,
      title: item.title,
      snippet: item.snippet,
      type: item.type,
      serviceUrl: service,
      access: item.access,
      tags: item.tags,
      modifiedUtc: millisToUtc(item.modified),
      nextActions: [`deutschlandatlasctl table fields --table ${table}`, `deutschlandatlasctl indicator dossier --table ${table}`]
    };
  });
}

function compactFields(layerData: JsonObject): JsonObject[] {
  return asArray(layerData.fields).map((field: JsonObject) => ({ name: field.name, alias: field.alias, type: field.type, length: intValue(field.length), domain: field.domain }));
}

function compactFeatures(data: JsonObject, includeGeometry: boolean): JsonObject[] {
  return asArray(data.features).map((feature: JsonObject) => {
    const item: JsonObject = { attributes: feature.attributes };
    if (includeGeometry) item.geometry = feature.geometry;
    return item;
  });
}

function likelyIndicatorFields(fields: JsonObject[]): string[] {
  const skip = new Set(["objectid", "shape", "gf", "gen", "bez", "gebietskennziffer", "name", "shape_length", "shape_area"]);
  return fields.map((field) => String(field.name ?? "")).filter((name) => name && !skip.has(name.toLowerCase()) && !name.toLowerCase().startsWith("shape"));
}

function firstLikelyIndicator(fields: JsonObject[]): string {
  const likely = likelyIndicatorFields(fields);
  return likely.length ? likely[0] : "*";
}

function serviceSummary(data: JsonObject): JsonObject {
  return {
    serviceDescription: data.serviceDescription,
    mapName: data.mapName,
    supportedQueryFormats: data.supportedQueryFormats,
    maxRecordCount: intValue(data.maxRecordCount),
    layers: asArray(data.layers).map((layer: JsonObject) => ({ id: intValue(layer.id), name: layer.name, type: layer.type }))
  };
}

function sourcesForTable(table: string, layer: number): JsonObject[] {
  return [
    { title: "Deutschlandatlas start page", url: OFFICIAL_HOME_URL, kind: "official_context" },
    { title: "Deutschlandatlas data downloads and method notes", url: OFFICIAL_DOWNLOADS_URL, kind: "official_downloads" },
    { title: "bundesAPI Deutschlandatlas OpenAPI wrapper", url: GITHUB_SPEC_URL, kind: "openapi_reference" },
    { title: "ArcGIS service metadata", url: serviceUrl(table), kind: "api_service" },
    { title: "ArcGIS layer metadata", url: layerUrl(table, layer), kind: "api_layer" },
    { title: "ArcGIS portal search", url: portalSearchUrl(table, 10, 1), kind: "api_discovery" }
  ];
}

function defaultSources(): JsonObject[] {
  return [
    { title: "Deutschlandatlas start page", url: OFFICIAL_HOME_URL, kind: "official_context" },
    { title: "Deutschlandatlas data downloads and method notes", url: OFFICIAL_DOWNLOADS_URL, kind: "official_downloads" },
    { title: "bundesAPI Deutschlandatlas OpenAPI wrapper", url: GITHUB_SPEC_URL, kind: "openapi_reference" },
    { title: "ArcGIS portal Deutschlandatlas search", url: portalSearchUrl("", 100, 1), kind: "api_discovery" }
  ];
}

function defaultWarnings(): string[] {
  return [
    "No exact published API rate limit was found in reviewed materials; keep requests small and cache stable metadata.",
    "Official download notes state that missing values in tabular downloads are represented as -9999; check field notes before statistical interpretation.",
    "ArcGIS services can enforce maxRecordCount/transfer limits; use filters, fields, and pagination rather than broad full-table pulls."
  ];
}

function nextActionsForTables(items: JsonObject[]): string[] {
  const actions = items.slice(0, 3).filter((item) => item.table).map((item) => `deutschlandatlasctl indicator dossier --table ${item.table}`);
  return actions.length ? actions : ['deutschlandatlasctl tables search --term "Apotheken" --limit 5'];
}

function envelope(command: string, requestUrl: string, request: any): JsonObject {
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

function emit(value: any): void {
  console.log(JSON.stringify(value, null, 2));
}

function fail(exitCode: number, code: string, message: string): void {
  emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message } });
  process.exit(exitCode);
}

function matches(args: string[], ...parts: string[]): boolean {
  return parts.every((part, index) => args[index] === part);
}

function isHelp(value: string): boolean {
  return value === "--help" || value === "-h" || value === "help";
}

function firstNonEmpty(...values: any[]): string {
  for (const value of values) {
    if (value !== undefined && value !== null && String(value).trim()) return String(value).trim();
  }
  return "";
}

function firstPosition(parsed: ParsedArgs): string {
  return parsed.positionals.length ? parsed.positionals[0] : "";
}

function flagBool(parsed: ParsedArgs, key: string): boolean {
  return ["true", "1", "yes", "y"].includes(String(parsed.flags[key] ?? "").toLowerCase());
}

function boolString(value: boolean): string {
  return value ? "true" : "false";
}

function limitFlag(parsed: ParsedArgs, fallback: number, max: number): number {
  const raw = firstNonEmpty(parsed.flags.limit, parsed.flags.resultrecordcount, parsed.params.resultRecordCount, parsed.params.resultrecordcount);
  let value = raw ? Number.parseInt(raw, 10) : fallback;
  if (!Number.isFinite(value) || value < 1) value = fallback;
  if (value > max && !flagBool(parsed, "allow-large-output")) throw new CLIError(2, "limit_exceeds_safe_max", `limit ${value} exceeds safe max ${max}; pass --allow-large-output to override`);
  return value;
}

function intFlag(parsed: ParsedArgs, key: string, fallback: number): number {
  const value = Number.parseInt(parsed.flags[key] ?? "", 10);
  return Number.isFinite(value) ? value : fallback;
}

function intValue(value: any): number {
  const parsed = Number.parseInt(String(value ?? ""), 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

function tableFromUrl(raw: string): string {
  const parts = raw.replace(/^\/+|\/+$/g, "").split("/");
  for (let i = 0; i < parts.length; i++) {
    if (parts[i] === "services" && i + 1 < parts.length) return parts[i + 1];
  }
  return "";
}

function millisToUtc(value: any): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? new Date(parsed).toISOString() : "";
}

function asArray(value: any): JsonObject[] {
  return Array.isArray(value) ? value : [];
}

main(process.argv.slice(2)).then((code) => {
  process.exitCode = code;
});
