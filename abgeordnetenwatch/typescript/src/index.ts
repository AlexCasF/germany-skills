import * as https from "node:https";
import { URL, URLSearchParams } from "node:url";

const APP_NAME = "abgeordnetenwatch";
const BASE_URL = "https://www.abgeordnetenwatch.de/api/v2";
const ROOT_URL = "https://www.abgeordnetenwatch.de";

type ParsedArgs = {
  flags: Record<string, string>;
  params: Record<string, string>;
  positionals: string[];
};

type ApiResponse = {
  url: string;
  status: number;
  contentType: string;
  body: string;
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

const rawEntities: Record<string, string> = {
  "parliaments": "Parliaments",
  "parliament-periods": "Parliament periods, legislatures, and elections",
  "politicians": "Politicians and candidate/person profile data",
  "candidacies-mandates": "Candidacies and mandates",
  "polls": "Named votes / poll metadata",
  "sidejobs": "Side jobs and disclosed outside income",
  "sidejob-organizations": "Organizations connected to side jobs",
  "votes": "Individual vote records",
  "parties": "Parties",
  "committees": "Committees",
  "committee-memberships": "Committee memberships",
  "fractions": "Parliamentary groups/fractions",
  "electoral-lists": "Electoral lists",
  "constituencies": "Constituencies",
  "election-programs": "Election programs",
  "topics": "Topics",
  "cities": "Cities used in side-job data",
  "countries": "Countries used in side-job data"
};

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
    if (argv[0] === "doctor") {
      await runDoctor();
    } else if (matches(argv, "politicians", "search")) {
      await runPoliticianSearch(argv.slice(2));
    } else if (matches(argv, "politicians", "page")) {
      await runPoliticianPage(argv.slice(2));
    } else if (matches(argv, "politicians", "source")) {
      await runPoliticianSource(argv.slice(2));
    } else if (matches(argv, "politicians", "dossier")) {
      await runPoliticianDossier(argv.slice(2));
    } else if (matches(argv, "mandates", "for-politician")) {
      await runMandatesForPolitician(argv.slice(2));
    } else if (matches(argv, "sidejobs", "for-politician")) {
      await runSidejobsForPolitician(argv.slice(2));
    } else if (argv[0] === "page") {
      await runPoliticianPage(argv.slice(1));
    } else if (argv[0] === "source") {
      await runPoliticianSource(argv.slice(1));
    } else {
      await runRaw(argv);
    }
  } catch (error) {
    if (error instanceof CLIError) {
      fail(error.exitCode, error.code, error.message);
      return error.exitCode;
    }
    const message = error instanceof Error ? error.message : String(error);
    fail(1, "unexpected_error", message);
    return 1;
  }
  return 0;
}

function printRootHelp(): void {
  console.log(`abgeordnetenwatch -- abgeordnetenwatch.de public transparency data

Purpose
  Search and cite public politician, mandate, voting, profile, and side-job
  data from abgeordnetenwatch.de.

Fast paths
  abgeordnetenwatch doctor
  abgeordnetenwatch politicians search --name "Mustername" --limit 3
  abgeordnetenwatch politicians dossier --name "Mustername" --grep Suchbegriff

Raw endpoint commands
  <entity> list|get

Research commands
  doctor
  politicians search
  politicians page
  politicians source
  politicians dossier
  mandates for-politician
  sidejobs for-politician
`);
}

function printHelp(path: string[]): void {
  const joined = path.join(" ");
  if (joined === "politicians dossier") {
    console.log(`abgeordnetenwatch politicians dossier

Builds a compact evidence bundle for one politician with API profile data,
mandates, side jobs, source URLs, page metadata, optional profile-page snippets,
warnings, and next actions.

Examples
  abgeordnetenwatch politicians dossier --name "Mustername" --grep Suchbegriff
  abgeordnetenwatch politicians dossier --id <politician-id> --limit 5
`);
    return;
  }
  if (joined === "politicians page") {
    console.log(`abgeordnetenwatch politicians page

Fetches a public profile page and extracts canonical URL, title, description,
profile ID hints, text preview, and grep snippets.
`);
    return;
  }
  if (joined === "politicians search") {
    console.log(`abgeordnetenwatch politicians search

Searches politicians by name with a small default limit and normalized source URLs.
`);
    return;
  }
  printRootHelp();
}

