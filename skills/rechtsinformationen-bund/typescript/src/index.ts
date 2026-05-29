import * as https from "node:https";
import { URL, URLSearchParams } from "node:url";

const APP_NAME = "rechtsinformationen-bund";
const BASE_URL = "https://testphase.rechtsinformationen.bund.de/v1";
const ROOT_URL = "https://testphase.rechtsinformationen.bund.de";

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

type RawCommand = {
  path: string;
  kind: "singleton" | "list" | "document" | "legislation" | "text";
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

const rawCommands: Record<string, RawCommand> = {
  "statistics": { path: "/statistics", kind: "singleton" },
  "documents list": { path: "/document", kind: "list" },
  "documents search": { path: "/document/lucene-search", kind: "list" },
  "documents search-administrative-directive": { path: "/document/lucene-search/administrative-directive", kind: "list" },
  "documents search-case-law": { path: "/document/lucene-search/case-law", kind: "list" },
  "documents search-legislation": { path: "/document/lucene-search/legislation", kind: "list" },
  "documents search-literature": { path: "/document/lucene-search/literature", kind: "list" },
  "administrative-directive list": { path: "/administrative-directive", kind: "list" },
  "administrative-directive get": { path: "/administrative-directive/{documentNumber}", kind: "document" },
  "administrative-directive html": { path: "/administrative-directive/{documentNumber}.html", kind: "text" },
  "administrative-directive xml": { path: "/administrative-directive/{documentNumber}.xml", kind: "text" },
  "case-law list": { path: "/case-law", kind: "list" },
  "case-law courts": { path: "/case-law/courts", kind: "singleton" },
  "case-law get": { path: "/case-law/{documentNumber}", kind: "document" },
  "case-law html": { path: "/case-law/{documentNumber}.html", kind: "text" },
  "case-law xml": { path: "/case-law/{documentNumber}.xml", kind: "text" },
  "legislation list": { path: "/legislation", kind: "list" },
  "legislation get": { path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}", kind: "legislation" },
  "legislation html": { path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.html", kind: "text" },
  "legislation xml": { path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.xml", kind: "text" },
  "legislation article-html": { path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}/{articleEid}.html", kind: "text" },
  "literature list": { path: "/literature", kind: "list" },
  "literature get": { path: "/literature/{documentNumber}", kind: "document" },
  "literature html": { path: "/literature/{documentNumber}.html", kind: "text" },
  "literature xml": { path: "/literature/{documentNumber}.xml", kind: "text" }
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
    } else if (argv[0] === "source" || argv.slice(0, 2).join(" ") === "documents source") {
      await runSource(argv[0] === "source" ? argv.slice(1) : argv.slice(2));
    } else if (argv.slice(0, 2).join(" ") === "documents text") {
      await runText(argv.slice(2));
    } else if (argv.slice(0, 2).join(" ") === "documents dossier") {
      await runDossier(argv.slice(2));
    } else if (argv.slice(0, 2).join(" ") === "case-law dossier") {
      await runDossier(["--type", "case-law", ...argv.slice(2)]);
    } else if (argv.slice(0, 2).join(" ") === "legislation dossier") {
      await runDossier(["--type", "legislation", ...argv.slice(2)]);
    } else if (argv[0] === "cite") {
      await runCite(argv.slice(1));
    } else {
      const resolved = resolveRaw(argv);
      await runRaw(resolved.command, resolved.rest);
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
  console.log(`rechtsinformationen-bund -- official German federal legal information preview API

Purpose
  Search and cite legal information from the Rechtsinformationen des Bundes
  trial service: federal legislation, federal case law, legal literature, and
  administrative directives.

Fast paths
  rechtsinformationen-bund doctor
  rechtsinformationen-bund documents search --search-term "Suchbegriff" --limit 3
  rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision

Raw endpoint commands
  statistics
  documents list|search|search-case-law|search-legislation
  administrative-directive list|get|html|xml
  case-law list|courts|get|html|xml
  legislation list|get|html|xml|article-html
  literature list|get|html|xml

Research commands
  doctor
  source / documents source
  documents text
  documents dossier
  cite
`);
}

function printHelp(path: string[]): void {
  if (path.join(" ") === "documents dossier") {
    console.log(`rechtsinformationen-bund documents dossier

Builds a compact evidence bundle with metadata, source URLs, optional text
snippets, warnings, and next actions.

Examples
  rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision
  rechtsinformationen-bund documents dossier --search-term "Suchbegriff" --grep Suchbegriff
`);
    return;
  }
  if (path.join(" ") === "documents text") {
    console.log(`rechtsinformationen-bund documents text

Fetches the best HTML/XML source rendition for a known document and extracts
plain text plus optional grep snippets.
`);
    return;
  }
  printRootHelp();
}

async function runDoctor(): Promise<void> {
  const stats = await apiJson("/statistics");
  emit(envelope("doctor", {
    authRequired: false,
    baseUrl: BASE_URL,
    rateLimit: "600 requests per minute per client IP",
    rateLimitExceeded: "may return HTTP 503",
    statistics: stats,
    trialService: true
  }, "/statistics"));
}

async function runRaw(command: string, argv: string[]): Promise<void> {
  const raw = rawCommands[command];
  const parsed = parseArgs(argv);
  const path = fillPath(raw.path, parsed, command);
  const params = rawParams(parsed);

  if (command === "documents search" && parsed.flags["search-term"]) {
    await runCompactSearch(path, params, parsed);
    return;
  }

  const resp = await apiGet(path, params);
  if (raw.kind === "text") {
    console.log(resp.body);
    return;
  }
  try {
    emit(JSON.parse(resp.body));
  } catch {
    console.log(resp.body);
  }
}

async function runCompactSearch(path: string, params: Record<string, string>, parsed: ParsedArgs): Promise<void> {
  const data = await apiJson(path, params);
  const members = memberList(data);
  const limit = Number.parseInt(parsed.flags.limit || parsed.flags.size || String(members.length || 10), 10);
  const items = members.slice(0, limit).map(summarizeRecord);
  const payload = envelope("documents search", {
    searchTerm: parsed.flags["search-term"],
    returned: items.length,
    clientLimit: limit,
    totalItems: data.totalItems,
    nextPage: data.view?.next || data["hydra:view"]?.["hydra:next"]
  }, pathWithQuery(path, params));
  payload.items = items;
  emit(payload);
}

async function runSource(argv: string[]): Promise<void> {
  const source = await buildSource(argv);
  emit(envelope("source", source, source.record["@id"] || source.record.id || "/source"));
}

async function runText(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const source = await buildSource(argv);
  const textInfo = await sourceText(source);
  emit(envelope("documents text", {
    record: source.record,
    textLength: textInfo.text.length,
    sourceUrl: textInfo.usedUrl,
    grep: parsed.flags.grep,
    snippets: parsed.flags.grep ? snippets(textInfo.text, parsed.flags.grep) : [],
    textPreview: textInfo.text.slice(0, 1200)
  }, textInfo.usedUrl));
}

async function runDossier(argv: string[]): Promise<void> {
  const parsed = parseArgs(argv);
  const source = await buildSource(argv);
  const summary: Record<string, unknown> = {
    record: source.record,
    sourceCount: source.sourceCount,
    citation: citation(source.record)
  };
  if (parsed.flags.grep) {
    const textInfo = await sourceText(source);
    summary.textSourceUrl = textInfo.usedUrl;
    summary.textLength = textInfo.text.length;
    summary.grep = parsed.flags.grep;
    summary.snippets = snippets(textInfo.text, parsed.flags.grep);
  }
  emit(envelope("documents dossier", summary, source.record["@id"] || "/dossier"));
}

async function runCite(argv: string[]): Promise<void> {
  const source = await buildSource(argv);
  emit(envelope("cite", {
    citation: citation(source.record),
    record: source.record,
    sources: source.record.sources || []
  }, source.record["@id"] || "/cite"));
}

async function buildSource(argv: string[]): Promise<Record<string, any>> {
  const parsed = parseArgs(argv);
  const flags = parsed.flags;
  let record: Record<string, any>;

  if (flags["search-term"]) {
    const data = await apiJson("/document/lucene-search", { searchTerm: flags["search-term"], size: "1" });
    const members = memberList(data);
    if (members.length === 0) {
      throw new CLIError(1, "not_found", "search returned no records");
    }
    const identity = inferRecordIdentity(members[0]);
    record = await apiJson(recordPath(identity.docType, identity.docId));
  } else if (flags.url) {
    record = await apiJson(urlToApiPath(flags.url));
  } else {
    const docId = flags["document-number"] || flags.eli || flags.id;
    const docType = flags.type || inferType(docId);
    if (!docType || !docId) {
      throw new CLIError(2, "missing_input", "provide --type and --document-number/--eli, --url, or --search-term");
    }
    record = await apiJson(recordPath(docType, docId));
  }

  const sources = sourceLinks(record);
  const compact = summarizeRecord(record);
  compact.sources = sources;
  return {
    citationSource: "Rechtsinformationen des Bundes",
    record: compact,
    sourceCount: sources.length
  };
}

async function sourceText(source: Record<string, any>): Promise<{ text: string; usedUrl: string }> {
  const links = source.record.sources || [];
  let chosen = links.find((link: Record<string, string>) => link.kind === "html")?.url;
  if (!chosen) {
    chosen = links.find((link: Record<string, string>) => link.kind === "xml")?.url;
  }
  if (!chosen) {
    throw new CLIError(1, "no_text_source", "no HTML or XML source URL was found");
  }
  const resp = await httpGetAbsolute(chosen);
  const text = resp.contentType.includes("html") || chosen.endsWith(".html") ? stripHtml(resp.body) : stripXml(resp.body);
  return { text, usedUrl: resp.url };
}

function resolveRaw(argv: string[]): { command: string; rest: string[] } {
  for (const width of [3, 2, 1]) {
    const key = argv.slice(0, width).join(" ");
    if (rawCommands[key]) {
      return { command: key, rest: argv.slice(width) };
    }
  }
  throw new CLIError(2, "unknown_command", "unknown command path: " + argv.join(" "));
}

function parseArgs(argv: string[]): ParsedArgs {
  const flags: Record<string, string> = {};
  const params: Record<string, string> = {};
  const positionals: string[] = [];
  let i = 0;
  while (i < argv.length) {
    const token = argv[i];
    if ((token === "--param" || token === "--query") && i + 1 < argv.length) {
      addKeyValue(params, argv[i + 1]);
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

function rawParams(parsed: ParsedArgs): Record<string, string> {
  const params: Record<string, string> = { ...parsed.params };
  const direct: Record<string, string> = {
    "search-term": "searchTerm",
    "size": "size",
    "limit": "size",
    "page-index": "pageIndex",
    "court-type": "courtType",
    "file-number": "fileNumber",
    "decision-date": "decisionDate",
    "document-type": "documentType"
  };
  for (const [flag, param] of Object.entries(direct)) {
    if (parsed.flags[flag]) {
      params[param] = parsed.flags[flag];
    }
  }
  return params;
}

function fillPath(path: string, parsed: ParsedArgs, command: string): string {
  const flags = parsed.flags;
  const positionals = [...parsed.positionals];
  const values: Record<string, string> = { ...flags };
  if (!values.documentNumber && values["document-number"]) {
    values.documentNumber = values["document-number"];
  }
  const placeholders = Array.from(path.matchAll(/\{([^}]+)\}/g)).map((match) => match[1]);
  for (const name of placeholders) {
    const value = values[name] || values[camelToKebab(name)] || positionals.shift();
    if (!value) {
      throw new CLIError(2, "missing_argument", `${command} needs --${camelToKebab(name)}`);
    }
    path = path.replace("{" + name + "}", encodeURIComponent(value));
  }
  return path;
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
  const url = path.startsWith("http") ? path : BASE_URL + pathWithQuery(path, params);
  return httpGetAbsolute(url);
}

function httpGetAbsolute(rawUrl: string): Promise<ApiResponse> {
  return new Promise((resolve, reject) => {
    const req = https.request(rawUrl, {
      headers: {
        "User-Agent": APP_NAME,
        "Accept": "application/json, text/html, application/xml;q=0.9, */*;q=0.8"
      },
      timeout: 30000
    }, (res) => {
      const chunks: Buffer[] = [];
      res.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
      res.on("end", () => {
        const body = Buffer.concat(chunks).toString("utf8");
        const status = res.statusCode || 0;
        if (status >= 400) {
          reject(new CLIError(1, "request_failed", `API returned HTTP ${status}: ${body.slice(0, 400)}`));
          return;
        }
        resolve({
          url: res.headers.location || rawUrl,
          status,
          contentType: String(res.headers["content-type"] || ""),
          body
        });
      });
    });
    req.on("timeout", () => {
      req.destroy(new Error("request timed out"));
    });
    req.on("error", (error) => reject(new CLIError(1, "request_failed", error.message)));
    req.end();
  });
}

function pathWithQuery(path: string, params: Record<string, string>): string {
  const keys = Object.keys(params);
  if (keys.length === 0) {
    return path;
  }
  const query = new URLSearchParams();
  for (const key of keys) {
    query.set(key, params[key]);
  }
  return path + "?" + query.toString();
}

function urlToApiPath(raw: string): string {
  const parsed = new URL(raw);
  if (parsed.hostname !== "testphase.rechtsinformationen.bund.de") {
    throw new CLIError(2, "unsupported_url", "URL must belong to testphase.rechtsinformationen.bund.de");
  }
  let path = parsed.pathname;
  if (path.startsWith("/v1/")) {
    path = path.slice(3);
  }
  return path + parsed.search;
}

function recordPath(docType: string, docId: string): string {
  if (docType === "case-law") {
    return "/case-law/" + encodeURIComponent(docId);
  }
  if (docType === "literature") {
    return "/literature/" + encodeURIComponent(docId);
  }
  if (docType === "administrative-directive") {
    return "/administrative-directive/" + encodeURIComponent(docId);
  }
  if (docType === "legislation") {
    return docId.startsWith("eli/") ? "/legislation/" + docId : "/legislation/eli/" + docId;
  }
  throw new CLIError(2, "unsupported_type", "unsupported --type: " + docType);
}

function sourceLinks(record: Record<string, any>): Record<string, string>[] {
  const links: Record<string, string>[] = [];
  const id = record["@id"] || record.id;
  if (id) {
    links.push({ kind: "api", title: "@id", url: ROOT_URL + id });
  }
  const encodings = Array.isArray(record.encoding) ? record.encoding : [];
  for (const enc of encodings) {
    const contentUrl = enc.contentUrl;
    if (!contentUrl) {
      continue;
    }
    const format = String(enc.encodingFormat || "");
    let kind = "source";
    if (format.includes("html") || contentUrl.endsWith(".html")) {
      kind = "html";
    } else if (format.includes("xml") || contentUrl.endsWith(".xml")) {
      kind = "xml";
    } else if (format.includes("zip") || contentUrl.endsWith(".zip")) {
      kind = "zip";
    }
    links.push({ kind, title: format || kind, url: contentUrl.startsWith("/") ? ROOT_URL + contentUrl : contentUrl });
  }
  return links;
}

function summarizeRecord(record: Record<string, any>): Record<string, any> {
  if (record["@type"] === "SearchResult" && typeof record.item === "object") {
    const out = summarizeRecord(record.item);
    const matches = Array.isArray(record.textMatches) ? record.textMatches : [];
    out.textMatchCount = matches.length;
    if (matches.length > 0) {
      out.firstTextMatch = matches[0];
    }
    out.sources = sourceLinks(record.item);
    return out;
  }

  const out: Record<string, any> = {};
  for (const key of ["@id", "@type", "documentNumber", "ecli", "eli", "legislationIdentifier", "headline", "name", "abbreviation", "decisionDate", "courtName", "courtType", "documentType", "inLanguage"]) {
    if (record[key] !== undefined) {
      out[key] = record[key];
    }
  }
  if (Object.keys(out).length === 0) {
    for (const [key, value] of Object.entries(record).slice(0, 12)) {
      if (["string", "number", "boolean"].includes(typeof value) || value === null) {
        out[key] = value;
      }
    }
  }
  return out;
}

function inferRecordIdentity(record: Record<string, any>): { docType: string; docId: string } {
  if (record["@type"] === "SearchResult" && typeof record.item === "object") {
    return inferRecordIdentity(record.item);
  }
  if (record.documentNumber) {
    return { docType: inferType(record.documentNumber), docId: record.documentNumber };
  }
  if (record.legislationIdentifier) {
    return { docType: "legislation", docId: record.legislationIdentifier };
  }
  if (record.eli) {
    return { docType: "legislation", docId: record.eli };
  }
  const id = String(record["@id"] || "");
  if (id.includes("/case-law/")) {
    return { docType: "case-law", docId: id.split("/").pop() || "" };
  }
  if (id.includes("/legislation/")) {
    return { docType: "legislation", docId: id.split("/legislation/")[1] };
  }
  throw new CLIError(1, "unrecognized_record", "could not infer document identity from search result");
}

function inferType(identifier?: string): string {
  if (!identifier) {
    return "";
  }
  if (identifier.startsWith("eli/")) {
    return "legislation";
  }
  if (identifier.toUpperCase().startsWith("K")) {
    return "case-law";
  }
  return "";
}

function citation(record: Record<string, any>): string {
  if (record.ecli) {
    return [record.courtName, record.decisionDate, record.headline, record.ecli].filter(Boolean).join(", ");
  }
  if (record.legislationIdentifier || record.eli) {
    return [record.name, record.abbreviation, record.legislationIdentifier || record.eli].filter(Boolean).join(", ");
  }
  return record.headline || record.name || record.documentNumber || "Rechtsinformationen des Bundes";
}

function snippets(text: string, term: string): string[] {
  if (!term) {
    return [];
  }
  const result: string[] = [];
  const lower = text.toLowerCase();
  const needle = term.toLowerCase();
  let start = 0;
  while (result.length < 10) {
    const idx = lower.indexOf(needle, start);
    if (idx < 0) {
      break;
    }
    const left = Math.max(0, idx - 220);
    const right = Math.min(text.length, idx + term.length + 220);
    result.push(clean(text.slice(left, right)));
    start = idx + term.length;
  }
  return result;
}

function stripHtml(raw: string): string {
  let text = raw.replace(/<script[^>]*>.*?<\/script>|<style[^>]*>.*?<\/style>/gis, " ");
  text = text.replace(/<[^>]+>/gs, " ");
  return clean(decodeEntities(text));
}

function stripXml(raw: string): string {
  return clean(decodeEntities(raw.replace(/<[^>]+>/gs, " ")));
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

function memberList(data: Record<string, any>): Record<string, any>[] {
  const value = data.member || data["hydra:member"] || [];
  return Array.isArray(value) ? value : [];
}

function addKeyValue(params: Record<string, string>, raw: string): void {
  const idx = raw.indexOf("=");
  if (idx < 0) {
    throw new CLIError(2, "bad_param", "--param expects key=value");
  }
  params[raw.slice(0, idx)] = raw.slice(idx + 1);
}

function camelToKebab(value: string): string {
  return value.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();
}

function envelope(command: string, summary: Record<string, any>, pathOrUrl: string): Record<string, any> {
  const requestUrl = pathOrUrl.startsWith("http") ? pathOrUrl : BASE_URL + pathOrUrl;
  return {
    tool: APP_NAME,
    command,
    status: "ok",
    retrievedAt: new Date().toISOString(),
    request: { method: "GET", url: requestUrl, redactions: [] },
    summary,
    sources: [
      { kind: "portal", title: "Portal", url: ROOT_URL + "/" },
      { kind: "documentation", title: "API documentation", url: "https://docs.rechtsinformationen.bund.de/" },
      { kind: "openapi", title: "OpenAPI JSON", url: ROOT_URL + "/openapi.json" }
    ],
    warnings: [
      "This is a trial service and may change.",
      "The dataset is not yet complete.",
      "Use existing official sources for production-grade legal research."
    ],
    nextActions: [
      "rechtsinformationen-bund documents search --search-term \"Suchbegriff\" --limit 3",
      "rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision"
    ]
  };
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
