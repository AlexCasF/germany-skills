#!/usr/bin/env node

type JsonObject = Record<string, unknown>;
type ParsedArgs = { flags: Record<string, string>; params: Record<string, string>; positionals: string[] };

const APP_NAME = "bundeshaushalt";
const BASE_URL = "https://bundeshaushalt.de";
const BUDGET_DATA_URL = `${BASE_URL}/internalapi/budgetData`;
const DIGITAL_URL = "https://www.bundeshaushalt.de/DE/Bundeshaushalt-digital/bundeshaushalt-digital.html";
const USER_NOTES_URL = "https://www.bundeshaushalt.de/DE/Service/Benutzerhinweise/benutzerhinweise.html";
const ROBOTS_URL = "https://www.bundeshaushalt.de/robots.txt";
const BMF_BUDGET_URL = "https://www.bundesfinanzministerium.de/Web/DE/Themen/Oeffentliche_Finanzen/Bundeshaushalt/bundeshaushalt.html";
const BMF_DATA_USE_URL = "https://www.bundesfinanzministerium.de/Datenportal/Nutzungshinweise/nutzungshinweise.html";
const OPENAPI_WRAPPER_URL = "https://github.com/bundesAPI/bundeshaushalt-api";
const USER_AGENT = "germany-skills/bundeshaushalt-node";
const KNOWN_YEARS = Array.from({ length: 15 }, (_, index) => 2012 + index);
const EARLIEST_KNOWN_YEAR = 2012;
const LATEST_TARGET_YEAR = 2026;
const LATEST_ACTUAL_YEAR = 2024;
const DEFAULT_LIMIT = 10;
const SAFE_LIMIT = 100;
const DEFAULT_SEARCH_DEPTH = 3;

class CliError extends Error {
  code: string;
  exitCode: number;

  constructor(code: string, message: string, exitCode = 1) {
    super(message);
    this.code = code;
    this.exitCode = exitCode;
  }
}