async function runDoctor(): Promise<void> {
  const data = await apiJson("/politicians", { range_end: "1" });
  const meta = data.meta || {};
  const apiInfo = meta.abgeordnetenwatch_api || {};
  const result = meta.result || {};
  const payload = envelope("doctor", `${BASE_URL}/politicians?range_end=1`);
  payload.summary = {
    authRequired: false,
    baseUrl: BASE_URL,
    apiVersion: apiInfo.version,
    licence: apiInfo.licence,
    licenceLink: apiInfo.licence_link,
    documentation: [
      "https://www.abgeordnetenwatch.de/api",
      "https://www.abgeordnetenwatch.de/api/response",
      "https://www.abgeordnetenwatch.de/api/version-changelog/aktuell"
    ],
    publishedRateLimit: "not found in official API docs",
    resultLimit: "default 100; range_end/pager_limit up to 1000 per official docs",
    health: {
      status: meta.status,
      count: result.count,
      sampleTotal: result.total
    }
  };
  payload.sources = defaultSources();
  payload.warnings = standardWarnings();
  payload.nextActions = [
    "abgeordnetenwatch politicians search --name \"Mustername\" --limit 3",
    "abgeordnetenwatch politicians dossier --id <politician-id> --grep Suchbegriff"
  ];
  emit(payload);
}

async function runRaw(argv: string[]): Promise<void> {
  if (argv.length < 2) {
    throw new CLIError(2, "unknown_command", "expected <entity> list|get");
  }
  const entity = argv[0];
  const action = argv[1];
  if (!rawEntities[entity]) {
    throw new CLIError(2, "unknown_entity", "unknown entity: " + entity);
  }
  const parsed = parseArgs(argv.slice(2));
  const params = normalizeParams(parsed);
  if (action === "list") {
    const resp = await apiGet("/" + entity, params);
    console.log(resp.body);
    return;
  }
  if (action === "get") {
    const id = parsed.flags.id || parsed.positionals[0];
    if (!id) {
      throw new CLIError(2, "missing_id", entity + " get requires --id");
    }
    const resp = await apiGet("/" + entity + "/" + encodeURIComponent(id), params);
    console.log(resp.body);
    return;
  }
  throw new CLIError(2, "unknown_action", `unknown action for ${entity}: ${action}`);
}

async function runPoliticianSearch(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const params = normalizeParams(parsed);
  const limit = limitFlag(parsed, 5, 50);
  params.range_end = String(limit);
  if (parsed.flags.name) {
    params["label[cn]"] = parsed.flags.name;
  }
  if (parsed.flags["first-name"]) {
    params["first_name[cn]"] = parsed.flags["first-name"];
  }
  if (parsed.flags["last-name"]) {
    params["last_name[cn]"] = parsed.flags["last-name"];
  }
  if (parsed.flags.party) {
    params["party[entity.label][cn]"] = parsed.flags.party;
  }
  const data = await apiJson("/politicians", params);
  const items = summarizeRecords(dataList(data), limit);
  const payload = envelope("politicians search", BASE_URL + "/politicians?" + new URLSearchParams(params).toString());
  payload.summary = { search: searchSummary(parsed), returned: items.length, total: total(data), clientLimit: limit };
  payload.items = items;
  payload.sources = [{ kind: "api", title: "Politicians endpoint", url: BASE_URL + "/politicians" }];
  payload.warnings = ["Search results are public transparency data; verify official parliamentary records separately when needed."];
  payload.nextActions = nextForPoliticianItems(items);
  emit(payload);
}

async function runPoliticianSource(argv: string[]): Promise<void> {
  const resolved = await resolvePolitician(argv);
  const payload = envelope("politicians source", apiUrlFromRecord(resolved.record));
  payload.summary = { record: summarizePolitician(resolved.record), sources: politicianSources(resolved.record) };
  payload.sources = politicianSources(resolved.record);
  payload.warnings = standardWarnings();
  payload.nextActions = [
    `abgeordnetenwatch politicians page --id ${resolved.record.id}`,
    `abgeordnetenwatch politicians dossier --id ${resolved.record.id}`
  ];
  emit(payload);
}

