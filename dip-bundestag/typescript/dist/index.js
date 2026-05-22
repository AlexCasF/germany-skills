import { request } from "node:https";
const APP_NAME = "dipctl";
const BASE_URL = "https://search.dip.bundestag.de/api/v1";
const ENTITIES = new Set([
    "vorgang",
    "vorgangsposition",
    "drucksache",
    "drucksache-text",
    "plenarprotokoll",
    "plenarprotokoll-text",
    "person",
    "aktivitaet",
]);
class CliError extends Error {
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
        printHelpFor(argv.slice(0, -1));
        return 0;
    }
    try {
        if (argv[0] === "doctor") {
            await runDoctor(argv.slice(1));
        }
        else if (argv.length >= 2 && argv[0] === "person" && argv[1] === "search") {
            await runPersonSearch(argv.slice(2));
        }
        else if (argv.length >= 2 && argv[0] === "person" && argv[1] === "dossier") {
            await runPersonDossier(argv.slice(2));
        }
        else if (argv.length >= 2 && argv[0] === "vorgang" && argv[1] === "dossier") {
            await runVorgangDossier(argv.slice(2));
        }
        else if (argv.length >= 1 && argv[0] === "source") {
            await runSource(argv.slice(1));
        }
        else if (argv.length >= 2 && (argv[0] === "plenarprotokoll" || argv[0] === "drucksache") && argv[1] === "text") {
            await runDocumentText(argv[0], argv.slice(2));
        }
        else if (argv.length >= 3 && argv[0] === "plenary" && argv[1] === "speech" && argv[2] === "search") {
            await runPlenarySpeechSearch(argv.slice(3));
        }
        else if (argv.length >= 2 && ENTITIES.has(argv[0]) && argv[1] === "list") {
            await runLegacyList(argv[0], argv.slice(2));
        }
        else if (argv.length >= 2 && ENTITIES.has(argv[0]) && argv[1] === "get") {
            await runLegacyGet(argv[0], argv.slice(2));
        }
        else {
            fail(2, "unknown_command", "unknown command path: " + argv.join(" "));
        }
    }
    catch (err) {
        if (err instanceof CliError) {
            fail(err.exitCode, err.code, err.message);
        }
        throw err;
    }
    return 0;
}
function printRootHelp() {
    console.log(`dipctl -- official Bundestag DIP research CLI

Purpose
  Search and cite official parliamentary material from the Bundestag DIP API.

Use this when
  - you need official proceedings, printed papers, protocols, people, or activities
  - you need to distinguish official plenary records from media or campaign quotes
  - you need API-backed source URLs and citation metadata

Do not use this when
  - you need general news context
  - you need lobbying register financial data
  - you need live Bundestag presentation feeds

Fast paths
  Check auth and endpoint health:
    dipctl doctor

  Find a person:
    dipctl person search --name "Gauweiler" --limit 3

  Build an evidence bundle:
    dipctl person dossier --name "Gauweiler"

Legacy endpoint commands
  dipctl vorgang list|get
  dipctl drucksache list|get
  dipctl plenarprotokoll list|get
  dipctl person list|get
  dipctl aktivitaet list|get

Research commands
  doctor
  person search
  person dossier
  vorgang dossier
  source
  plenarprotokoll text
  drucksache text
  plenary speech search

Auth
  Prefer DIP_API_KEY from the environment. --apikey remains supported.
`);
}
function printHelpFor(path) {
    if (path[0] === "person" && path[1] === "dossier") {
        console.log(`dipctl person dossier

What it does
  Builds a compact official-source evidence bundle for one person.

Inputs
  --id      Stable DIP person ID
  --name    Name to search when ID is not known
  --limit   Related activity limit, default 10

Examples
  dipctl person dossier --id 760
  dipctl person dossier --name "Gauweiler"
`);
    }
    else if (path[0] === "doctor") {
        console.log("dipctl doctor\n\nWhat it does\n  Checks auth and endpoint health without printing the API key.");
    }
    else {
        printRootHelp();
    }
}
async function runDoctor(args) {
    const { flags } = parseArgs(args);
    const [key, source] = resolveKey(flags);
    const out = {
        status: key ? "ok" : "error",
        tool: APP_NAME,
        command: "doctor",
        retrievedAt: now(),
        summary: {
            baseUrl: BASE_URL,
            authRequired: true,
            apiKeyConfigured: Boolean(key),
            apiKeySource: source,
            maxConcurrentRequests: 25,
            normalListMaxItems: 100,
            fullTextListMaxItems: "usually 10",
        },
        sources: docSources(),
        warnings: [
            "Do not exceed 25 concurrent API requests.",
            "Detailed rate-limit internals beyond official notes are not published.",
            "Use source attribution: Deutscher Bundestag/Bundesrat - DIP.",
        ],
        nextActions: [
            'dipctl person search --name "Gauweiler"',
            'dipctl plenarprotokoll text --document-number "20/139" --grep "Bürgergeld"',
        ],
    };
    if (!key) {
        out.error = { code: "missing_api_key", message: "Set DIP_API_KEY or pass --apikey." };
        writeJson(out);
        process.exit(2);
    }
    const { body, requestUrl } = await apiGet("person", { "f.person": ["Steinmeier"], format: ["json"] }, key);
    JSON.parse(body);
    out.request = requestMeta(requestUrl);
    out.summary.healthStatusCode = 200;
    writeJson(out);
}
async function runLegacyList(entity, args) {
    const { flags, params } = parseArgs(args);
    const key = mustKey(flags);
    if (!params.format)
        params.format = ["json"];
    let { body } = await apiGet(entity, params, key);
    if (flags.limit) {
        const data = JSON.parse(body);
        if (Array.isArray(data.documents)) {
            const limit = positiveInt(flags.limit, "limit");
            data.documents = data.documents.slice(0, limit);
            data.clientLimit = limit;
            data.clientReturned = data.documents.length;
            body = JSON.stringify(data, null, 2);
        }
    }
    console.log(body);
}
async function runLegacyGet(entity, args) {
    const { flags, params } = parseArgs(args);
    const id = flags.id;
    if (!id)
        throw new CliError(2, "invalid_arguments", "missing required flag --id");
    const key = mustKey(flags);
    if (!params.format)
        params.format = ["json"];
    const { body } = await apiGet(`${entity}/${encodeURIComponent(id)}`, params, key);
    console.log(body);
}
async function runPersonSearch(args) {
    const { flags } = parseArgs(args);
    const name = flags.name;
    if (!name)
        throw new CliError(2, "invalid_arguments", "missing required flag --name");
    const limit = positiveInt(flags.limit ?? "10", "limit");
    const key = mustKey(flags);
    const { body, requestUrl } = await apiGet("person", { "f.person": [name], format: ["json"] }, key);
    const data = JSON.parse(body);
    const docs = takeDocuments(data, limit);
    writeJson({
        status: "ok",
        tool: APP_NAME,
        command: "person search",
        retrievedAt: now(),
        request: requestMeta(requestUrl),
        summary: { query: name, numFound: data.numFound, returned: docs.length, clientLimit: limit },
        items: docs.map(compactItem),
        sources: [{ title: "DIP API person endpoint", url: BASE_URL + "/person", kind: "api" }],
        warnings: [],
        nextActions: ["dipctl person dossier --id <id>", 'dipctl aktivitaet list --param "f.person_id=<id>"'],
    });
}
async function runPersonDossier(args) {
    const { flags } = parseArgs(args);
    const key = mustKey(flags);
    const limit = positiveInt(flags.limit ?? "10", "limit");
    let id = flags.id;
    let searchSummary = null;
    if (!id) {
        if (!flags.name)
            throw new CliError(2, "invalid_arguments", "pass --id or --name");
        const { body, requestUrl } = await apiGet("person", { "f.person": [flags.name], format: ["json"] }, key);
        const docs = takeDocuments(JSON.parse(body), 1);
        if (docs.length === 0)
            throw new CliError(1, "not_found", "no person found for --name");
        id = String(docs[0].id);
        searchSummary = { request: requestMeta(requestUrl), selected: compactItem(docs[0]) };
    }
    const personResp = await apiGet(`person/${encodeURIComponent(id)}`, { format: ["json"] }, key);
    const person = JSON.parse(personResp.body);
    const warnings = [
        "Dossier uses official DIP person and activity records only.",
        "Outside quotes, campaign statements, and news context are not included.",
    ];
    let activities = [];
    try {
        const actResp = await apiGet("aktivitaet", { "f.person_id": [id], format: ["json"] }, key);
        activities = takeDocuments(JSON.parse(actResp.body), limit).map(compactItem);
    }
    catch (err) {
        if (err instanceof CliError)
            warnings.push("Related activities could not be loaded: " + err.message);
    }
    writeJson({
        status: "ok",
        tool: APP_NAME,
        command: "person dossier",
        retrievedAt: now(),
        request: { person: requestMeta(personResp.requestUrl) },
        summary: { person: compactItem(person), relatedActivitiesShown: activities.length, search: searchSummary },
        record: person,
        related: { activities },
        sources: dedupeSources([...extractSources(person), { title: "DIP API person detail", url: BASE_URL + "/person/" + id, kind: "api" }]),
        warnings,
        nextActions: [`dipctl aktivitaet list --param "f.person_id=${id}"`, `dipctl plenary speech search --person-id ${id} --term <term>`],
    });
}
async function runVorgangDossier(args) {
    const { flags } = parseArgs(args);
    const id = flags.id;
    if (!id)
        throw new CliError(2, "invalid_arguments", "missing required flag --id");
    const key = mustKey(flags);
    const limit = positiveInt(flags.limit ?? "10", "limit");
    const recordResp = await apiGet(`vorgang/${encodeURIComponent(id)}`, { format: ["json"] }, key);
    const record = JSON.parse(recordResp.body);
    const warnings = ["Dossier uses official DIP proceeding and proceeding-position records."];
    let positions = [];
    try {
        const posResp = await apiGet("vorgangsposition", { "f.vorgang": [id], format: ["json"] }, key);
        positions = takeDocuments(JSON.parse(posResp.body), limit).map(compactItem);
    }
    catch (err) {
        if (err instanceof CliError)
            warnings.push("Related positions could not be loaded: " + err.message);
    }
    writeJson({
        status: "ok",
        tool: APP_NAME,
        command: "vorgang dossier",
        retrievedAt: now(),
        request: requestMeta(recordResp.requestUrl),
        summary: { vorgang: compactItem(record), relatedPositionsShown: positions.length },
        record,
        related: { positions },
        sources: dedupeSources([...extractSources(record), { title: "DIP API proceeding detail", url: BASE_URL + "/vorgang/" + id, kind: "api" }]),
        warnings,
        nextActions: [`dipctl vorgangsposition list --param "f.vorgang=${id}"`],
    });
}
async function runSource(args) {
    const { flags } = parseArgs(args);
    const entity = flags.type ?? flags.entity;
    if (!entity)
        throw new CliError(2, "invalid_arguments", "missing required flag --type");
    if (!ENTITIES.has(entity))
        throw new CliError(2, "invalid_arguments", "unknown --type: " + entity);
    const key = mustKey(flags);
    const { record, request } = await resolveRecord(entity, flags.id, flags["document-number"], key);
    const sources = extractSources(record);
    const id = String(record.id ?? "");
    if (id)
        sources.push({ title: "DIP API " + entity + " detail", url: BASE_URL + "/" + entity + "/" + id, kind: "api" });
    writeJson({
        status: "ok",
        tool: APP_NAME,
        command: "source",
        retrievedAt: now(),
        request,
        summary: {
            entity,
            record: compactItem(record),
            sourceCount: dedupeSources(sources).length,
            citationSource: "Deutscher Bundestag/Bundesrat - DIP",
        },
        sources: dedupeSources(sources),
        warnings: ["Cite DIP as source. For BT plenary protocols use BT-PlPr. plus document number."],
        nextActions: ["dipctl " + entity + " get --id <id>"],
    });
}
async function runDocumentText(kind, args) {
    const { flags } = parseArgs(args);
    const key = mustKey(flags);
    const entity = kind + "-text";
    const { record, request } = await resolveRecord(entity, flags.id, flags["document-number"], key);
    const text = String(record.text ?? "");
    const term = flags.grep ?? "";
    const context = positiveInt(flags.context ?? "220", "context");
    const sources = extractSources(record);
    const id = String(record.id ?? "");
    if (id)
        sources.push({ title: "DIP API " + entity + " detail", url: BASE_URL + "/" + entity + "/" + id, kind: "api" });
    const out = {
        status: "ok",
        tool: APP_NAME,
        command: kind + " text",
        retrievedAt: now(),
        request,
        summary: { record: compactItem(record), textLength: text.length, grep: term, snippetCount: 0 },
        sources: dedupeSources(sources),
        warnings: ["Full text is official DIP text where available.", "Use source attribution: Deutscher Bundestag/Bundesrat - DIP."],
        nextActions: ["dipctl source --type " + kind + " --id " + id],
    };
    if (term) {
        const snips = snippets(text, term, context);
        out.summary.snippetCount = snips.length;
        out.snippets = snips;
    }
    else {
        out.textPreview = preview(text, 1800);
    }
    writeJson(out);
}
async function runPlenarySpeechSearch(args) {
    const { flags } = parseArgs(args);
    const term = flags.term;
    if (!term)
        throw new CliError(2, "invalid_arguments", "missing required flag --term");
    if (flags["document-number"] || flags.id) {
        await runDocumentText("plenarprotokoll", [...args, "--grep", term]);
        return;
    }
    const key = mustKey(flags);
    const limit = positiveInt(flags.limit ?? "10", "limit");
    const params = { format: ["json"] };
    if (flags["person-id"])
        params["f.person_id"] = [flags["person-id"]];
    else if (flags.person)
        params["f.person"] = [flags.person];
    else
        throw new CliError(2, "invalid_arguments", "pass --document-number, --person-id, or --person");
    const { body, requestUrl } = await apiGet("aktivitaet", params, key);
    const matches = [];
    for (const doc of takeDocuments(JSON.parse(body), 100)) {
        if (JSON.stringify(doc).toLowerCase().includes(term.toLowerCase())) {
            matches.push(compactItem(doc));
            if (matches.length >= limit)
                break;
        }
    }
    writeJson({
        status: "ok",
        tool: APP_NAME,
        command: "plenary speech search",
        retrievedAt: now(),
        request: requestMeta(requestUrl),
        summary: { mode: "aktivitaet-search", term, returned: matches.length, clientLimit: limit },
        items: matches,
        sources: [{ title: "DIP API activity endpoint", url: BASE_URL + "/aktivitaet", kind: "api" }],
        warnings: ["Activity search is official DIP metadata, not a full transcript search."],
        nextActions: [`dipctl plenarprotokoll text --document-number <number> --grep "${term}"`],
    });
}
async function resolveRecord(entity, id, documentNumber, key) {
    if (id) {
        const resp = await apiGet(`${entity}/${encodeURIComponent(id)}`, { format: ["json"] }, key);
        return { record: JSON.parse(resp.body), request: requestMeta(resp.requestUrl) };
    }
    if (!documentNumber)
        throw new CliError(2, "invalid_arguments", "pass --id or --document-number");
    const resp = await apiGet(entity, { "f.dokumentnummer": [documentNumber], format: ["json"] }, key);
    const docs = takeDocuments(JSON.parse(resp.body), 1);
    if (docs.length === 0)
        throw new CliError(1, "not_found", "no record found for document number");
    return { record: docs[0], request: requestMeta(resp.requestUrl) };
}
async function apiGet(path, params, key) {
    const u = new URL(BASE_URL + "/" + path.replace(/^\/+/, ""));
    for (const [paramKey, values] of Object.entries(params)) {
        for (const value of values)
            u.searchParams.append(paramKey, value);
    }
    const requestUrl = u.toString();
    const { statusCode, body } = await httpsGet(requestUrl, {
        Accept: "application/json",
        Authorization: "ApiKey " + key,
    });
    if (statusCode < 200 || statusCode >= 300) {
        throw new CliError(1, "request_failed", `DIP API returned HTTP ${statusCode}: ${preview(body, 500)}`);
    }
    return { body, requestUrl };
}
function httpsGet(requestUrl, headers) {
    return new Promise((resolve, reject) => {
        const req = request(requestUrl, { method: "GET", headers, timeout: 30000 }, (res) => {
            const chunks = [];
            res.on("data", (chunk) => chunks.push(chunk));
            res.on("end", () => {
                resolve({
                    statusCode: res.statusCode ?? 0,
                    body: Buffer.concat(chunks).toString("utf8"),
                });
            });
        });
        req.on("timeout", () => {
            req.destroy(new Error("request timed out"));
        });
        req.on("error", reject);
        req.end();
    }).catch((err) => {
        const message = err instanceof Error ? err.message : String(err);
        throw new CliError(1, "request_failed", message);
    });
}
function parseArgs(args) {
    const flags = {};
    const params = {};
    for (let i = 0; i < args.length; i += 1) {
        const arg = args[i];
        if (!arg.startsWith("--"))
            throw new CliError(2, "invalid_arguments", "unexpected positional argument: " + arg);
        let nameValue = arg.slice(2);
        let name = nameValue;
        let value = "";
        if (nameValue.includes("=")) {
            [name, value] = nameValue.split(/=(.*)/s, 2);
        }
        else if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
            i += 1;
            value = args[i];
        }
        else {
            value = "true";
        }
        if (name === "param") {
            const cut = value.indexOf("=");
            if (cut < 1)
                throw new CliError(2, "invalid_arguments", "--param must be key=value");
            const key = value.slice(0, cut);
            const val = value.slice(cut + 1);
            if (!params[key])
                params[key] = [];
            params[key].push(val);
        }
        else {
            flags[name] = value;
        }
    }
    return { flags, params };
}
function resolveKey(flags) {
    if (flags.apikey)
        return [flags.apikey, "flag"];
    if (process.env.DIP_API_KEY)
        return [process.env.DIP_API_KEY, "env:DIP_API_KEY"];
    return ["", "missing"];
}
function mustKey(flags) {
    const [key] = resolveKey(flags);
    if (!key)
        throw new CliError(2, "missing_api_key", "set DIP_API_KEY or pass --apikey");
    return key;
}
function compactItem(doc) {
    const keys = ["id", "typ", "dokumentart", "vorgangstyp", "titel", "dokumentnummer", "wahlperiode", "herausgeber", "datum", "aktualisiert", "person_id"];
    const out = {};
    for (const key of keys)
        if (doc[key] !== undefined)
            out[key] = doc[key];
    if (doc.titel)
        out.title = doc.titel;
    const sources = extractSources(doc);
    if (sources.length > 0)
        out.sources = sources;
    return out;
}
function extractSources(value) {
    const out = [];
    const walk = (node, key = "") => {
        if (Array.isArray(node)) {
            for (const item of node)
                walk(item, key);
        }
        else if (node && typeof node === "object") {
            for (const [childKey, childValue] of Object.entries(node))
                walk(childValue, childKey);
        }
        else if (typeof node === "string" && (node.startsWith("https://") || node.startsWith("http://"))) {
            const lower = key.toLowerCase();
            let kind = "url";
            if (lower.includes("pdf"))
                kind = "pdf";
            else if (lower.includes("xml"))
                kind = "xml";
            else if (lower.includes("api"))
                kind = "api";
            out.push({ title: key, url: node, kind });
        }
    };
    walk(value);
    return dedupeSources(out);
}
function dedupeSources(sources) {
    const seen = new Set();
    const out = [];
    for (const source of sources) {
        const url = String(source.url ?? "");
        if (url && !seen.has(url)) {
            seen.add(url);
            out.push(source);
        }
    }
    return out.sort((a, b) => String(a.url).localeCompare(String(b.url)));
}
function snippets(text, term, context) {
    const out = [];
    const lower = text.toLowerCase();
    const needle = term.toLowerCase();
    let from = 0;
    while (out.length < 10) {
        const idx = lower.indexOf(needle, from);
        if (idx < 0)
            break;
        const end = idx + term.length;
        const start = Math.max(0, idx - context);
        const stop = Math.min(text.length, end + context);
        out.push({ start: idx, end, snippet: clean(text.slice(start, stop)) });
        from = end;
    }
    return out;
}
function takeDocuments(data, limit) {
    if (!Array.isArray(data.documents))
        return [];
    return data.documents.slice(0, limit).filter((item) => item && typeof item === "object");
}
function requestMeta(requestUrl) {
    return { method: "GET", url: redactUrl(requestUrl), redactions: ["Authorization", "apikey"] };
}
function redactUrl(requestUrl) {
    const u = new URL(requestUrl);
    if (u.searchParams.has("apikey"))
        u.searchParams.set("apikey", "REDACTED");
    return u.toString();
}
function writeJson(value) {
    console.log(JSON.stringify(value, null, 2));
}
function fail(exitCode, code, message) {
    console.error(JSON.stringify({
        status: "error",
        tool: APP_NAME,
        retrievedAt: now(),
        error: { code, message },
    }, null, 2));
    process.exit(exitCode);
}
function positiveInt(raw, name) {
    const value = Number.parseInt(raw, 10);
    if (!Number.isFinite(value) || value < 1)
        throw new CliError(2, "invalid_arguments", `--${name} must be a positive integer`);
    return value;
}
function docSources() {
    return [
        { title: "DIP API help", url: "https://dip.bundestag.de/%C3%BCber-dip/hilfe/api", kind: "documentation" },
        { title: "DIP short documentation PDF", url: "https://dip.bundestag.de/documents/informationsblatt_zur_dip_api.pdf", kind: "documentation" },
        { title: "DIP terms PDF", url: "https://dip.bundestag.de/documents/nutzungsbedingungen_dip.pdf", kind: "terms" },
        { title: "DIP OpenAPI YAML", url: "https://search.dip.bundestag.de/api/v1/openapi.yaml", kind: "openapi" },
    ];
}
function preview(text, maxLen) {
    const cleaned = clean(text);
    return cleaned.length <= maxLen ? cleaned : cleaned.slice(0, maxLen) + "...";
}
function clean(text) {
    return String(text).split(/\s+/u).filter(Boolean).join(" ");
}
function now() {
    return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}
function isHelp(arg) {
    return arg === "--help" || arg === "-h" || arg === "help";
}
main(process.argv.slice(2)).then((code) => process.exit(code));
