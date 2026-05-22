const APP_NAME = "destatisctl";
const BASE_URL = "https://www-genesis.destatis.de/genesisWS/rest/2020";
const UI_URL = "https://www-genesis.destatis.de/datenbank/online";
const DOCS_URL = "https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html";

type JsonObject = Record<string, unknown>;
type ParsedArgs = { flags: Record<string, string>; params: Record<string, string>; positionals: string[] };
type Credentials = { username: string; password: string; source: string; guest: boolean };

const legacyPaths: Record<string, string> = {
  "catalogue statistics": "/catalogue/statistics",
  "catalogue tables": "/catalogue/tables",
  "catalogue variables": "/catalogue/variables",
  "metadata table": "/metadata/table",
  "metadata timeseries": "/metadata/timeseries",
  "data table": "/data/table",
  "data timeseries": "/data/timeseries",
  "find search": "/find/find"
};

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
    else if (argv[0] === "search") await runSearch(argv.slice(1));
    else if (matches(argv, "table", "source")) await runTableSource(argv.slice(2));
    else if (matches(argv, "table", "dossier")) await runTableDossier(argv.slice(2));
    else if (matches(argv, "table", "sample")) await runTableSample(argv.slice(2));
    else if (matches(argv, "timeseries", "dossier")) await runTimeseriesDossier(argv.slice(2));
    else if (matches(argv, "variables", "explain")) await runVariablesExplain(argv.slice(2));
    else await runLegacy(argv);
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
  console.log(`destatisctl -- Destatis GENESIS-Online statistics CLI

Purpose
  Search and retrieve official German statistics from Destatis GENESIS-Online.

Fast paths
  destatisctl doctor
  destatisctl search --term "Arbeitslose" --limit 5
  destatisctl table source --name 12211-0900
  destatisctl table dossier --name 12211-0900

Legacy endpoint commands
  catalogue statistics|tables|variables
  metadata table|timeseries
  data table|timeseries
  find search

Research commands
  doctor
  search
  table source
  table dossier
  table sample
  timeseries dossier
  variables explain

Auth
  Prefer DESTATIS_USERNAME and DESTATIS_PASSWORD from the environment.
  --username and --password still work and are redacted from output.
  If no credentials are configured, the CLI uses GAST/GAST for public discovery.
`);
}

function printHelp(path: string[]): void {
  const joined = path.join(" ");
  if (joined === "table dossier") {
    console.log(`destatisctl table dossier

Build a cautious evidence bundle for one GENESIS table code. With full
credentials it tries metadata and a small data sample; with guest credentials it
returns source metadata and structured warnings if protected endpoints return 401.
`);
    return;
  }
  if (joined === "search") {
    console.log(`destatisctl search

Friendly alias for the GENESIS find endpoint. Keeps output compact.

Example
  destatisctl search --term "Arbeitslose" --limit 5
`);
    return;
  }
  printRootHelp();
}

async function runDoctor(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const payload = envelope("doctor", "/helloworld/logincheck", undefined, cred);
  payload.summary = {
    baseUrl: BASE_URL,
    webUi: UI_URL,
    docs: DOCS_URL,
    authConfigured: !cred.guest || Boolean(process.env.DESTATIS_USERNAME || process.env.DESTATIS_PASSWORD),
    credentialSource: cred.source,
    guestFallbackEnabled: cred.guest,
    publishedRateLimit: "not found in official Destatis docs reviewed; use small pagelength values and avoid parallel broad requests",
    license: "Datenlizenz Deutschland - Namensnennung - Version 2.0 for GENESIS-Online usage per Destatis Open Data page"
  };
  payload.sources = defaultSources();
  payload.warnings = standardWarnings(cred);
  try {
    const login = await apiPost("/helloworld/logincheck", {}, cred);
    (payload.summary as JsonObject).health = { ok: true, message: login.Status, username: redactUsername(asString(login.Username)) };
  } catch (error) {
    payload.status = "error";
    (payload.summary as JsonObject).health = { ok: false, error: redact(error instanceof Error ? error.message : String(error)) };
  }
  try {
    const found = await apiPost("/find/find", { term: "Arbeitslose", category: "all", pagelength: "1", language: "de" }, cred);
    (payload.summary as JsonObject).findCheck = {
      ok: true,
      status: found.Status,
      tablesFound: asArray(found.Tables).length
    };
  } catch (error) {
    (payload.summary as JsonObject).findCheck = { ok: false, error: redact(error instanceof Error ? error.message : String(error)) };
  }
  payload.nextActions = ['destatisctl search --term "Arbeitslose" --limit 5', "destatisctl table source --name 12211-0900"];
  emit(payload);
}