async function runPoliticianPage(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const resolved = await resolvePolitician(argv);
  const profileUrl = resolved.rawUrl || resolved.record.abgeordnetenwatch_url;
  if (!profileUrl) {
    throw new CLIError(1, "missing_profile_url", "politician record has no profile URL");
  }
  const page = await fetchPage(profileUrl, parsed.flags.grep || "");
  const payload = envelope("politicians page", page.url);
  payload.summary = page;
  payload.sources = [{ kind: "profile", title: "Public profile page", url: page.url }];
  payload.warnings = standardWarnings();
  payload.nextActions = [`abgeordnetenwatch politicians dossier --id ${resolved.record.id}`];
  emit(payload);
}

async function runPoliticianDossier(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const limit = limitFlag(parsed, 10, 50);
  const resolved = await resolvePolitician(argv);
  const record = resolved.record;
  const id = String(record.id);
  const mandates = await fetchCollection("/candidacies-mandates", { politician: id, range_end: String(limit) }, limit);
  const sidejobs = await sidejobsForMandates(mandates, limit);
  let page: Record<string, any> | null = null;
  if (record.abgeordnetenwatch_url) {
    try {
      page = await fetchPage(record.abgeordnetenwatch_url, parsed.flags.grep || "");
    } catch {
      page = null;
    }
  }
  const payload = envelope("politicians dossier", apiUrlFromRecord(record));
  payload.summary = {
    politician: summarizePolitician(record),
    mandateCount: mandates.length,
    mandates: summarizeRecords(mandates, limit),
    sidejobCount: sidejobs.length,
    sidejobs: summarizeRecords(sidejobs, limit),
    sidejobIncomeSum: sumNumeric(sidejobs, "income"),
    profilePage: page,
    sourceCategories: ["api", "public-profile-page", "mandates", "sidejobs"]
  };
  payload.sources = politicianSources(record);
  payload.warnings = [
    ...standardWarnings(),
    "Side-job income fields may be partial and depend on disclosed Bundestag data as processed by abgeordnetenwatch.",
    "Do not equate outside income or mandates with corruption without independent evidence."
  ];
  payload.nextActions = [
    `abgeordnetenwatch sidejobs for-politician --id ${id} --limit ${limit}`,
    `abgeordnetenwatch politicians page --id ${id} --grep Suchbegriff`
  ];
  emit(payload);
}

async function runMandatesForPolitician(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const limit = limitFlag(parsed, 10, 50);
  const resolved = await resolvePolitician(argv);
  const id = String(resolved.record.id);
  const mandates = await fetchCollection("/candidacies-mandates", { politician: id, range_end: String(limit) }, limit);
  const payload = envelope("mandates for-politician", `${BASE_URL}/candidacies-mandates?politician=${encodeURIComponent(id)}`);
  payload.summary = { politician: summarizePolitician(resolved.record), returned: mandates.length };
  payload.items = summarizeRecords(mandates, limit);
  payload.sources = [{ kind: "api", title: "Candidacies/mandates endpoint", url: BASE_URL + "/candidacies-mandates" }];
  payload.warnings = standardWarnings();
  payload.nextActions = [`abgeordnetenwatch sidejobs for-politician --id ${id}`];
  emit(payload);
}

async function runSidejobsForPolitician(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const limit = limitFlag(parsed, 10, 50);
  const resolved = await resolvePolitician(argv);
  const id = String(resolved.record.id);
  const mandates = await fetchCollection("/candidacies-mandates", { politician: id, range_end: String(limit) }, limit);
  const sidejobs = await sidejobsForMandates(mandates, limit);
  const payload = envelope("sidejobs for-politician", BASE_URL + "/sidejobs");
  payload.summary = {
    politician: summarizePolitician(resolved.record),
    mandates: mandates.length,
    returned: sidejobs.length,
    incomeSum: sumNumeric(sidejobs, "income"),
    clientLimit: limit
  };
  payload.items = summarizeRecords(sidejobs, limit);
  payload.sources = [{ kind: "api", title: "Sidejobs endpoint", url: BASE_URL + "/sidejobs" }];
  payload.warnings = [...standardWarnings(), "Side-job data is disclosure data; interpret categories and income fields cautiously."];
  payload.nextActions = [`abgeordnetenwatch politicians dossier --id ${id} --grep Suchbegriff`];
  emit(payload);
}

