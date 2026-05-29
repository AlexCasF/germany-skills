import * as https from "node:https";
import { URL, URLSearchParams } from "node:url";
const APP_NAME = "bundestag-lobbyregister";
const BASE_URL = "https://api.lobbyregister.bundestag.de/rest/v2";
const PUBLIC_URL = "https://www.lobbyregister.bundestag.de";
const RAW_SEARCH_URL = "https://www.lobbyregister.bundestag.de/sucheDetailJson";
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
        if (argv[0] === "doctor") {
            await runDoctor(argv.slice(1));
        }
        else if (argv[0] === "statistics") {
            await runStatistics(argv.slice(1));
        }
        else if (argv[0] === "search") {
            await runSearch(argv.slice(1));
        }
        else if (matches(argv, "entry", "get")) {
            await runEntryGet(argv.slice(2));
        }
        else if (matches(argv, "entry", "source")) {
            await runEntrySource(argv.slice(2));
        }
        else if (matches(argv, "entry", "dossier")) {
            await runEntryDossier(argv.slice(2));
        }
        else if (matches(argv, "financial", "summary")) {
            await runFinancialSummary(argv.slice(2));
        }
        else if (matches(argv, "statements", "list")) {
            await runStatementsList(argv.slice(2));
        }
        else if (matches(argv, "raw", "search")) {
            await runRawSearch(argv.slice(2));
        }
        else {
            throw new CLIError(2, "unknown_command", "unknown command path: " + argv.join(" "));
        }
    }
    catch (error) {
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
function printRootHelp() {
    console.log(`bundestag-lobbyregister -- Bundestag Lobbyregister research CLI

Purpose
  Search and cite public lobby-register data for interests represented
  toward the German Bundestag and Federal Government.

Fast paths
  bundestag-lobbyregister doctor
  bundestag-lobbyregister search --term "Musterverband" --limit 3
  bundestag-lobbyregister entry dossier --register-number <register-number> --grep "Foerderung"
  bundestag-lobbyregister financial summary --register-number <register-number>

Research commands
  doctor
  statistics
  search
  entry get
  entry source
  entry dossier
  financial summary
  statements list

Raw endpoint command
  raw search

Auth
  Prefer LOBBYREGISTER_API_KEY from the environment.
  --apikey still works for local compatibility and is redacted from output.
`);
}
function printHelp(path) {
    const joined = path.join(" ");
    if (joined === "entry dossier") {
        console.log(`bundestag-lobbyregister entry dossier

Builds a compact evidence bundle for one register entry.

Examples
  bundestag-lobbyregister entry dossier --register-number <register-number> --grep "Laerm"
  bundestag-lobbyregister entry dossier --name "Musterverband"
`);
        return;
    }
    if (joined === "search") {
        console.log("bundestag-lobbyregister search\n\nSafe free-text search with compact summaries and a small default limit.");
        return;
    }
    if (joined === "entry get") {
        console.log("bundestag-lobbyregister entry get\n\nFetch one official register entry by register number.");
        return;
    }
    if (joined === "financial summary") {
        console.log("bundestag-lobbyregister financial summary\n\nNormalize financial ranges, funding, donations, membership fees, public allowances, annual-report links, and caveats.");
        return;
    }
    printRootHelp();
}
async function runDoctor(argv) {
    const parsed = parseArgs(argv);
    const key = apiKey(parsed);
    const payload = envelope("doctor", `${BASE_URL}/statistics/registerentries?format=json`);
    payload.summary = {
        authRequired: true,
        apiKeyConfigured: Boolean(key),
        apiKeySource: keySource(parsed),
        baseUrl: BASE_URL,
        publicRegisterUrl: PUBLIC_URL,
        openApiYaml: `${BASE_URL}/R2.21-de.yaml`,
        swaggerUi: `${BASE_URL}/swagger-ui/`,
        termsAndOpenDataPage: `${PUBLIC_URL}/informationen-und-hilfe/open-data-1049716`,
        publishedRateLimit: "not found in official docs reviewed; use small limits and retry politely",
        recommendedDefaultLimit: 5
    };
    payload.sources = defaultSources();
    payload.warnings = standardWarnings();
    if (!key) {
        payload.warnings.push("LOBBYREGISTER_API_KEY is not configured; live API calls will fail.");
        payload.nextActions = ["Set LOBBYREGISTER_API_KEY, then run: bundestag-lobbyregister statistics"];
        emit(payload);
        return;
    }
    try {
        const { data, requestUrl } = await apiJson("/statistics/registerentries", { format: "json" }, key);
        payload.request.url = requestUrl;
        payload.summary.health = {
            ok: true,
            sourceDate: data.sourceDate,
            totalLobbyists: get(data, "lobbyists", "totalNumber"),
            activeLobbyists: get(data, "lobbyists", "active", "number"),
            inactiveLobbyists: get(data, "lobbyists", "inactive", "number")
        };
    }
    catch (error) {
        payload.status = "error";
        payload.summary.health = {
            ok: false,
            error: redact(error instanceof Error ? error.message : String(error))
        };
    }
    payload.nextActions = [
        'bundestag-lobbyregister search --term "Musterverband" --limit 3',
        "bundestag-lobbyregister entry dossier --register-number <register-number>"
    ];
    emit(payload);
}
async function runStatistics(argv) {
    const parsed = parseArgs(argv);
    const key = requireKey(parsed);
    const { data, requestUrl } = await apiJson("/statistics/registerentries", { format: "json" }, key);
    const payload = envelope("statistics", requestUrl);
    payload.summary = {
        source: data.source,
        sourceDate: data.sourceDate,
        totalLobbyists: get(data, "lobbyists", "totalNumber"),
        activeLobbyists: get(data, "lobbyists", "active", "number"),
        inactiveLobbyists: get(data, "lobbyists", "inactive", "number"),
        peopleInvolved: get(data, "lobbyists", "peopleInvolvedInLobbyistWork", "totalNumber")
    };
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    payload.sources = defaultSources();
    payload.nextActions = ['bundestag-lobbyregister search --term "Energie" --limit 5'];
    emit(payload);
}
async function runSearch(argv) {
    const parsed = parseArgs(argv);
    const key = requireKey(parsed);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.flags.name);
    if (!term)
        throw new CLIError(2, "missing_term", "search requires --term, --q, or --name");
    const limit = limitFlag(parsed, 5, 25);
    const params = { format: "json", q: term };
    if (parsed.flags.cursor)
        params.cursor = parsed.flags.cursor;
    const { data, requestUrl } = await apiJson("/registerentries", params, key);
    const results = Array.isArray(data.results) ? data.results : [];
    const items = results.slice(0, limit).map((entry) => summarizeEntry(asObject(entry)));
    const payload = envelope("search", requestUrl);
    payload.summary = {
        query: term,
        returnedByApi: data.resultCount,
        totalResultCount: data.totalResultCount,
        limitApplied: limit,
        cursorPresent: Boolean(data.cursor),
        sourceDate: data.sourceDate
    };
    payload.items = items;
    payload.sources = defaultSources();
    payload.warnings = standardWarnings();
    payload.nextActions = searchNextActions(items);
    if (flagBool(parsed, "include-raw"))
        payload.raw = data;
    emit(payload);
}
async function runEntryGet(argv) {
    const parsed = parseArgs(argv);
    const { entry, requestUrl } = await getEntryFromArgs(parsed);
    const payload = envelope("entry get", requestUrl);
    payload.summary = summarizeEntry(entry);
    payload.sources = entrySources(entry);
    payload.warnings = standardWarnings();
    payload.nextActions = nextActionsForEntry(entry);
    if (flagBool(parsed, "include-raw"))
        payload.raw = entry;
    emit(payload);
}
async function runEntrySource(argv) {
    const parsed = parseArgs(argv);
    const { entry, requestUrl } = await getEntryFromArgs(parsed);
    const payload = envelope("entry source", requestUrl);
    payload.summary = {
        registerNumber: entry.registerNumber,
        name: get(entry, "lobbyistIdentity", "name"),
        version: get(entry, "registerEntryDetails", "version"),
        sourceDate: entry.sourceDate
    };
    payload.sources = entrySources(entry);
    payload.nextActions = nextActionsForEntry(entry);
    emit(payload);
}
async function runEntryDossier(argv) {
    const parsed = parseArgs(argv);
    const { entry, requestUrl } = await getEntryFromArgs(parsed);
    const limit = limitFlag(parsed, 5, 20);
    const payload = envelope("entry dossier", requestUrl);
    payload.summary = summarizeEntry(entry);
    payload.financial = financialBlock(entry);
    payload.regulatoryProjects = compactProjects(entry, limit);
    payload.statements = compactStatements(entry, parsed.flags.grep || "", limit);
    payload.sources = entrySources(entry);
    payload.warnings = standardWarnings();
    payload.nextActions = nextActionsForEntry(entry);
    if (flagBool(parsed, "include-raw"))
        payload.raw = entry;
    emit(payload);
}
async function runFinancialSummary(argv) {
    const parsed = parseArgs(argv);
    const { entry, requestUrl } = await getEntryFromArgs(parsed);
    const payload = envelope("financial summary", requestUrl);
    payload.summary = {
        registerNumber: entry.registerNumber,
        name: get(entry, "lobbyistIdentity", "name"),
        sourceDate: entry.sourceDate
    };
    payload.financial = financialBlock(entry);
    payload.sources = entrySources(entry);
    payload.warnings = [...standardWarnings(), "Financial ranges are register disclosures, not audited findings by this tool."];
    payload.nextActions = nextActionsForEntry(entry);
    emit(payload);
}
async function runStatementsList(argv) {
    const parsed = parseArgs(argv);
    const { entry, requestUrl } = await getEntryFromArgs(parsed);
    const limit = limitFlag(parsed, 10, 50);
    const payload = envelope("statements list", requestUrl);
    payload.summary = {
        registerNumber: entry.registerNumber,
        name: get(entry, "lobbyistIdentity", "name"),
        statementsPresent: get(entry, "statements", "statementsPresent"),
        statementsCount: get(entry, "statements", "statementsCount"),
        limitApplied: limit
    };
    payload.items = compactStatements(entry, parsed.flags.grep || "", limit);
    payload.sources = entrySources(entry);
    payload.warnings = [...standardWarnings(), "Statement text may include copyrighted material; quote only short excerpts."];
    payload.nextActions = nextActionsForEntry(entry);
    emit(payload);
}
async function runRawSearch(argv) {
    const parsed = parseArgs(argv);
    const params = new URLSearchParams(parsed.params);
    for (const [key, value] of Object.entries(parsed.flags)) {
        if (!["include-raw", "timeout"].includes(key))
            params.set(key, value);
    }
    const url = `${RAW_SEARCH_URL}?${params.toString()}`;
    const body = await httpGetText(url, {});
    console.log(body);
}
async function getEntryFromArgs(parsed) {
    const key = requireKey(parsed);
    let registerNumber = firstNonEmpty(parsed.flags["register-number"], parsed.flags.registerNumber, parsed.flags.id);
    if (!registerNumber && parsed.flags.name) {
        const first = await searchFirst(parsed.flags.name, key);
        registerNumber = asString(first.entry.registerNumber);
    }
    if (!registerNumber)
        throw new CLIError(2, "missing_register_number", "requires --register-number or --name");
    if (!/^R[0-9]{6}$/.test(registerNumber)) {
        throw new CLIError(2, "invalid_register_number", "register number must look like <register-number>");
    }
    let path = `/registerentries/${encodeURIComponent(registerNumber)}`;
    if (parsed.flags.version)
        path += `/${encodeURIComponent(parsed.flags.version)}`;
    const { data, requestUrl } = await apiJson(path, { format: "json" }, key);
    return { entry: data, requestUrl };
}
async function searchFirst(term, key) {
    const { data, requestUrl } = await apiJson("/registerentries", { format: "json", q: term }, key);
    const results = Array.isArray(data.results) ? data.results : [];
    if (results.length === 0)
        throw new CLIError(1, "not_found", "no register entry found for name: " + term);
    return { entry: asObject(results[0]), requestUrl };
}
async function apiJson(path, params, key) {
    if (!params.format)
        params.format = "json";
    const url = new URL(BASE_URL + path);
    for (const [paramKey, value] of Object.entries(params))
        url.searchParams.set(paramKey, value);
    const body = await httpGetText(url.toString(), {
        Authorization: "ApiKey " + key,
        Accept: "application/json"
    });
    return { data: JSON.parse(body), requestUrl: sanitizeUrl(url.toString()) };
}
function httpGetText(rawUrl, headers) {
    return new Promise((resolve, reject) => {
        const request = https.get(rawUrl, { headers, timeout: 60000 }, (response) => {
            const chunks = [];
            response.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
            response.on("end", () => {
                const body = Buffer.concat(chunks).toString("utf8");
                if ((response.statusCode || 0) >= 400) {
                    reject(new CLIError(1, "http_error", `HTTP ${response.statusCode} from Lobbyregister API: ${truncate(body, 300)}`));
                    return;
                }
                resolve(body);
            });
        });
        request.on("timeout", () => {
            request.destroy(new Error("request timed out"));
        });
        request.on("error", reject);
    });
}
function summarizeEntry(entry) {
    return {
        registerNumber: entry.registerNumber,
        name: get(entry, "lobbyistIdentity", "name"),
        identity: get(entry, "lobbyistIdentity", "identity"),
        legalForm: firstNonEmpty(asString(get(entry, "lobbyistIdentity", "legalForm", "de")), asString(get(entry, "lobbyistIdentity", "legalForm", "en"))),
        activeLobbyist: get(entry, "accountDetails", "activeLobbyist"),
        firstPublicationDate: get(entry, "accountDetails", "firstPublicationDate"),
        lastUpdateDate: get(entry, "accountDetails", "lastUpdateDate"),
        version: get(entry, "registerEntryDetails", "version"),
        detailsPageUrl: get(entry, "registerEntryDetails", "detailsPageUrl"),
        pdfUrl: get(entry, "registerEntryDetails", "pdfUrl"),
        financialExpensesEuro: get(entry, "financialExpenses", "financialExpensesEuro"),
        financialFiscalYear: fiscalYear(entry, "financialExpenses"),
        employeeFTE: get(entry, "employeesInvolvedInLobbying", "employeeFTE"),
        fieldsOfInterest: labelsFromArray(asArray(get(entry, "activitiesAndInterests", "fieldsOfInterest")), 10),
        activityDescriptionHint: truncate(asString(get(entry, "activitiesAndInterests", "activityDescription")), 280),
        mainFundingSources: labelsFromArray(asArray(get(entry, "mainFundingSources", "mainFundingSources")), 8),
        totalDonationsEuro: get(entry, "donators", "totalDonationsEuro"),
        totalMembershipFees: get(entry, "membershipFees", "totalMembershipFees"),
        publicAllowancesPresent: get(entry, "publicAllowances", "publicAllowancesPresent"),
        regulatoryProjectsCount: get(entry, "regulatoryProjects", "regulatoryProjectsCount"),
        statementsCount: get(entry, "statements", "statementsCount"),
        contractsCount: get(entry, "contracts", "contractsCount")
    };
}
function financialBlock(entry) {
    return {
        financialExpenses: {
            fiscalYear: fiscalYear(entry, "financialExpenses"),
            rangeEuro: get(entry, "financialExpenses", "financialExpensesEuro")
        },
        mainFundingSources: labelsFromArray(asArray(get(entry, "mainFundingSources", "mainFundingSources")), 20),
        publicAllowances: get(entry, "publicAllowances"),
        donations: {
            fiscalYear: fiscalYear(entry, "donators"),
            totalEuro: get(entry, "donators", "totalDonationsEuro"),
            items: compactNamedItems(asArray(get(entry, "donators", "donators")), 20)
        },
        membershipFees: {
            fiscalYear: fiscalYear(entry, "membershipFees"),
            totalEuro: get(entry, "membershipFees", "totalMembershipFees"),
            individualContributors: compactNamedItems(asArray(get(entry, "membershipFees", "individualContributors")), 20)
        },
        annualReport: {
            exists: get(entry, "annualReports", "annualReportLastFiscalYearExists"),
            pdfUrl: get(entry, "annualReports", "annualReportPdfUrl")
        }
    };
}
function compactProjects(entry, limit) {
    return asArray(get(entry, "regulatoryProjects", "regulatoryProjects")).slice(0, limit).map((value) => {
        const project = asObject(value);
        return {
            number: project.regulatoryProjectNumber,
            title: project.title,
            descriptionHint: truncate(asString(project.description), 320),
            affectedLaws: labelsFromArray(asArray(project.affectedLaws), 8),
            fieldsOfInterest: labelsFromArray(asArray(project.fieldsOfInterest), 8),
            projectUrl: project.projectUrl
        };
    });
}
function compactStatements(entry, grep, limit) {
    const out = [];
    for (const value of asArray(get(entry, "statements", "statements"))) {
        if (out.length >= limit)
            break;
        const statement = asObject(value);
        const text = asString(get(statement, "text", "text"));
        const item = {
            regulatoryProjectNumber: statement.regulatoryProjectNumber,
            regulatoryProjectTitle: statement.regulatoryProjectTitle,
            pdfUrl: statement.pdfUrl,
            pdfPageCount: statement.pdfPageCount,
            recipientGroups: statement.recipientGroups,
            textPreview: truncate(text, 420)
        };
        if (grep) {
            const hits = snippets(text, grep, 3);
            if (hits.length === 0)
                continue;
            item.snippets = hits;
        }
        out.push(item);
    }
    return out;
}
function entrySources(entry) {
    const sources = defaultSources();
    addSource(sources, "Public detail page", asString(get(entry, "registerEntryDetails", "detailsPageUrl")), "public-page");
    addSource(sources, "Public PDF export", asString(get(entry, "registerEntryDetails", "pdfUrl")), "pdf");
    addSource(sources, "Annual report PDF", asString(get(entry, "annualReports", "annualReportPdfUrl")), "pdf");
    for (const value of asArray(get(entry, "statements", "statements"))) {
        const statement = asObject(value);
        addSource(sources, "Statement PDF: " + asString(statement.regulatoryProjectTitle), asString(statement.pdfUrl), "statement-pdf");
    }
    return sources;
}
function addSource(sources, title, url, kind) {
    if (url)
        sources.push({ title, url, kind });
}
function defaultSources() {
    return [
        { title: "Bundestag Lobbyregister", url: PUBLIC_URL, kind: "official-register" },
        { title: "Open Data/API page", url: `${PUBLIC_URL}/informationen-und-hilfe/open-data-1049716`, kind: "terms" },
        { title: "Swagger UI", url: `${BASE_URL}/swagger-ui/`, kind: "api-docs" },
        { title: "OpenAPI YAML", url: `${BASE_URL}/R2.21-de.yaml`, kind: "openapi" }
    ];
}
function standardWarnings() {
    return [
        "API calls require an API key; this tool redacts keys from normalized output.",
        "Register disclosures describe published self-reported register data; corroborate contentious claims with additional official sources.",
        "Use small limits for broad searches; the upstream search endpoint returns full-detail records."
    ];
}
function nextActionsForEntry(entry) {
    const rn = asString(entry.registerNumber);
    if (!rn)
        return [];
    return [
        `bundestag-lobbyregister entry source --register-number ${rn}`,
        `bundestag-lobbyregister financial summary --register-number ${rn}`,
        `bundestag-lobbyregister statements list --register-number ${rn} --grep <term>`
    ];
}
function searchNextActions(items) {
    return items
        .map((item) => asString(item.registerNumber))
        .filter(Boolean)
        .slice(0, 5)
        .map((rn) => `bundestag-lobbyregister entry dossier --register-number ${rn}`);
}
function parseArgs(argv) {
    const out = { flags: {}, params: {}, positionals: [] };
    for (let i = 0; i < argv.length; i++) {
        const arg = argv[i];
        if (arg === "--param" && i + 1 < argv.length) {
            const [key, value] = splitKeyValue(argv[++i]);
            if (key)
                out.params[key] = value;
            continue;
        }
        if (arg.startsWith("--param=")) {
            const [key, value] = splitKeyValue(arg.slice("--param=".length));
            if (key)
                out.params[key] = value;
            continue;
        }
        if (arg.startsWith("--")) {
            const name = arg.slice(2);
            if (name.includes("=")) {
                const [key, value] = splitKeyValue(name);
                out.flags[key] = value;
            }
            else if (i + 1 < argv.length && !argv[i + 1].startsWith("--")) {
                out.flags[name] = argv[++i];
            }
            else {
                out.flags[name] = "true";
            }
            continue;
        }
        out.positionals.push(arg);
    }
    return out;
}
function splitKeyValue(raw) {
    const index = raw.indexOf("=");
    if (index < 0)
        return ["", ""];
    return [raw.slice(0, index), raw.slice(index + 1)];
}
function requireKey(parsed) {
    const key = apiKey(parsed);
    if (!key)
        throw new CLIError(2, "missing_api_key", "set LOBBYREGISTER_API_KEY or pass --apikey");
    return key;
}
function apiKey(parsed) {
    return parsed.flags.apikey || process.env.LOBBYREGISTER_API_KEY || "";
}
function keySource(parsed) {
    if (parsed.flags.apikey)
        return "flag:redacted";
    if (process.env.LOBBYREGISTER_API_KEY)
        return "env:LOBBYREGISTER_API_KEY";
    return "missing";
}
function envelope(command, requestUrl) {
    return {
        status: "ok",
        tool: APP_NAME,
        command,
        retrievedAt: new Date().toISOString(),
        request: {
            method: "GET",
            url: sanitizeUrl(requestUrl),
            authConfigured: true,
            redactedHeaders: ["Authorization"],
            redactedQueryKeys: ["apikey"]
        }
    };
}
function emit(payload) {
    console.log(JSON.stringify(payload, null, 2));
}
function fail(exitCode, code, message) {
    emit({
        status: "error",
        tool: APP_NAME,
        retrievedAt: new Date().toISOString(),
        error: { code, message: redact(message) }
    });
    process.exit(exitCode);
}
function get(obj, ...path) {
    let cur = obj;
    for (const key of path) {
        if (!cur || typeof cur !== "object" || Array.isArray(cur))
            return undefined;
        cur = cur[key];
    }
    return cur;
}
function asObject(value) {
    return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}