main(process.argv.slice(2)).then(
  (exitCode) => {
    process.exitCode = exitCode;
  },
  (error: unknown) => {
    emitError("unexpected_error", error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  },
);

async function main(argv: string[]): Promise<number> {
  try {
    if (argv.length === 0 || isHelp(argv[0])) {
      printRootHelp();
      return 0;
    }
    if (isHelp(argv[argv.length - 1])) {
      printHelp(argv.slice(0, -1));
      return 0;
    }
    if (argv[0] === "doctor") await runDoctor(argv.slice(1));
    else if (argv[0] === "examples") printExamples();
    else if (argv[0] === "fields") runFields(argv.slice(1));
    else if (argv[0] === "source") runSource(argv.slice(1));
    else if (match(argv, "years", "list")) runYearsList(argv.slice(2));
    else if (match(argv, "budget", "tree")) await runBudgetTree(argv.slice(2));
    else if (match(argv, "budget", "sample")) await runSample(argv.slice(2));
    else if (argv[0] === "sample") await runSample(argv.slice(1));
    else if (match(argv, "title", "get")) await runTitleGet(argv.slice(2));
    else if (argv[0] === "search") await runSearch(argv.slice(1));
    else if (argv[0] === "compare") await runCompare(argv.slice(1));
    else if (argv[0] === "budget-data") await runBudgetData(argv.slice(1));
    else throw new CliError("unknown_command", `unknown command: ${argv.join(" ")}`, 2);
    return 0;
  } catch (error) {
    if (error instanceof CliError) {
      emitError(error.code, error.message);
      return error.exitCode;
    }
    emitError("unexpected_error", error instanceof Error ? error.message : String(error));
    return 1;
  }
}

function printRootHelp(): void {
  console.log(`bundeshaushalt - Bundeshaushalt Digital research CLI

Usage:
  bundeshaushalt doctor
  bundeshaushalt years list
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt search --year 2025 --account expenses --term "Suchbegriff" --limit 5
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

JSON is the default output. Use --raw on endpoint-style commands to emit upstream JSON.`);
}

function printHelp(args: string[]): void {
  if (args.length === 0) printRootHelp();
  else if (args[0] === "budget-data") console.log("budget-data flags: --year --account expenses|income --quota target|actual --unit single|function|group --id --param key=value --raw");
  else if (args[0] === "search") console.log("search flags: --year --account --quota --unit --term/--q --depth --max-requests --limit --include-raw");
  else if (match(args, "budget", "tree")) console.log("budget tree flags: --year --account --quota --unit --id --limit --grep --include-raw --raw");
  else if (args[0] === "compare") console.log("compare flags: --years 2024,2025 --account --quota --unit --id");
  else printRootHelp();
}

function printExamples(): void {
  console.log(`Examples:
  bundeshaushalt doctor
  bundeshaushalt source
  bundeshaushalt years list
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt search --year 2025 --account expenses --term "Arbeit" --limit 5
  bundeshaushalt title get --year 2025 --account expenses --id 110168112
  bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112`);
}

async function runDoctor(_argv: string[]): Promise<void> {
  const checks: JsonObject[] = [];
  for (const [name, params] of [
    ["latestTargetExpenses", { year: String(LATEST_TARGET_YEAR), account: "expenses", quota: "target", unit: "single" }],
    ["latestActualExpenses", { year: String(LATEST_ACTUAL_YEAR), account: "expenses", quota: "actual", unit: "single" }],
  ] as Array<[string, Record<string, string>]>) {
    const url = withParams(BUDGET_DATA_URL, params);
    try {
      const response = await fetchRaw(url);
      const data = response.status < 300 ? JSON.parse(response.body) : {};
      checks.push({ name, ok: response.status < 300, url, bodyBytes: response.body.length, meta: data.meta });
    } catch (error) {
      checks.push({ name, ok: false, url, error: error instanceof Error ? error.message : String(error) });
    }
  }
  const payload = envelope("doctor", BUDGET_DATA_URL, {});
  payload.summary = {
    authRequired: false,
    publishedRateLimit: "No exact public request quota was found. robots.txt publishes Crawl-delay: 30 for crawling-style workflows.",
    endpointBehavior: "GET /internalapi/budgetData requires at least year and account; some actual values return 404 until accounting data exists.",
    checks,
  };
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = [
    "bundeshaushalt years list",
    `bundeshaushalt budget tree --year ${LATEST_TARGET_YEAR} --account expenses --quota target --limit 8`,
    'bundeshaushalt search --year 2025 --account expenses --term "Arbeit" --limit 5',
  ];
  emit(payload);
}

function runFields(_argv: string[]): void {
  const payload = envelope("fields", BUDGET_DATA_URL, {});
  payload.summary = {
    accounts: [{ value: "expenses", meaning: "Ausgaben" }, { value: "income", meaning: "Einnahmen" }],
    quotas: [{ value: "target", meaning: "Soll/plan" }, { value: "actual", meaning: "Ist/accounting data" }],
    units: [
      { value: "single", meaning: "Einzelplan/ministry and title hierarchy" },
      { value: "function", meaning: "Functional classification" },
      { value: "group", meaning: "Revenue/expenditure group classification" },
    ],
    coreFields: ["id", "budgetNumber", "label", "value", "relativeToParentValue", "relativeValue"],
    valueUnits: "Nominal euro amounts; helper fields expose valueEur and valueBillionEur.",
  };
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = [`bundeshaushalt budget tree --year ${LATEST_TARGET_YEAR} --account expenses --limit 8`];
  emit(payload);
}

function runSource(_argv: string[]): void {
  const payload = envelope("source", BUDGET_DATA_URL, {});
  payload.summary = {
    publisher: "Bundesministerium der Finanzen / Bundeshaushalt Digital",
    authRequired: false,
    rateLimit: "No exact public quota found; robots.txt contains Crawl-delay: 30.",
    knownEndpoint: BUDGET_DATA_URL,
    knownYears: { earliest: EARLIEST_KNOWN_YEAR, latestTarget: LATEST_TARGET_YEAR, latestActual: LATEST_ACTUAL_YEAR },
  };
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = ["bundeshaushalt fields", "bundeshaushalt years list"];
  emit(payload);
}

function runYearsList(_argv: string[]): void {
  const items = KNOWN_YEARS.map((year) => ({
    year,
    targetLikely: true,
    actualLikely: year <= LATEST_ACTUAL_YEAR,
    exampleTargetCmd: `bundeshaushalt budget tree --year ${year} --account expenses --quota target --limit 8`,
  }));
  const payload = envelope("years list", BUDGET_DATA_URL, {});
  payload.summary = {
    count: items.length,
    earliestKnownYear: EARLIEST_KNOWN_YEAR,
    latestTargetYear: LATEST_TARGET_YEAR,
    latestActualYear: LATEST_ACTUAL_YEAR,
    note: "Known from live endpoint probes; the bundled OpenAPI enum stops at 2021.",
  };
  payload.items = items;
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = [`bundeshaushalt budget tree --year ${LATEST_TARGET_YEAR} --account expenses --quota target --limit 8`];
  emit(payload);
}

async function runBudgetTree(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const params = budgetParams(parsed, true);
  const { raw, data, requestUrl } = await fetchBudget(params);
  if (flagBool(parsed, "raw")) {
    process.stdout.write(raw);
    return;
  }
  emitBudgetEnvelope("budget tree", requestUrl, params, data, parsed);
}

async function runSample(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  parsed.flags.year ??= String(LATEST_TARGET_YEAR);
  parsed.flags.limit ??= "5";
  const params = budgetParams(parsed, true);
  const { data, requestUrl } = await fetchBudget(params);
  emitBudgetEnvelope("budget sample", requestUrl, params, data, parsed);
}

async function runTitleGet(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  if (!first(parsed.flags.id, parsed.params.id)) throw new CliError("missing_id", "title get requires --id", 2);
  const params = budgetParams(parsed, true);
  const { raw, data, requestUrl } = await fetchBudget(params);
  if (flagBool(parsed, "raw")) {
    process.stdout.write(raw);
    return;
  }
  emitBudgetEnvelope("title get", requestUrl, params, data, parsed);
}

async function runSearch(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const term = first(parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
  if (!term) throw new CliError("missing_term", "search requires --term", 2);
  const params = budgetParams(parsed, true);
  delete params.id;
  const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
  const depth = intFlag(parsed, "depth", DEFAULT_SEARCH_DEPTH, 6);
  const maxRequests = intFlag(parsed, "max-requests", 60, 250);
  const { items, requests } = await searchHierarchy(params, term, depth, maxRequests, limit, flagBool(parsed, "include-raw"));
  const payload = envelope("search", BUDGET_DATA_URL, { year: params.year, account: params.account, quota: params.quota, unit: params.unit, term, depth, maxRequests, limit });
  payload.summary = { term, returned: items.length, requestsUsed: requests, requestCap: maxRequests, traversalDepth: depth };
  payload.items = items;
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = nextActionsFromSearch(items, params);
  emit(payload);
}

async function runCompare(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const rawYears = first(parsed.flags.years, parsed.params.years, parsed.flags.year, parsed.params.year);
  if (!rawYears) throw new CliError("missing_years", "compare requires --years 2024,2025", 2);
  const years = rawYears.split(/[;,]/).map((part) => Number(part.trim())).filter((year) => Number.isFinite(year));
  if (years.length < 2) throw new CliError("missing_years", "compare needs at least two years", 2);
  const baseParams = budgetParams(parsed, false);
  delete baseParams.year;
  const items: JsonObject[] = [];
  for (const year of years) {
    const params = { ...baseParams, year: String(year) };
    try {
      const { data, requestUrl } = await fetchBudget(params);
      items.push({ year, ok: true, requestUrl, meta: data.meta, detail: compactElement((data.detail as JsonObject) ?? {}, (data.meta as JsonObject) ?? {}, params), childCount: Array.isArray(data.children) ? data.children.length : 0 });
    } catch (error) {
      items.push({ year, ok: false, error: error instanceof Error ? error.message : String(error), requestUrl: withParams(BUDGET_DATA_URL, params) });
    }
  }
  const payload = envelope("compare", BUDGET_DATA_URL, { years, account: baseParams.account, quota: baseParams.quota, unit: baseParams.unit, id: baseParams.id });
  payload.summary = { years, returned: items.length, account: baseParams.account, quota: baseParams.quota, unit: baseParams.unit, id: baseParams.id };
  payload.items = items;
  payload.status = items.every((item) => item.ok === true) ? "ok" : "partial";
  payload.sources = defaultSources();
  payload.warnings = defaultWarnings();
  payload.nextActions = ["bundeshaushalt source"];
  emit(payload);
}

async function runBudgetData(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const params = budgetParams(parsed, false, false);
  if (!params.year || !params.account) throw new CliError("missing_required_params", "budget-data requires --year and --account (or --param year=... --param account=...)", 2);
  const { raw, data, requestUrl } = await fetchBudget(params);
  if (flagBool(parsed, "raw")) {
    process.stdout.write(raw);
    return;
  }
  emitBudgetEnvelope("budget-data", requestUrl, params, data, parsed);
}

function emitBudgetEnvelope(command: string, requestUrl: string, params: Record<string, string>, data: JsonObject, parsed: ParsedArgs): void {
  const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
  const grep = (parsed.flags.grep ?? "").toLowerCase();
  const children = Array.isArray(data.children) ? (data.children as JsonObject[]) : [];
  let items = children.map((child) => compactElement(child, (data.meta as JsonObject) ?? {}, params));
  if (grep) items = items.filter((item) => String(item.label ?? "").toLowerCase().includes(grep));
  if (limit >= 0) items = items.slice(0, limit);
  const payload = envelope(command, requestUrl, { ...params, limit, grep });
  payload.summary = {
    meta: data.meta,
    detail: compactElement((data.detail as JsonObject) ?? {}, (data.meta as JsonObject) ?? {}, params),
    childrenTotal: children.length,
    childrenShown: items.length,
    parentsLevels: Array.isArray(data.parents) ? data.parents.length : 0,
    relatedKeys: data.related && typeof data.related === "object" ? Object.keys(data.related as JsonObject).sort() : null,
  };
  payload.items = items;
  payload.sources = [{ kind: "api_request", title: "Bundeshaushalt API request", url: requestUrl }, ...defaultSources()];
  payload.warnings = defaultWarningsForResponse(data);
  payload.nextActions = nextActionsFromChildren(items);
  emit(payload);
}

async function searchHierarchy(params: Record<string, string>, term: string, depthLimit: number, maxRequests: number, limit: number, includeRaw: boolean): Promise<{ items: JsonObject[]; requests: number }> {
  const needle = term.toLowerCase();
  const queue: Array<{ id: string; depth: number }> = [{ id: "", depth: 0 }];
  const seen = new Set<string>();
  const items: JsonObject[] = [];
  let requests = 0;
  while (queue.length > 0 && requests < maxRequests && items.length < limit) {
    const node = queue.shift();
    if (!node || seen.has(node.id) || node.depth > depthLimit) continue;
    seen.add(node.id);
    const requestParams = { ...params };
    if (node.id) requestParams.id = node.id;
    let data: JsonObject;
    let requestUrl: string;
    try {
      const fetched = await fetchBudget(requestParams);
      data = fetched.data;
      requestUrl = fetched.requestUrl;
      requests += 1;
    } catch {
      requests += 1;
      continue;
    }
    const detail = ((data.detail as JsonObject) ?? {});
    if (matchesElement(detail, needle)) {
      const item = compactElement(detail, (data.meta as JsonObject) ?? {}, requestParams);
      item.matchType = "detail";
      item.requestUrl = requestUrl;
      if (includeRaw) item.raw = data;
      items.push(item);
      if (items.length >= limit) break;
    }
    for (const child of Array.isArray(data.children) ? (data.children as JsonObject[]) : []) {
      const childId = String(child.id ?? "");
      if (matchesElement(child, needle)) {
        const item = compactElement(child, (data.meta as JsonObject) ?? {}, params);
        item.matchType = "child";
        item.parentId = node.id;
        item.parentLabel = detail.label;
        if (includeRaw) item.raw = child;
        items.push(item);
        if (items.length >= limit) break;
      }
      if (childId && node.depth + 1 <= depthLimit) queue.push({ id: childId, depth: node.depth + 1 });
    }
  }
  return { items, requests };
}

function compactElement(element: JsonObject, meta: JsonObject, params: Record<string, string>): JsonObject {
  const value = Number(element.value ?? 0);
  const item: JsonObject = {
    id: String(element.id ?? ""),
    budgetNumber: String(element.budgetNumber ?? ""),
    label: String(element.label ?? ""),
    value,
    valueEur: value,
    valueBillionEur: value / 1_000_000_000,
    relativeToParentValue: element.relativeToParentValue,
    relativeValue: element.relativeValue,
    tableLabel: String(element.tableLabel ?? ""),
    selectionLabel: String(element.selectionLabel ?? ""),
    year: meta.year,
    account: String(meta.account ?? params.account ?? ""),
    quota: String(meta.quota ?? params.quota ?? ""),
    unit: String(meta.unit ?? params.unit ?? ""),
    entity: meta.entity,
    levelCur: meta.levelCur,
    levelMax: meta.levelMax,
  };
  if (item.id) {
    item.nextActions = [
      `bundeshaushalt title get --year ${item.year} --account ${item.account} --quota ${item.quota} --unit ${item.unit} --id ${item.id}`,
      `bundeshaushalt budget tree --year ${item.year} --account ${item.account} --quota ${item.quota} --unit ${item.unit} --id ${item.id} --limit 10`,
    ];
  }
  return item;
}

async function fetchBudget(params: Record<string, string>): Promise<{ raw: string; data: JsonObject; requestUrl: string }> {
  const requestUrl = withParams(BUDGET_DATA_URL, params);
  const response = await fetchRaw(requestUrl);
  if (response.status < 200 || response.status >= 300) throw new CliError("upstream_http_error", `upstream status ${response.status} from ${requestUrl}: ${stripSpace(response.body).slice(0, 300)}`);
  return { raw: response.body, data: JSON.parse(response.body) as JsonObject, requestUrl };
}

async function fetchRaw(url: string): Promise<{ status: number; body: string }> {
  let lastStatus = 0;
  let lastBody = "";
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    if (attempt > 0) await sleep(attempt * 750);
    try {
      const response = await fetch(url, { headers: { "User-Agent": USER_AGENT, Accept: "application/json" } });
      const body = await response.text();
      if (![429, 502, 503, 504].includes(response.status)) return { status: response.status, body };
      lastStatus = response.status;
      lastBody = body;
    } catch (error) {
      lastError = error;
    }
  }
  if (lastStatus) return { status: lastStatus, body: lastBody };
  throw lastError instanceof Error ? lastError : new Error(String(lastError));
}

function budgetParams(parsed: ParsedArgs, requireYear: boolean, requireAccount = false): Record<string, string> {
  const params: Record<string, string> = { ...parsed.params };
  for (const key of ["year", "account", "quota", "unit", "id"]) {
    if (parsed.flags[key]) params[key] = parsed.flags[key];
  }
  params.quota ??= "target";
  params.unit ??= "single";
  if (requireYear && !params.year) throw new CliError("missing_year", "command requires --year", 2);
  if (requireAccount && !params.account) throw new CliError("missing_account", "command requires --account", 2);
  params.account ??= "expenses";
  validateBudgetParams(params);
  return Object.fromEntries(Object.entries(params).filter(([, value]) => String(value).trim())) as Record<string, string>;
}

function validateBudgetParams(params: Record<string, string>): void {
  if (params.year) {
    const year = Number(params.year);
    if (!Number.isInteger(year) || year < 2000 || year > 2100) throw new CliError("invalid_year", "year must be a plausible four-digit year", 2);
  }
  if (params.account && !["expenses", "income"].includes(params.account)) throw new CliError("invalid_account", "account must be expenses or income", 2);
  if (params.quota && !["target", "actual"].includes(params.quota)) throw new CliError("invalid_quota", "quota must be target or actual", 2);
  if (params.unit && !["single", "function", "group"].includes(params.unit)) throw new CliError("invalid_unit", "unit must be single, function, or group", 2);
}

function parseArgs(argv: string[]): ParsedArgs {
  const flags: Record<string, string> = {};
  const params: Record<string, string> = {};
  const positionals: string[] = [];
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token.startsWith("--")) {
      const key = token.slice(2);
      if (key === "param") {
        const value = argv[++index] ?? "";
        const equals = value.indexOf("=");
        if (equals < 1) throw new CliError("invalid_param", "--param requires key=value", 2);
        params[value.slice(0, equals)] = value.slice(equals + 1);
      } else if (argv[index + 1] && !argv[index + 1].startsWith("--")) {
        flags[key] = argv[++index];
      } else {
        flags[key] = "true";
      }
    } else {
      positionals.push(token);
    }
  }
  return { flags, params, positionals };
}