async function resolvePolitician(argv: string[]): Promise<{ record: Record<string, any>; rawUrl: string }> {
  const parsed = parseArgs(argv);
  if (parsed.flags.url) {
    let id = idFromProfileUrl(parsed.flags.url);
    if (!id) {
      try {
        const page = await fetchPage(parsed.flags.url, "");
        id = page.politicianId;
      } catch {
        id = "";
      }
    }
    if (!id) {
      throw new CLIError(2, "unsupported_profile_url", "could not infer politician ID from URL; use --id or --name");
    }
    return { record: await getPolitician(id), rawUrl: parsed.flags.url };
  }
  if (parsed.flags.id) {
    return { record: await getPolitician(parsed.flags.id), rawUrl: "" };
  }
  if (parsed.flags.name) {
    const data = await apiJson("/politicians", { "label[cn]": parsed.flags.name, range_end: "1" });
    const rows = dataList(data);
    if (rows.length === 0) {
      throw new CLIError(1, "not_found", "no politician found for name: " + parsed.flags.name);
    }
    return { record: await getPolitician(String(rows[0].id)), rawUrl: "" };
  }
  throw new CLIError(2, "missing_input", "provide --id, --name, or --url");
}

async function getPolitician(id: string): Promise<Record<string, any>> {
  const data = await apiJson("/politicians/" + encodeURIComponent(id));
  if (!data.data || typeof data.data !== "object" || Array.isArray(data.data)) {
    throw new CLIError(1, "not_found", "politician not found: " + id);
  }
  return data.data;
}

async function fetchCollection(path: string, params: Record<string, string>, limit: number): Promise<Record<string, any>[]> {
  if (!params.range_end && !params.pager_limit) {
    params.range_end = String(limit);
  }
  const data = await apiJson(path, params);
  return dataList(data).slice(0, limit);
}

async function sidejobsForMandates(mandates: Record<string, any>[], limit: number): Promise<Record<string, any>[]> {
  const out: Record<string, any>[] = [];
  const seen = new Set<string>();
  for (const mandate of mandates) {
    if (out.length >= limit) {
      break;
    }
    const id = String(mandate.id || "");
    if (!id) {
      continue;
    }
    let rows: Record<string, any>[] = [];
    try {
      rows = await fetchCollection("/sidejobs", { mandates: id, range_end: String(limit) }, limit);
    } catch {
      rows = [];
    }
    for (const row of rows) {
      const rid = String(row.id || "");
      if (seen.has(rid)) {
        continue;
      }
      seen.add(rid);
      out.push(row);
      if (out.length >= limit) {
        break;
      }
    }
  }
  return out;
}