function asArray(value) {
    return Array.isArray(value) ? value : [];
}
function asString(value) {
    if (value === undefined || value === null)
        return "";
    return String(value);
}
function labelsFromArray(items, limit) {
    const labels = [];
    for (const value of items.slice(0, limit)) {
        const item = asObject(value);
        const label = firstNonEmpty(asString(item.de), asString(item.title), asString(item.name), asString(item.en), asString(item.code));
        if (label)
            labels.push(label);
    }
    return labels;
}
function compactNamedItems(items, limit) {
    return items.slice(0, limit).map((value) => {
        const item = asObject(value);
        return {
            name: firstNonEmpty(asString(item.name), asString(item.lastName)),
            rawHint: truncate(JSON.stringify(item), 240)
        };
    });
}
function fiscalYear(entry, block) {
    return {
        finished: get(entry, block, "relatedFiscalYearFinished"),
        start: get(entry, block, "relatedFiscalYearStart"),
        end: get(entry, block, "relatedFiscalYearEnd")
    };
}
function snippets(text, term, limit) {
    const hits = [];
    const lower = text.toLowerCase();
    const needle = term.toLowerCase();
    let start = 0;
    while (hits.length < limit) {
        const idx = lower.indexOf(needle, start);
        if (idx < 0)
            break;
        const left = Math.max(0, idx - 160);
        const right = Math.min(text.length, idx + term.length + 160);
        hits.push(collapseSpace(text.slice(left, right)));
        start = idx + term.length;
    }
    return hits;
}
function truncate(text, limit) {
    const collapsed = collapseSpace(text);
    return collapsed.length <= limit ? collapsed : collapsed.slice(0, limit - 3) + "...";
}
function collapseSpace(text) {
    return String(text).split(/\s+/).filter(Boolean).join(" ");
}
function limitFlag(parsed, fallback, maximum) {
    const parsedValue = Number.parseInt(parsed.flags.limit || String(fallback), 10);
    if (!Number.isFinite(parsedValue) || parsedValue < 1)
        return fallback;
    return Math.min(parsedValue, maximum);
}
function flagBool(parsed, name) {
    return ["1", "true", "yes"].includes((parsed.flags[name] || "").toLowerCase());
}
function firstNonEmpty(...values) {
    for (const value of values) {
        if (value && value.trim())
            return value;
    }
    return "";
}
function sanitizeUrl(raw) {
    try {
        const url = new URL(raw);
        if (url.searchParams.has("apikey"))
            url.searchParams.set("apikey", "REDACTED");
        return url.toString();
    }
    catch {
        return redact(raw);
    }
}
function redact(text) {
    return String(text)
        .replace(/(apikey=)[^&\s]+/gi, "$1REDACTED")
        .replace(/ApiKey\s+[A-Za-z0-9._-]+/gi, "ApiKey REDACTED")
        .replace(/(--apikey\s+)[A-Za-z0-9._-]+/gi, "$1REDACTED");
}
function isHelp(arg) {
    return arg === "-h" || arg === "--help" || arg === "help";
}
function matches(argv, ...parts) {
    return parts.every((part, index) => argv[index] === part);
}
main(process.argv.slice(2)).then((code) => {
    if (code !== 0)
        process.exit(code);
});