function envelope(command: string, requestUrl: string, request: unknown): JsonObject {
  return { status: "ok", tool: APP_NAME, command, retrievedAt: new Date().toISOString(), request: { method: "GET", url: requestUrl, params: request ?? {} }, summary: {}, items: [], sources: [], warnings: [], nextActions: [] };
}

function emit(payload: JsonObject): void {
  console.log(JSON.stringify(payload, null, 2));
}

function emitError(code: string, message: string): void {
  emit({ status: "error", tool: APP_NAME, retrievedAt: new Date().toISOString(), error: { code, message } });
}

function defaultSources(): JsonObject[] {
  return [
    { kind: "official_application", title: "Bundeshaushalt Digital", url: DIGITAL_URL },
    { kind: "api_endpoint", title: "Bundeshaushalt internal API endpoint", url: BUDGET_DATA_URL },
    { kind: "official_context", title: "BMF Bundeshaushalt overview", url: BMF_BUDGET_URL },
    { kind: "terms", title: "BMF Datenportal usage notes", url: BMF_DATA_USE_URL },
    { kind: "terms", title: "Bundeshaushalt user notes", url: USER_NOTES_URL },
    { kind: "fair_use", title: "Bundeshaushalt robots.txt", url: ROBOTS_URL },
    { kind: "openapi_reference", title: "OpenAPI wrapper", url: OPENAPI_WRAPPER_URL },
  ];
}