async function fetchPage(rawUrl: string, grep: string): Promise<Record<string, any>> {
  const validUrl = validateAwUrl(rawUrl);
  const resp = await httpGet(validUrl, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8");
  const text = stripHtml(resp.body);
  const page: Record<string, any> = {
    url: resp.url,
    title: firstMatch(resp.body, /<title[^>]*>(.*?)<\/title>/is),
    canonical: attrMatch(resp.body, "link", "rel", "canonical", "href"),
    shortlink: attrMatch(resp.body, "link", "rel", "shortlink", "href"),
    description: metaContent(resp.body, "description"),
    politicianId: politicianIdFromHtml(resp.body),
    textLength: text.length,
    textPreview: text.slice(0, 1200)
  };
  if (grep) {
    page.grep = grep;
    page.snippets = snippets(text, grep, 10);
  }
  return page;
}

async function apiJson(path: string, params: Record<string, string> = {}): Promise<Record<string, any>> {
  const resp = await apiGet(path, params);
  try {
    return JSON.parse(resp.body);
  } catch (error) {
    throw new CLIError(1, "invalid_json", "API did not return JSON: " + String(error));
  }
}

async function apiGet(path: string, params: Record<string, string> = {}): Promise<ApiResponse> {
  const query = Object.keys(params).length ? "?" + new URLSearchParams(params).toString() : "";
  return httpGet(BASE_URL + path + query, "application/json");
}

function httpGet(rawUrl: string, accept: string): Promise<ApiResponse> {
  return new Promise((resolve, reject) => {
    const req = https.request(rawUrl, {
      headers: {
        "Accept": accept,
        "User-Agent": APP_NAME + " (+https://github.com/AlexCasF/germany-skills)"
      },
      timeout: 30000
    }, (res) => {
      const chunks: Buffer[] = [];
      res.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
      res.on("end", () => {
        const body = Buffer.concat(chunks).toString("utf8");
        const status = res.statusCode || 0;
        if (status >= 400) {
          reject(new CLIError(1, "request_failed", `HTTP ${status}: ${body.slice(0, 500)}`));
          return;
        }
        resolve({
          url: rawUrl,
          status,
          contentType: String(res.headers["content-type"] || ""),
          body: body.slice(0, 8 * 1024 * 1024)
        });
      });
    });
    req.on("timeout", () => req.destroy(new Error("request timed out")));
    req.on("error", (error) => reject(new CLIError(1, "request_failed", error.message)));
    req.end();
  });
}

function parseArgs(argv: string[]): ParsedArgs {
  const flags: Record<string, string> = {};
  const params: Record<string, string> = {};
  const positionals: string[] = [];
  let i = 0;
  while (i < argv.length) {
    const token = argv[i];
    if ((token === "--param" || token === "--query") && i + 1 < argv.length) {
      const eq = argv[i + 1].indexOf("=");
      if (eq >= 0) {
        params[argv[i + 1].slice(0, eq)] = argv[i + 1].slice(eq + 1);
      }
      i += 2;
      continue;
    }
    if (token.startsWith("--")) {
      const key = token.slice(2);
      if (i + 1 < argv.length && !argv[i + 1].startsWith("--")) {
        flags[key] = argv[i + 1];
        i += 2;
      } else {
        flags[key] = "true";
        i += 1;
      }
      continue;
    }
    positionals.push(token);
    i += 1;
  }
  return { flags, params, positionals };
}

function normalizeParams(parsed: ParsedArgs): Record<string, string> {
  const params = { ...parsed.params };
  if (parsed.flags.limit && !params.range_end && !params.pager_limit) {
    params.range_end = parsed.flags.limit;
  }
  if (parsed.flags.page) {
    params.page = parsed.flags.page;
  }
  if (parsed.flags["pager-limit"]) {
    params.pager_limit = parsed.flags["pager-limit"];
  }
  if (parsed.flags["related-data"]) {
    params.related_data = parsed.flags["related-data"];
  }
  return params;
}

function limitFlag(parsed: ParsedArgs, defaultValue: number, maximum: number): number {
  const value = Number.parseInt(parsed.flags.limit || "", 10);
  if (!Number.isFinite(value) || value < 1) {
    return defaultValue;
  }
  return Math.min(value, maximum);
}

function summarizeRecords(rows: Record<string, any>[], limit: number): Record<string, any>[] {
  return rows.slice(0, limit).map(summarizeRecord);
}

function summarizeRecord(row: Record<string, any>): Record<string, any> {
  if (row.entity_type === "politician") {
    return summarizePolitician(row);
  }
  const out: Record<string, any> = {};
  for (const key of ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "type", "start_date", "end_date", "income", "income_level", "income_total", "interval", "data_change_date", "job_title_extra", "additional_information"]) {
    if (row[key] !== undefined) {
      out[key] = row[key];
    }
  }
  for (const key of ["sidejob_organization", "party", "parliament_period", "politician"]) {
    if (row[key] && typeof row[key] === "object") {
      out[key] = summarizeReference(row[key]);
    }
  }
  if (row.api_url) {
    out.sources = [{ kind: "api", title: "API record", url: row.api_url }];
  }
  return out;
}