async function runSearch(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.flags.selection);
  if (!term) throw new CLIError(2, "missing_term", "search requires --term");
  const limit = limitFlag(parsed, 5, 25);
  const params = {
    ...parsed.params,
    term,
    category: firstNonEmpty(parsed.flags.category, parsed.params.category, "all"),
    pagelength: String(limit),
    language: firstNonEmpty(parsed.flags.language, parsed.params.language, "de")
  };
  const data = await apiPost("/find/find", params, cred);
  const items = compactFind(data, limit);
  const payload = envelope("search", "/find/find", params, cred);
  payload.summary = {
    term,
    limitApplied: limit,
    status: data.Status,
    statistics: asArray(data.Statistics).length,
    tables: asArray(data.Tables).length,
    timeseries: asArray(data.Timeseries).length
  };
  payload.items = items;
  payload.sources = defaultSources();
  payload.warnings = standardWarnings(cred);
  payload.nextActions = nextActionsForFind(items);
  if (flagBool(parsed, "include-raw")) payload.raw = data;
  emit(payload);
}

async function runTableSource(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const name = requiredName(parsed);
  const payload = envelope("table source", "/metadata/table", { name }, cred);
  payload.summary = tableSourceSummary(name);
  payload.sources = sourcesForTable(name);
  payload.warnings = ["Source URLs identify official GENESIS locations; table availability and metadata detail can depend on credentials."];
  payload.nextActions = [`destatisctl table dossier --name ${name}`, `destatisctl metadata table --param name=${name}`];
  emit(payload);
}

async function runTableDossier(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const name = requiredName(parsed);
  const payload = envelope("table dossier", "/metadata/table", { name }, cred);
  payload.summary = tableSourceSummary(name);
  payload.sources = sourcesForTable(name);
  payload.warnings = standardWarnings(cred);
  payload.nextActions = [`destatisctl table sample --name ${name}`, `destatisctl variables explain --table ${name}`];
  try {
    const metadata = await apiPost("/metadata/table", { name, language: parsed.flags.language || "de" }, cred);
    payload.metadata = summarizeDestatisPayload(metadata);
    if (flagBool(parsed, "include-raw")) payload.rawMetadata = metadata;
  } catch (error) {
    payload.metadata = { available: false, error: redact(error instanceof Error ? error.message : String(error)) };
    (payload.warnings as string[]).push("Metadata request failed; guest credentials can be insufficient for metadata/data endpoints.");
  }
  if (flagBool(parsed, "sample")) {
    try {
      const sample = await apiPostText("/data/table", { name, area: "all", format: "ffcsv", compress: "true", transpose: "false", language: "de" }, cred);
      payload.sample = { available: true, preview: truncate(sample, 1200) };
    } catch (error) {
      payload.sample = { available: false, error: redact(error instanceof Error ? error.message : String(error)) };
      (payload.warnings as string[]).push("Data sample request failed; use personal GENESIS credentials for protected data endpoints.");
    }
  }
  emit(payload);
}