function defaultWarnings(): string[] {
  return [
    "No exact public rate limit for the Bundeshaushalt Digital API was found; robots.txt publishes Crawl-delay: 30 for crawling-like workflows.",
    "Actual/Ist values are only available after accounting data exists; newer years can return 404 for quota=actual.",
    "The bundled OpenAPI enum stops at 2021; live endpoint checks show newer target years are available.",
    "Budget values are nominal euro amounts; use statistical APIs for inflation, population, or macroeconomic context.",
    "Use BMF attribution and preserve dataset/page URLs in final citations.",
  ];
}

function defaultWarningsForResponse(data: JsonObject): string[] {
  const warnings = defaultWarnings();
  const meta = ((data.meta as JsonObject) ?? {});
  if (meta.quota === "actual" && Number(meta.year ?? 0) > LATEST_ACTUAL_YEAR) warnings.push("This actual/Ist year is newer than the latest actual year observed during testing; verify availability carefully.");
  if (meta.unit === "function" || meta.unit === "group") warnings.push("Function and group views classify titles differently from Einzelplan ministry structure; do not mix categories without saying so.");
  return warnings;
}

function nextActionsFromChildren(items: JsonObject[]): string[] {
  const actions = ["bundeshaushalt source"];
  for (const item of items.slice(0, 3)) {
    if (item.id) actions.push(`bundeshaushalt budget tree --year ${item.year} --account ${item.account} --quota ${item.quota} --unit ${item.unit} --id ${item.id} --limit 10`);
  }
  return actions;
}