function summarizePolitician(row: Record<string, any>): Record<string, any> {
  const out: Record<string, any> = {};
  for (const key of ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "first_name", "last_name", "sex", "year_of_birth", "education", "occupation", "statistic_questions", "statistic_questions_answered", "ext_id_bundestagsverwaltung", "qid_wikidata"]) {
    if (row[key] !== undefined) {
      out[key] = row[key];
    }
  }
  if (row.party && typeof row.party === "object") {
    out.party = summarizeReference(row.party);
  }
  out.sources = politicianSources(row);
  return out;
}

function summarizeReference(row: Record<string, any>): Record<string, any> {
  const out: Record<string, any> = {};
  for (const key of ["id", "entity_type", "label", "api_url", "abgeordnetenwatch_url"]) {
    if (row[key] !== undefined) {
      out[key] = row[key];
    }
  }
  return out;
}

function politicianSources(row: Record<string, any>): Record<string, string>[] {
  const sources: Record<string, string>[] = [];
  if (row.api_url) {
    sources.push({ kind: "api", title: "API record", url: row.api_url });
  }
  if (row.abgeordnetenwatch_url) {
    sources.push({ kind: "profile", title: "Public profile", url: row.abgeordnetenwatch_url });
  }
  if (row.id !== undefined) {
    sources.push({ kind: "api", title: "Mandates for politician", url: BASE_URL + "/candidacies-mandates?politician=" + encodeURIComponent(String(row.id)) });
  }
  return sources;
}

function dataList(data: Record<string, any>): Record<string, any>[] {
  return Array.isArray(data.data) ? data.data : [];
}

function total(data: Record<string, any>): any {
  return data.meta?.result?.total;
}

function searchSummary(parsed: ParsedArgs): Record<string, string> {
  const out: Record<string, string> = {};
  for (const key of ["name", "first-name", "last-name", "party", "limit"]) {
    if (parsed.flags[key]) {
      out[key] = parsed.flags[key];
    }
  }
  return out;
}

function nextForPoliticianItems(items: Record<string, any>[]): string[] {
  const out: string[] = [];
  for (const item of items) {
    if (item.id === undefined) {
      continue;
    }
    out.push(`abgeordnetenwatch politicians dossier --id ${item.id}`);
    out.push(`abgeordnetenwatch politicians page --id ${item.id} --grep Suchbegriff`);
    if (out.length >= 4) {
      break;
    }
  }
  return out;
}

function envelope(command: string, requestUrl: string): Record<string, any> {
  return {
    tool: APP_NAME,
    command,
    status: "ok",
    retrievedAt: new Date().toISOString(),
    request: { method: "GET", url: requestUrl, redactions: [] },
    summary: {},
    sources: [],
    warnings: [],
    nextActions: []
  };
}

function defaultSources(): Record<string, string>[] {
  return [
    { kind: "documentation", title: "API documentation", url: "https://www.abgeordnetenwatch.de/api" },
    { kind: "documentation", title: "API response format", url: "https://www.abgeordnetenwatch.de/api/response" },
    { kind: "documentation", title: "API changelog", url: "https://www.abgeordnetenwatch.de/api/version-changelog/aktuell" },
    { kind: "license", title: "CC0 1.0", url: "https://creativecommons.org/publicdomain/zero/1.0/deed.de" }
  ];
}

function standardWarnings(): string[] {
  return [
    "abgeordnetenwatch is a transparency platform, not an official parliamentary archive.",
    "Use official Bundestag/DIP records when the exact official parliamentary record matters.",
    "No exact API rate limit was found in official docs; keep requests bounded."
  ];
}

function validateAwUrl(raw: string): string {
  const parsed = new URL(raw);
  if (parsed.hostname !== "www.abgeordnetenwatch.de" && parsed.hostname !== "abgeordnetenwatch.de") {
    throw new CLIError(2, "unsupported_url", "URL must belong to abgeordnetenwatch.de");
  }
  parsed.protocol = parsed.protocol || "https:";
  if (parsed.hostname === "abgeordnetenwatch.de") {
    parsed.hostname = "www.abgeordnetenwatch.de";
  }
  return parsed.toString();
}