async function runTableSample(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const name = requiredName(parsed);
  const params = {
    ...parsed.params,
    name,
    area: firstNonEmpty(parsed.flags.area, parsed.params.area, "all"),
    format: firstNonEmpty(parsed.flags.format, parsed.params.format, "ffcsv"),
    compress: firstNonEmpty(parsed.flags.compress, parsed.params.compress, "true"),
    transpose: firstNonEmpty(parsed.flags.transpose, parsed.params.transpose, "false"),
    language: firstNonEmpty(parsed.flags.language, parsed.params.language, "de")
  };
  const payload = envelope("table sample", "/data/table", params, cred);
  payload.summary = tableSourceSummary(name);
  payload.sources = sourcesForTable(name);
  payload.warnings = standardWarnings(cred);
  try {
    const sample = await apiPostText("/data/table", params, cred);
    payload.sample = { available: true, preview: truncate(sample, 1600) };
  } catch (error) {
    payload.status = "partial";
    payload.sample = { available: false, error: redact(error instanceof Error ? error.message : String(error)) };
  }
  emit(payload);
}

async function runTimeseriesDossier(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const name = requiredName(parsed);
  const params = { name, language: parsed.flags.language || "de" };
  const payload = envelope("timeseries dossier", "/metadata/timeseries", params, cred);
  payload.summary = { name, kind: "timeseries", webUi: `${UI_URL}/timeseries/${encodeURIComponent(name)}` };
  payload.sources = defaultSources();
  payload.warnings = standardWarnings(cred);
  try {
    const metadata = await apiPost("/metadata/timeseries", params, cred);
    payload.metadata = summarizeDestatisPayload(metadata);
  } catch (error) {
    payload.status = "partial";
    payload.metadata = { available: false, error: redact(error instanceof Error ? error.message : String(error)) };
  }
  emit(payload);
}

async function runVariablesExplain(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const cred = resolveCredentials(parsed);
  const table = firstNonEmpty(parsed.flags.table, parsed.flags.name, parsed.flags.code);
  if (!table) throw new CLIError(2, "missing_table", "variables explain requires --table");
  const params = { name: table, language: parsed.flags.language || "de" };
  const payload = envelope("variables explain", "/catalogue/tables2variable", params, cred);
  payload.summary = { table, purpose: "discover variables/dimensions connected to a GENESIS table" };
  payload.sources = sourcesForTable(table);
  payload.warnings = standardWarnings(cred);
  try {
    const variables = await apiPost("/catalogue/tables2variable", params, cred);
    payload.variables = summarizeDestatisPayload(variables);
  } catch (error) {
    payload.status = "partial";
    payload.variables = { available: false, error: redact(error instanceof Error ? error.message : String(error)) };
  }
  payload.nextActions = [`destatisctl table dossier --name ${table}`];
  emit(payload);
}

async function runLegacy(argv: string[]): Promise<void> {
  if (argv.length < 2) throw new CLIError(2, "unknown_command", "expected command group and action");
  const command = argv.slice(0, 2).join(" ");
  const path = legacyPaths[command];
  if (!path) throw new CLIError(2, "unknown_command", "unknown command path: " + argv.join(" "));
  const parsed = parseArgs(argv.slice(2));
  const cred = resolveCredentials(parsed);
  const params: Record<string, string> = { ...parsed.params };
  for (const [key, value] of Object.entries(parsed.flags)) {
    if (!["username", "password", "limit", "include-raw", "sample"].includes(key)) params[key] = value;
  }
  if (command === "find search" && !params.term && parsed.flags.selection) params.term = parsed.flags.selection;
  params.language ||= "de";
  params.pagelength ||= String(limitFlag(parsed, 10, 100));
  console.log(await apiPostText(path, params, cred));
}

async function apiPost(path: string, params: Record<string, string>, cred: Credentials): Promise<JsonObject> {
  return JSON.parse(await apiPostText(path, params, cred)) as JsonObject;
}