function nextActionsFromSearch(items: JsonObject[], params: Record<string, string>): string[] {
  const actions = items.slice(0, 3).filter((item) => item.id).map((item) => `bundeshaushalt title get --year ${item.year} --account ${item.account} --quota ${item.quota} --unit ${item.unit} --id ${item.id}`);
  if (actions.length === 0) actions.push(`bundeshaushalt budget tree --year ${params.year} --account ${params.account} --limit 8`);
  return actions;
}

function withParams(base: string, params: Record<string, string>): string {
  const url = new URL(base);
  for (const [key, value] of Object.entries(params)) {
    if (String(value).trim()) url.searchParams.set(key, value);
  }
  return url.toString();
}

function flagBool(parsed: ParsedArgs, key: string): boolean {
  return ["1", "true", "yes", "on"].includes((parsed.flags[key] ?? "").toLowerCase());
}

function intFlag(parsed: ParsedArgs, key: string, fallback: number, max: number): number {
  if (!parsed.flags[key]) return fallback;
  const number = Number(parsed.flags[key]);
  if (!Number.isInteger(number)) throw new CliError("invalid_integer", `--${key} must be an integer`, 2);
  return Math.max(0, Math.min(number, max));
}

function limitFlag(parsed: ParsedArgs, fallback: number, max: number): number {
  return intFlag(parsed, "limit", fallback, max);
}

function matchesElement(element: JsonObject, needle: string): boolean {
  return `${String(element.id ?? "")} ${String(element.budgetNumber ?? "")} ${String(element.label ?? "")}`.toLowerCase().includes(needle);
}

function first(...values: Array<string | undefined>): string {
  for (const value of values) {
    if (value && value.trim()) return value.trim();
  }
  return "";
}

function stripSpace(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isHelp(value: string): boolean {
  return ["--help", "-h", "help"].includes(value);
}

function match(args: string[], ...expected: string[]): boolean {
  return args.length >= expected.length && expected.every((value, index) => args[index] === value);
}