function idFromProfileUrl(raw: string): string {
  return raw.match(/\/politician\/([0-9]+)/)?.[1] || "";
}

function politicianIdFromHtml(body: string): string {
  for (const pattern of [/currentPath":"politician\/([0-9]+)"/, /\/politician\/([0-9]+)/, /view_args":"([0-9]+)"/]) {
    const match = body.match(pattern);
    if (match) {
      return match[1];
    }
  }
  return "";
}

function stripHtml(raw: string): string {
  let text = raw.replace(/<script[^>]*>.*?<\/script>|<style[^>]*>.*?<\/style>|<svg[^>]*>.*?<\/svg>/gis, " ");
  text = text.replace(/<[^>]+>/gs, " ");
  return clean(decodeEntities(text));
}

function snippets(text: string, term: string, limit: number): string[] {
  const out: string[] = [];
  const lower = text.toLowerCase();
  const needle = term.toLowerCase();
  let start = 0;
  while (out.length < limit) {
    const idx = lower.indexOf(needle, start);
    if (idx < 0) {
      break;
    }
    const left = Math.max(0, idx - 240);
    const right = Math.min(text.length, idx + term.length + 240);
    out.push(clean(text.slice(left, right)));
    start = idx + term.length;
  }
  return out;
}

function firstMatch(raw: string, pattern: RegExp): string {
  const match = raw.match(pattern);
  return match ? clean(decodeEntities(match[1])) : "";
}

function attrMatch(raw: string, tag: string, attrName: string, attrValue: string, wanted: string): string {
  const re = new RegExp(`<${tag}[^>]*>`, "gis");
  for (const match of raw.matchAll(re)) {
    const tagText = match[0];
    const low = tagText.toLowerCase();
    if (low.includes(`${attrName}="${attrValue}"`) || low.includes(`${attrName}='${attrValue}'`)) {
      const value = attrValueFromTag(tagText, wanted);
      if (value) {
        return decodeEntities(value);
      }
    }
  }
  return "";
}

function metaContent(raw: string, name: string): string {
  for (const match of raw.matchAll(/<meta[^>]*>/gis)) {
    const tagText = match[0];
    const low = tagText.toLowerCase();
    if (low.includes(`name="${name.toLowerCase()}"`) || low.includes(`property="${name.toLowerCase()}"`)) {
      return clean(decodeEntities(attrValueFromTag(tagText, "content")));
    }
  }
  return "";
}

function attrValueFromTag(tag: string, attr: string): string {
  const re = new RegExp(`${attr}\\s*=\\s*["']([^"']+)["']`, "is");
  return tag.match(re)?.[1] || "";
}

function apiUrlFromRecord(row: Record<string, any>): string {
  return row.api_url || (row.id !== undefined ? `${BASE_URL}/politicians/${row.id}` : BASE_URL);
}

function sumNumeric(rows: Record<string, any>[], key: string): number {
  return rows.reduce((sum, row) => typeof row[key] === "number" ? sum + row[key] : sum, 0);
}

function decodeEntities(raw: string): string {
  return raw
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, "\"")
    .replace(/&#39;/g, "'");
}

function clean(value: string): string {
  return (value || "").replace(/\s+/g, " ").trim();
}

function matches(argv: string[], ...parts: string[]): boolean {
  return parts.every((part, index) => argv[index] === part);
}

function isHelp(value: string): boolean {
  return value === "--help" || value === "-h" || value === "help";
}

function emit(value: unknown): void {
  console.log(JSON.stringify(value, null, 2));
}

function fail(exitCode: number, code: string, message: string): void {
  emit({
    tool: APP_NAME,
    status: "error",
    retrievedAt: new Date().toISOString(),
    error: { code, message }
  });
  process.exitCode = exitCode;
}

main(process.argv.slice(2)).then((code) => {
  if (code !== 0) {
    process.exitCode = code;
  }
});