async function apiPostText(path: string, params: Record<string, string>, cred: Credentials): Promise<string> {
  const form = new URLSearchParams({ ...params, username: cred.username, password: cred.password });
  const response = await fetch(BASE_URL + path, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded", Accept: "application/json,text/plain,*/*" },
    body: form.toString()
  });
  const text = await response.text();
  if (!response.ok) throw new CLIError(1, "http_error", `HTTP ${response.status} from Destatis GENESIS API: ${truncate(text, 280)}`);
  return text;
}

function compactFind(data: JsonObject, limit: number): JsonObject[] {
  const items: JsonObject[] = [];
  const add = (kind: string, key: string): void => {
    for (const value of asArray(data[key])) {
      if (items.length >= limit) return;
      const row = asObject(value);
      const code = asString(row.Code);
      items.push({ kind, code, title: row.Content || "", time: row.Time || "", cubes: row.Cubes || "", sources: sourceLinks(kind, code) });
    }
  };
  add("statistic", "Statistics");
  add("table", "Tables");
  add("timeseries", "Timeseries");
  add("cube", "Cubes");
  return items;
}

function summarizeDestatisPayload(data: JsonObject): JsonObject {
  return {
    status: data.Status,
    ident: data.Ident,
    parameters: redactParamMap(asObject(data.Parameter)),
    objectKeys: Object.keys(data),
    preview: truncate(JSON.stringify(data), 1400)
  };
}

function tableSourceSummary(name: string): JsonObject {
  return {
    name,
    kind: "table",
    apiBaseUrl: BASE_URL,
    webUi: `${UI_URL}/table/${encodeURIComponent(name)}`,
    license: "Datenlizenz Deutschland - Namensnennung - Version 2.0 per Destatis Open Data page"
  };
}

function sourceLinks(kind: string, code: string): JsonObject[] {
  if (kind === "table") return sourcesForTable(code);
  if (kind === "statistic") return [
    { title: "GENESIS statistic page", url: `${UI_URL}/statistic/${encodeURIComponent(code)}`, kind: "web-ui" },
    { title: "GENESIS REST API", url: BASE_URL, kind: "api" }
  ];
  return defaultSources();
}

function sourcesForTable(name: string): JsonObject[] {
  return [
    { title: "GENESIS table page", url: `${UI_URL}/table/${encodeURIComponent(name)}`, kind: "web-ui" },
    { title: "GENESIS metadata endpoint", url: `${BASE_URL}/metadata/table`, kind: "api" },
    { title: "GENESIS data endpoint", url: `${BASE_URL}/data/table`, kind: "api" },
    { title: "Destatis GENESIS API/Webservice page", url: DOCS_URL, kind: "docs" }
  ];
}

function defaultSources(): JsonObject[] {
  return [
    { title: "Destatis GENESIS API/Webservices page", url: DOCS_URL, kind: "docs" },
    { title: "GENESIS-Online database", url: UI_URL, kind: "web-ui" },
    { title: "GENESIS REST base URL", url: BASE_URL, kind: "api" }
  ];
}

function standardWarnings(cred: Credentials): string[] {
  const warnings = [
    "Use small pagelength values for discovery; inspect metadata before requesting data.",
    "Preserve table/statistic codes, units, time periods, and source dates in final answers.",
    "Credentials are redacted from normalized output and errors."
  ];
  if (cred.guest) warnings.push("Using GAST/GAST fallback: discovery works, but metadata/data endpoints may return 401; configure DESTATIS_USERNAME and DESTATIS_PASSWORD for full access.");
  return warnings;
}

function nextActionsForFind(items: JsonObject[]): string[] {
  const actions: string[] = [];
  for (const item of items) {
    const code = asString(item.code);
    if (!code) continue;
    if (item.kind === "table") actions.push(`destatisctl table dossier --name ${code}`);
    else if (item.kind === "timeseries") actions.push(`destatisctl timeseries dossier --name ${code}`);
    else if (item.kind === "statistic") actions.push(`destatisctl catalogue tables --param name=${code}`);
    if (actions.length >= 5) break;
  }
  return actions;
}

function parseArgs(argv: string[]): ParsedArgs {
  const out: ParsedArgs = { flags: {}, params: {}, positionals: [] };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--param" && i + 1 < argv.length) {
      addParam(out.params, argv[++i]);
      continue;
    }
    if (arg.startsWith("--param=")) {
      addParam(out.params, arg.slice("--param=".length));
      continue;
    }
    if (arg.startsWith("--")) {
      const name = arg.slice(2);
      if (name.includes("=")) {
        const [key, value] = splitKeyValue(name);
        out.flags[key] = value;
      } else if (i + 1 < argv.length && !argv[i + 1].startsWith("--")) {
        out.flags[name] = argv[++i];
      } else {
        out.flags[name] = "true";
      }
      continue;
    }
    out.positionals.push(arg);
  }
  return out;
}

function addParam(params: Record<string, string>, raw: string): void {
  const [key, value] = splitKeyValue(raw);
  if (key) params[key] = value;
}

function splitKeyValue(raw: string): [string, string] {
  const idx = raw.indexOf("=");
  return idx < 0 ? ["", ""] : [raw.slice(0, idx), raw.slice(idx + 1)];
}

function resolveCredentials(parsed: ParsedArgs): Credentials {
  const username = firstNonEmpty(parsed.flags.username, process.env.DESTATIS_USERNAME, "GAST");
  const password = firstNonEmpty(parsed.flags.password, process.env.DESTATIS_PASSWORD, "GAST");
  let source = "guest:GAST";
  if (parsed.flags.username || parsed.flags.password) source = "flags:redacted";
  else if (process.env.DESTATIS_USERNAME || process.env.DESTATIS_PASSWORD) source = "env:DESTATIS_USERNAME/DESTATIS_PASSWORD";
  return { username, password, source, guest: username === "GAST" && password === "GAST" };
}

function envelope(command: string, path: string, params: Record<string, string> | undefined, cred: Credentials): JsonObject {
  const request: JsonObject = { method: "POST", url: BASE_URL + path, credentialSource: cred.source, redactedFields: ["username", "password"] };
  if (params) request.params = redactParamMap(params);
  return { status: "ok", tool: APP_NAME, command, retrievedAt: new Date().toISOString(), request };
}

function requiredName(parsed: ParsedArgs): string {
  const name = firstNonEmpty(parsed.flags.name, parsed.flags.code, parsed.flags.table, parsed.positionals[0]);
  if (!name) throw new CLIError(2, "missing_name", "requires --name, --code, or --table");
  return name;
}

function redactParamMap(params: JsonObject): JsonObject {
  const out: JsonObject = {};
  for (const [key, value] of Object.entries(params)) out[key] = isSecretKey(key) ? "REDACTED" : value;
  return out;
}

function isSecretKey(key: string): boolean {
  const lower = key.toLowerCase();
  return ["username", "password", "passwort"].includes(lower) || lower.includes("token");
}

function emit(payload: unknown): void {
  console.log(JSON.stringify(payload, null, 2));
}

function fail(exitCode: number, code: string, message: string): void {
  emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message: redact(message) } });
  process.exit(exitCode);
}

function limitFlag(parsed: ParsedArgs, fallback: number, maximum: number): number {
  const raw = parsed.flags.limit || parsed.params.pagelength || String(fallback);
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) && n > 0 ? Math.min(n, maximum) : fallback;
}

function flagBool(parsed: ParsedArgs, name: string): boolean {
  return ["1", "true", "yes"].includes((parsed.flags[name] || "").toLowerCase());
}

function firstNonEmpty(...values: Array<string | undefined>): string {
  for (const value of values) if (value && value.trim()) return value;
  return "";
}

function truncate(text: string, limit: number): string {
  const collapsed = String(text).split(/\s+/).filter(Boolean).join(" ");
  return collapsed.length <= limit ? collapsed : collapsed.slice(0, limit - 3) + "...";
}

function redactUsername(username: string): string {
  return username === "" || username === "GAST" ? username : "REDACTED";
}

function redact(text: string): string {
  return String(text)
    .replace(/(username|password|passwort|token)=([^&\s]+)/gi, "$1=REDACTED")
    .replace(/(--(?:username|password|token)\s+)([^\s]+)/gi, "$1REDACTED");
}

function asObject(value: unknown): JsonObject {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonObject : {};
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function asString(value: unknown): string {
  return value === undefined || value === null ? "" : String(value);
}

function isHelp(arg: string): boolean {
  return arg === "-h" || arg === "--help" || arg === "help";
}

function matches(argv: string[], ...parts: string[]): boolean {
  return parts.every((part, index) => argv[index] === part);
}

main(process.argv.slice(2)).then((code) => {
  if (code !== 0) process.exit(code);
});
