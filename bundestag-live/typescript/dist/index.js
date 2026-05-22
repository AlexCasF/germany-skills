const APP_NAME = "bundestagctl";
const BASE_URL = "https://www.bundestag.de";
const SPEAKER_URL = `${BASE_URL}/static/appdata/plenum/v2/speaker.xml`;
const CONFERENCES_URL = `${BASE_URL}/static/appdata/plenum/v2/conferences.xml`;
const COMMITTEES_URL = `${BASE_URL}/xml/v2/ausschuesse/index.xml`;
const COMMITTEE_URL = `${BASE_URL}/xml/v2/ausschuesse/{id}.xml`;
const MEMBERS_URL = `${BASE_URL}/xml/v2/mdb/index.xml`;
const MEMBER_URL = `${BASE_URL}/xml/v2/mdb/biografien/{id}.xml`;
const ARTICLE_URL = `${BASE_URL}/blueprint/servlet/content/{id}/asAppV2NewsarticleXml`;
const VIDEO_URL = "http://webtv.bundestag.de/iptv/player/macros/_x_s-144277506/bttv/mobile/feed_vod.xml";
const OPENAPI_URL = "https://github.com/bundesAPI/bundestag-api";
const OPEN_DATA_URL = `${BASE_URL}/services/opendata`;
const IMPRINT_URL = `${BASE_URL}/services/impressum`;
const MEDIA_TERMS_URL = `${BASE_URL}/mediathek/nutzungsbedingungen-247892`;
const PRIVACY_URL = `${BASE_URL}/en/service/privacy`;
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
    if (!argv.length || isHelp(argv[0])) {
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
        else if (argv[0] === "examples")
            printExamples();
        else if (matches(argv, "plenum", "speaker"))
            await runPlenumSpeaker(argv.slice(2));
        else if (matches(argv, "plenum", "conferences") || matches(argv, "plenum", "agenda"))
            await runPlenumConferences(argv.slice(2));
        else if (matches(argv, "members", "list"))
            await runMembersList(argv.slice(2));
        else if (matches(argv, "members", "search"))
            await runMembersSearch(argv.slice(2));
        else if (matches(argv, "members", "biography"))
            await runMemberBiography(argv.slice(2));
        else if (matches(argv, "members", "dossier"))
            await runMemberDossier(argv.slice(2));
        else if (matches(argv, "committees", "list"))
            await runCommitteesList(argv.slice(2));
        else if (matches(argv, "committees", "search"))
            await runCommitteesSearch(argv.slice(2));
        else if (matches(argv, "committees", "get") || matches(argv, "committees", "dossier"))
            await runCommitteeDossier(argv.slice(2));
        else if (matches(argv, "article", "get"))
            await runArticleGet(argv.slice(2));
        else if (matches(argv, "article", "page"))
            await runArticlePage(argv.slice(2));
        else if (matches(argv, "video", "feed"))
            await runVideoFeed(argv.slice(2));
        else if (argv[0] === "source")
            runSource(argv.slice(1));
        else
            throw new CLIError(2, "unknown_command", "unknown command; run bundestagctl --help");
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
    console.log(`bundestagctl -- Bundestag live/site XML research CLI

Purpose
  Discover and normalize public Bundestag live/site XML feeds for current
  plenary agenda data, members, biographies, committees, articles, and video
  feed metadata.

Fast paths
  bundestagctl doctor
  bundestagctl members search --name "Amthor" --limit 3
  bundestagctl members dossier --name "Amthor" --grep "TÃ¤tigkeiten"
  bundestagctl committees search --term "Arbeit" --limit 5
  bundestagctl committees dossier --id a11 --member-limit 5
  bundestagctl plenum conferences --limit 2 --item-limit 3
  bundestagctl article get --article-id 1174778

Endpoint-compatible commands
  plenum speaker
  plenum conferences
  committees list
  committees get --id a11
  members list
  members biography --id 2022
  article get --article-id 1174778
  video feed --content-id 7529016
`);
}
function printHelp(path) {
    const joined = path.join(" ");
    if (joined === "members search")
        console.log('bundestagctl members search --name "Amthor" --limit 3');
    else if (joined === "members dossier")
        console.log('bundestagctl members dossier --id 2022 --grep "TÃ¤tigkeiten"');
    else if (joined === "committees dossier")
        console.log("bundestagctl committees dossier --id a11 --member-limit 5 --news-limit 3");
    else if (joined === "article page")
        console.log('bundestagctl article page --url "https://www.bundestag.de/..." --grep "term"');
    else
        printRootHelp();
}
function printExamples() {
    console.log(`bundestagctl examples

1. bundestagctl doctor
2. bundestagctl members search --name "Amthor" --limit 3
3. bundestagctl members dossier --id 2022 --grep "TÃ¤tigkeiten"
4. bundestagctl committees search --term "Arbeit" --limit 5
5. bundestagctl committees dossier --id a11 --member-limit 5 --news-limit 3
6. bundestagctl plenum conferences --limit 2 --item-limit 5
7. bundestagctl article get --article-id 1174778
8. bundestagctl article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778" --grep "Meinungsfreiheit"
9. bundestagctl members biography --id 2022 --raw
10. Use dipctl for full parliamentary proceedings and historical protocol research.
`);
}
async function runDoctor(argv) {
    const parsed = parseArgs(argv);
    const limit = limitFlag(parsed, 3, 10);
    const checks = [["speaker", SPEAKER_URL], ["conferences", CONFERENCES_URL], ["committees", COMMITTEES_URL], ["members", MEMBERS_URL]].slice(0, limit);
    const summary = {
        authRequired: false,
        publishedRateLimit: "No exact published request quota was found for these public Bundestag XML feeds. Use small limits, cache repeated index calls, and back off on 429/5xx responses.",
        fairUseHints: ["Use search commands before fetching detail records.", "Avoid repeated full member index downloads during one run.", "Use --limit and --item-limit on broad feeds.", "Treat video/media URLs under Bundestag media terms."],
        endpoints: []
    };
    let status = "ok";
    for (const [name, url] of checks) {
        const raw = await fetchRaw(url);
        const ok = raw.status >= 200 && raw.status < 300;
        if (!ok)
            status = "degraded";
        summary.endpoints.push({ name, url, statusCode: raw.status, contentType: raw.contentType, ok, bodyPreview: truncate(stripSpace(raw.body), 180) });
    }
    const payload = envelope("doctor", BASE_URL, { limit });
    payload.status = status;
    payload.summary = summary;
    payload.sources = defaultSources();
    payload.warnings = defaultWarnings();
    payload.nextActions = ['bundestagctl members search --name "Amthor" --limit 3', "bundestagctl committees search --term Arbeit --limit 5"];
    emit(payload);
}
async function runMembersList(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchXmlWithParams(MEMBERS_URL, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const items = parseMembers(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const payload = envelope("members list", requestUrl, { limit });
    payload.summary = { totalMembers: items.length, returned: Math.min(limit, items.length), documentStand: tag(body, "dokumentStand") };
    payload.items = items.slice(0, limit).map((item) => compactMember(item, flagBool(parsed, "include-raw")));
    payload.sources = source("Bundestag member XML index", MEMBERS_URL, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = ['bundestagctl members search --name "Amthor" --limit 3'];
    emit(payload);
}
async function runMembersSearch(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.name, parsed.flags.term, parsed.flags.q, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", "members search requires --name or --term");
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const { body, requestUrl } = await fetchXmlWithParams(MEMBERS_URL, {});
    const items = parseMembers(body);
    const found = items.filter((item) => memberSearchText(item).toLowerCase().includes(term.toLowerCase()));
    const payload = envelope("members search", requestUrl, { term, limit });
    payload.summary = { term, matches: found.length, returned: Math.min(limit, found.length), searchedMembers: items.length, documentStand: tag(body, "dokumentStand") };
    payload.items = found.slice(0, limit).map((item) => compactMember(item, flagBool(parsed, "include-raw")));
    payload.sources = source("Bundestag member XML index", MEMBERS_URL, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = found.slice(0, 3).map((item) => `bundestagctl members dossier --id ${item.id}`);
    emit(payload);
}
async function runMemberBiography(argv) {
    const parsed = parseArgs(argv);
    const id = firstNonEmpty(parsed.flags.id, parsed.positionals[0]);
    if (!id)
        throw new CLIError(2, "missing_id", "members biography requires --id");
    const { body, requestUrl } = await fetchXmlWithParams(MEMBER_URL.replace("{id}", encodeURIComponent(id)), parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const grep = parsed.flags.grep ?? "";
    const payload = envelope("members biography", requestUrl, { id });
    payload.summary = compactBiography(body, grep);
    payload.items = [memberEvidence(body, grep)];
    payload.sources = sourcesForMember(body, requestUrl);
    payload.warnings = defaultWarnings();
    payload.nextActions = [`bundestagctl members dossier --id ${id} --grep TÃ¤tigkeiten`];
    if (flagBool(parsed, "include-raw"))
        payload.rawXml = body;
    emit(payload);
}
async function runMemberDossier(argv) {
    const parsed = parseArgs(argv);
    let id = parsed.flags.id ?? "";
    let resolved = null;
    if (!id) {
        const name = firstNonEmpty(parsed.flags.name, parsed.flags.term, parsed.positionals.join(" "));
        if (!name)
            throw new CLIError(2, "missing_member", "members dossier requires --id or --name");
        resolved = await resolveMember(name);
        id = resolved.id;
    }
    const { body, requestUrl } = await fetchXmlWithParams(MEMBER_URL.replace("{id}", encodeURIComponent(id)), {});
    const grep = parsed.flags.grep ?? "";
    const payload = envelope("members dossier", requestUrl, { id, name: parsed.flags.name ?? "", grep });
    payload.summary = compactBiography(body, grep);
    payload.items = [memberEvidence(body, grep)];
    payload.sources = sourcesForMember(body, requestUrl);
    payload.warnings = [...defaultWarnings(), "Member biography and disclosure fields are based on Bundestag profile XML; disclosure text may reflect self-reported data and Bundestag publication rules."];
    payload.nextActions = [`bundestagctl members biography --id ${id} --raw`];
    if (resolved)
        payload.resolvedFromIndex = compactMember(resolved, false);
    if (flagBool(parsed, "include-raw"))
        payload.rawXml = body;
    emit(payload);
}
async function runCommitteesList(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchXmlWithParams(COMMITTEES_URL, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const items = parseCommittees(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const payload = envelope("committees list", requestUrl, { limit });
    payload.summary = { totalCommittees: items.length, returned: Math.min(limit, items.length), documentStand: tag(body, "dokumentStand") };
    payload.items = items.slice(0, limit).map((item) => compactCommittee(item, flagBool(parsed, "include-raw")));
    payload.sources = source("Bundestag committee XML index", COMMITTEES_URL, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = ["bundestagctl committees search --term Arbeit --limit 5"];
    emit(payload);
}
async function runCommitteesSearch(argv) {
    const parsed = parseArgs(argv);
    const term = firstNonEmpty(parsed.flags.term, parsed.flags.q, parsed.flags.name, parsed.positionals.join(" "));
    if (!term)
        throw new CLIError(2, "missing_term", "committees search requires --term");
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const { body, requestUrl } = await fetchXmlWithParams(COMMITTEES_URL, {});
    const items = parseCommittees(body);
    const found = items.filter((item) => committeeSearchText(item).toLowerCase().includes(term.toLowerCase()));
    const payload = envelope("committees search", requestUrl, { term, limit });
    payload.summary = { term, matches: found.length, returned: Math.min(limit, found.length), searchedCommittees: items.length };
    payload.items = found.slice(0, limit).map((item) => compactCommittee(item, flagBool(parsed, "include-raw")));
    payload.sources = source("Bundestag committee XML index", COMMITTEES_URL, "api_endpoint");
    payload.warnings = defaultWarnings();
    payload.nextActions = found.slice(0, 3).map((item) => `bundestagctl committees dossier --id ${item.id} --member-limit 5`);
    emit(payload);
}
async function runCommitteeDossier(argv) {
    const parsed = parseArgs(argv);
    const id = firstNonEmpty(parsed.flags.id, parsed.positionals[0]);
    if (!id)
        throw new CLIError(2, "missing_id", "committees dossier requires --id");
    const { body, requestUrl } = await fetchXmlWithParams(COMMITTEE_URL.replace("{id}", encodeURIComponent(id)), parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const memberLimit = limitFlagName(parsed, "member-limit", DEFAULT_LIMIT, SAFE_LIMIT);
    const newsLimit = limitFlagName(parsed, "news-limit", 5, 50);
    const grep = parsed.flags.grep ?? "";
    const members = parseCommitteeMembers(body);
    const news = parseCommitteeNews(body, grep);
    const payload = envelope("committees get", requestUrl, { id, memberLimit, newsLimit });
    payload.summary = { id: tag(body, "ausschussId"), name: tag(body, "ausschussName"), sourceUrl: tag(body, "ausschussSourceURL"), chairId: tag(body, "ausschussVorsitzId"), memberCount: members.length, newsCount: news.length, taskSnippets: grepSnippets(stripHtml(tag(body, "ausschussAufgabe")), grep, 3, 650), contact: stripHtml(tag(body, "ausschussKontakt")), membersShown: Math.min(memberLimit, members.length), newsShown: Math.min(newsLimit, news.length) };
    payload.items = [{ task: truncate(stripHtml(tag(body, "ausschussAufgabe")), 1200), contact: stripHtml(tag(body, "ausschussKontakt")), members: members.slice(0, memberLimit), news: news.slice(0, newsLimit) }];
    payload.sources = [{ title: "Bundestag committee detail XML", url: requestUrl, kind: "api_endpoint" }];
    if (tag(body, "ausschussSourceURL"))
        payload.sources.push({ title: "Bundestag committee page", url: tag(body, "ausschussSourceURL"), kind: "public_page" });
    payload.warnings = defaultWarnings();
    payload.nextActions = [`bundestagctl committees dossier --id ${id} --member-limit 5`];
    emit(payload);
}
async function runPlenumSpeaker(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchXmlWithParams(SPEAKER_URL, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const speakers = blocks(body, "speaker").map((block) => ({ firstName: tag(block, "firstName"), lastName: tag(block, "lastName"), name: tag(block, "name"), fraction: tag(block, "fraction"), party: tag(block, "party"), id: tag(block, "id") }));
    const payload = envelope("plenum speaker", requestUrl, null);
    payload.summary = { live: tag(body, "live"), topicNumber: tag(body, "topicNumber"), speakerCount: speakers.length };
    payload.items = speakers;
    payload.sources = source("Bundestag current speaker XML", SPEAKER_URL, "api_endpoint");
    payload.warnings = [...defaultWarnings(), "The current speaker feed can be empty when no plenary sitting is live."];
    payload.nextActions = ["bundestagctl plenum conferences --limit 2 --item-limit 5"];
    emit(payload);
}
async function runPlenumConferences(argv) {
    const parsed = parseArgs(argv);
    const { body, requestUrl } = await fetchXmlWithParams(CONFERENCES_URL, parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const limit = limitFlag(parsed, DEFAULT_LIMIT, SAFE_LIMIT);
    const itemLimit = limitFlagName(parsed, "item-limit", DEFAULT_LIMIT, SAFE_LIMIT);
    const dayBlocks = blocks(body, "tagesordnung");
    const nextActions = [];
    const days = dayBlocks.slice(0, limit).map((day) => {
        const itemBlocks = blocks(day, "diskussionspunkt");
        const items = itemBlocks.slice(0, itemLimit).map((item) => {
            const articleId = tag(item, "articleId");
            if (articleId && nextActions.length < 3)
                nextActions.push(`bundestagctl article get --article-id ${articleId}`);
            return { startTime: tag(item, "startzeit"), endTime: tag(item, "endzeit"), status: tag(item, "status"), title: tag(item, "titel"), articleId, top: tag(item, "top"), nextActions: articleId ? [`bundestagctl article get --article-id ${articleId}`] : [] };
        });
        return { date: tag(day, "date"), active: tag(day, "active"), sessionNumber: tag(day, "sitzungsnummer"), name: tag(day, "name"), itemCount: itemBlocks.length, items };
    });
    const payload = envelope("plenum conferences", requestUrl, { limit, itemLimit });
    payload.summary = { totalDays: dayBlocks.length, returned: days.length };
    payload.items = days;
    payload.sources = source("Bundestag plenary conference XML", CONFERENCES_URL, "api_endpoint");
    payload.warnings = [...defaultWarnings(), "Agenda article IDs point to Bundestag article XML/page records, not full plenary protocols."];
    payload.nextActions = nextActions.length ? nextActions : ["bundestagctl plenum speaker"];
    emit(payload);
}
async function runArticleGet(argv) {
    const parsed = parseArgs(argv);
    let id = firstNonEmpty(parsed.flags["article-id"], parsed.flags.id, parsed.positionals[0]);
    if (!id && parsed.flags.url)
        id = articleIdFromUrl(parsed.flags.url);
    if (!id)
        throw new CLIError(2, "missing_article_id", "article get requires --article-id or --url");
    const { body, requestUrl } = await fetchXmlWithParams(ARTICLE_URL.replace("{id}", encodeURIComponent(id)), parsed.params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const grep = parsed.flags.grep ?? "";
    const payload = envelope("article get", requestUrl, { articleId: id, grep });
    payload.summary = compactArticle(body, grep);
    payload.items = [articleEvidence(body, grep)];
    payload.sources = [{ title: "Bundestag article XML", url: requestUrl, kind: "api_endpoint" }];
    if (tag(body, "sourceURL")) {
        payload.sources.push({ title: "Bundestag public article page", url: tag(body, "sourceURL"), kind: "public_page" });
        payload.nextActions = [`bundestagctl article page --url "${tag(body, "sourceURL")}"`];
    }
    payload.warnings = defaultWarnings();
    emit(payload);
}
async function runArticlePage(argv) {
    const parsed = parseArgs(argv);
    let sourceUrl = parsed.flags.url ?? "";
    const id = firstNonEmpty(parsed.flags["article-id"], parsed.flags.id);
    if (!sourceUrl && id) {
        const { body } = await fetchXmlWithParams(ARTICLE_URL.replace("{id}", encodeURIComponent(id)), {});
        sourceUrl = tag(body, "sourceURL");
    }
    if (!sourceUrl)
        throw new CLIError(2, "missing_url", "article page requires --url or --article-id");
    if (!sourceUrl.startsWith(BASE_URL))
        throw new CLIError(2, "unsafe_url", "article page only accepts www.bundestag.de URLs");
    const raw = await fetchRaw(sourceUrl);
    const text = stripHtml(raw.body);
    const grep = parsed.flags.grep ?? "";
    const snippets = grepSnippets(text, grep, 5, 650);
    const payload = envelope("article page", sourceUrl, { url: sourceUrl, grep });
    payload.summary = { url: sourceUrl, statusCode: raw.status, contentType: raw.contentType, title: htmlTitle(raw.body), textLength: text.length, snippetCount: snippets.length };
    payload.items = snippets;
    payload.sources = source("Bundestag public article page", sourceUrl, "public_page");
    payload.warnings = [...defaultWarnings(), "Public HTML page extraction is best-effort; use article get for structured XML metadata when possible."];
    const articleId = articleIdFromUrl(sourceUrl);
    payload.nextActions = articleId ? [`bundestagctl article get --article-id ${articleId}`] : [];
    emit(payload);
}
async function runVideoFeed(argv) {
    const parsed = parseArgs(argv);
    const contentId = firstNonEmpty(parsed.flags["content-id"], parsed.flags.contentid, parsed.params.contentId, parsed.params.contentid);
    const params = { ...parsed.params };
    if (contentId && !params.contentId)
        params.contentId = contentId;
    const { body, requestUrl } = await fetchXmlWithParams(VIDEO_URL, params);
    if (flagBool(parsed, "raw"))
        return printRaw(body);
    const groups = [...body.matchAll(/<group\s+type="([^"]*)"[^>]*>([\s\S]*?)<\/group>/g)].map((match) => ({ type: match[1], streams: [...match[2].matchAll(/<stream\s+([^>]*?)\/>/g)].map((stream) => ({ bandwidth: attr(stream[1], "bandwidth"), href: attr(stream[1], "href") })) }));
    const payload = envelope("video feed", requestUrl, { contentId });
    payload.summary = { contentId, groups: groups.length, streamCount: groups.reduce((sum, group) => sum + group.streams.length, 0) };
    payload.items = groups;
    payload.sources = [{ title: "Bundestag WebTV feed", url: requestUrl, kind: "api_endpoint" }, { title: "Bundestag audio/video terms", url: MEDIA_TERMS_URL, kind: "terms" }];
    payload.warnings = [...defaultWarnings(), "Video/audio material is governed by Bundestag media terms; cite Deutscher Bundestag and avoid misleading edits."];
    payload.nextActions = ["bundestagctl plenum conferences --limit 2 --item-limit 5"];
    emit(payload);
}
function runSource(argv) {
    const parsed = parseArgs(argv);
    const sourceUrl = firstNonEmpty(parsed.flags.url, parsed.positionals[0]);
    if (!sourceUrl)
        throw new CLIError(2, "missing_url", "source requires --url");
    const payload = envelope("source", sourceUrl, { url: sourceUrl });
    payload.summary = { url: sourceUrl, kind: sourceKind(sourceUrl), citation: `Deutscher Bundestag, ${sourceUrl}` };
    payload.sources = source("Bundestag source", sourceUrl, sourceKind(sourceUrl));
    payload.warnings = defaultWarnings();
    const articleId = articleIdFromUrl(sourceUrl);
    payload.nextActions = articleId ? [`bundestagctl article get --article-id ${articleId}`] : [];
    emit(payload);
}
function parseMembers(xml) {
    return blocks(xml, "mdb").map((block) => {
        const fraction = attr(block.match(/^<mdb([^>]*)>/)?.[1] ?? "", "fraktion");
        return { id: tag(block, "mdbID"), status: firstNonEmpty(tagAttr(block, "mdbID", "status"), tagAttr(block, "mdbName", "status")), name: tag(block, "mdbName"), fraction, state: tag(block, "mdbLand"), constituency: { number: tag(block, "mdbWahlkreisNummer"), name: tag(block, "mdbWahlkreisName"), url: "" }, electionType: tag(block, "mdbGewaehlt"), bioUrl: tag(block, "mdbBioURL"), infoXmlUrl: tag(block, "mdbInfoXMLURL"), lastChanged: tag(block, "lastChanged"), raw: block };
    }).filter((item) => item.id && item.name);
}
function compactMember(item, includeRaw) {
    const out = { id: item.id, status: item.status, name: item.name, fraction: item.fraction, state: item.state, constituency: item.constituency, electionType: item.electionType, bioUrl: item.bioUrl, infoXmlUrl: item.infoXmlUrl, lastChanged: item.lastChanged };
    out.sources = [{ title: "Bundestag member profile", url: item.bioUrl, kind: "public_profile" }, { title: "Bundestag member biography XML", url: item.infoXmlUrl, kind: "api_endpoint" }];
    out.nextActions = [`bundestagctl members dossier --id ${item.id}`, `bundestagctl members biography --id ${item.id} --raw`];
    if (includeRaw)
        out.raw = item.raw;
    return out;
}
function memberSearchText(item) {
    return [item.id, item.name, item.fraction, item.state, item.electionType, item.bioUrl, JSON.stringify(item.constituency)].join(" ");
}
async function resolveMember(term) {
    const { body } = await fetchXmlWithParams(MEMBERS_URL, {});
    const found = parseMembers(body).find((item) => String(item.name).toLowerCase().includes(term.toLowerCase()));
    if (!found)
        throw new CLIError(2, "member_not_found", `member not found: ${term}`);
    return found;
}
function compactBiography(xml, grep) {
    const info = block(xml, "mdbInfo") || xml;
    const media = block(xml, "mdbMedien");
    const bioText = stripHtml(tag(info, "mdbBiografischeInformationen"));
    const disclosureText = stripHtml(tag(info, "mdbVeroeffentlichungspflichtigeAngaben"));
    return { id: tag(info, "mdbID"), status: tagAttr(info, "mdbID", "status"), name: stripSpace([tag(info, "mdbAkademischerTitel"), tag(info, "mdbVorname"), tag(info, "mdbZuname")].join(" ")), party: tag(info, "mdbPartei"), fraction: tag(info, "mdbFraktion"), state: tag(info, "mdbLand"), profession: tag(info, "mdbBeruf"), birthDate: tag(info, "mdbGeburtsdatum"), constituency: { number: tag(info, "mdbWahlkreisNummer"), name: tag(info, "mdbWahlkreisName"), url: tag(info, "mdbWahlkreisURL") }, electionType: tag(info, "mdbGewaehlt"), profileUrl: firstNonEmpty(tag(info, "sourceURL"), tag(info, "mdbBioURL")), homepageUrl: tag(info, "mdbHomepageURL"), speechesUrl: tag(media, "mdbRedenVorPlenumURL"), speechesRss: tag(media, "mdbRedenVorPlenumRSS"), biographySnippets: grepSnippets(bioText, grep, 3, 650), disclosureSnippets: grepSnippets(disclosureText, grep, 5, 650) };
}
function memberEvidence(xml, grep) {
    const info = block(xml, "mdbInfo") || xml;
    const media = block(xml, "mdbMedien");
    const photo = block(media, "mdbFoto");
    const websites = blocks(info, "mdbSonstigeWebsite").map((site) => ({ title: tag(site, "mdbSonstigeWebsiteTitel"), url: tag(site, "mdbSonstigeWebsiteURL") }));
    return { biography: truncate(stripHtml(tag(info, "mdbBiografischeInformationen")), 1500), disclosures: grepSnippets(stripHtml(tag(info, "mdbVeroeffentlichungspflichtigeAngaben")), grep, 8, 650), websites, media: { photoUrl: tag(photo, "mdbFotoURL"), photoSource: tag(photo, "mdbFotoCopyright"), speechesUrl: tag(media, "mdbRedenVorPlenumURL"), speechesRss: tag(media, "mdbRedenVorPlenumRSS") } };
}
function sourcesForMember(xml, requestUrl) {
    const info = block(xml, "mdbInfo") || xml;
    const media = block(xml, "mdbMedien");
    const out = [{ title: "Bundestag member biography XML", url: requestUrl, kind: "api_endpoint" }];
    const profile = firstNonEmpty(tag(info, "sourceURL"), tag(info, "mdbBioURL"));
    if (profile)
        out.push({ title: "Bundestag member profile", url: profile, kind: "public_profile" });
    if (tag(media, "mdbRedenVorPlenumURL"))
        out.push({ title: "Bundestag mediathek speeches filter", url: tag(media, "mdbRedenVorPlenumURL"), kind: "media_search" });
    if (tag(media, "mdbRedenVorPlenumRSS"))
        out.push({ title: "Bundestag speeches RSS", url: tag(media, "mdbRedenVorPlenumRSS"), kind: "rss" });
    return out;
}
function parseCommittees(xml) {
    const outer = block(xml, "ausschuesse");
    return [...outer.matchAll(/<ausschuss\s+id="([^"]*)"[^>]*>([\s\S]*?)<\/ausschuss>/g)].map((match) => ({ id: match[1], name: tag(match[2], "ausschussName"), shortName: tag(match[2], "ausschussKurzName"), teaser: stripHtml(tag(match[2], "ausschussTeaser")), detailXmlUrl: tag(match[2], "ausschussDetailXML"), imageUrl: tag(match[2], "imageURL"), imageSource: tag(match[2], "imageCopyright"), lastChanged: tag(match[2], "lastChanged"), raw: match[0] }));
}
function compactCommittee(item, includeRaw) {
    const out = { id: item.id, name: item.name, shortName: item.shortName, teaser: item.teaser, detailXmlUrl: item.detailXmlUrl, imageUrl: item.imageUrl, imageSource: item.imageSource, lastChanged: item.lastChanged };
    out.sources = source("Bundestag committee XML", item.detailXmlUrl, "api_endpoint");
    out.nextActions = [`bundestagctl committees dossier --id ${item.id} --member-limit 5`];
    if (includeRaw)
        out.raw = item.raw;
    return out;
}
function committeeSearchText(item) {
    return [item.id, item.name, item.shortName, item.teaser, item.detailXmlUrl].join(" ");
}
function parseCommitteeMembers(xml) {
    const outer = block(xml, "ausschussMitglieder");
    return blocks(outer, "mdb").map((item) => { const id = tag(item, "mdbID"); return { id, name: tag(item, "mdbName"), fraction: attr(item.match(/^<mdb([^>]*)>/)?.[1] ?? "", "fraktion"), state: tag(item, "mdbLand"), role: tag(item, "role"), bioUrl: tag(item, "mdbBioURL"), infoXmlUrl: tag(item, "mdbInfoXMLURL"), lastChanged: tag(item, "lastChanged"), nextActions: [`bundestagctl members dossier --id ${id}`] }; });
}
function parseCommitteeNews(xml, grep) {
    const outer = block(xml, "newslist");
    return [...outer.matchAll(/<news\s+articleId="([^"]*)"[^>]*>([\s\S]*?)<\/news>/g)].map((match) => {
        const articleId = match[1];
        const item = match[2];
        const text = stripHtml(tag(item, "teaser"));
        if (grep && !`${text} ${tag(item, "title")}`.toLowerCase().includes(grep.toLowerCase()))
            return null;
        return { articleId, date: tag(item, "date"), title: tag(item, "title"), teaser: truncate(text, 500), detailsXml: tag(item, "detailsXML"), videoUrl: tag(block(item, "video-stream"), "url"), fields: blocks(block(item, "politikfelder"), "politikfeld").map(stripHtml), changedDateTime: tag(item, "changedDateTime"), nextActions: [`bundestagctl article get --article-id ${articleId}`] };
    }).filter(Boolean);
}
function compactArticle(xml, grep) {
    const text = stripHtml(tag(xml, "text"));
    return { articleId: tag(xml, "articleId"), date: tag(xml, "date"), title: tag(xml, "title"), sourceUrl: tag(xml, "sourceURL"), fields: blocks(block(xml, "politikfelder"), "politikfeld").map(stripHtml), changedDateTime: tag(xml, "changedDateTime"), textLength: text.length, snippets: grepSnippets(text, grep, 5, 650) };
}
function articleEvidence(xml, grep) {
    const text = stripHtml(tag(xml, "text"));
    return { text: truncate(text, 1800), snippets: grepSnippets(text, grep, 5, 650), imageUrl: tag(xml, "imageURL"), imageSource: tag(xml, "imageCopyright"), imageAltText: tag(xml, "imageAltText") };
}
async function fetchXmlWithParams(base, params) {
    const requestUrl = withParams(base, params);
    const raw = await fetchRaw(requestUrl);
    if (raw.status < 200 || raw.status >= 300)
        throw new Error(`upstream status ${raw.status} from ${requestUrl}: ${truncate(stripSpace(raw.body), 300)}`);
    return { body: raw.body, requestUrl };
}
async function fetchRaw(requestUrl) {
    const response = await fetch(requestUrl, { headers: { "User-Agent": "germany-skills/bundestagctl-node-2.0" }, signal: AbortSignal.timeout(45000) });
    return { status: response.status, contentType: response.headers.get("content-type") ?? "", body: await response.text() };
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
function withParams(base, params) {
    const query = new URLSearchParams(params).toString();
    return query ? `${base}?${query}` : base;
}
function tag(xml, name) {
    if (!xml)
        return "";
    const match = new RegExp(`<${escapeRegExp(name)}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${escapeRegExp(name)}>`, "i").exec(xml);
    return match ? decodeEntities(stripSpace(match[1].replace(/^<!\[CDATA\[/, "").replace(/\]\]>$/, ""))) : "";
}
function tagAttr(xml, name, attrName) {
    const match = new RegExp(`<${escapeRegExp(name)}\\s+([^>]*)>`, "i").exec(xml);
    return match ? attr(match[1], attrName) : "";
}
function attr(text, name) {
    const match = new RegExp(`${escapeRegExp(name)}="([^"]*)"`).exec(text || "");
    return match ? decodeEntities(match[1]) : "";
}
function block(xml, name) {
    if (!xml)
        return "";
    const match = new RegExp(`<${escapeRegExp(name)}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${escapeRegExp(name)}>`, "i").exec(xml);
    return match ? match[1] : "";
}
function blocks(xml, name) {
    if (!xml)
        return [];
    return [...xml.matchAll(new RegExp(`<${escapeRegExp(name)}(?:\\s[^>]*)?>[\\s\\S]*?<\\/${escapeRegExp(name)}>`, "gi"))].map((match) => match[0]);
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
function printRaw(value) {
    process.stdout.write(value);
}
function source(title, url, kind) {
    return url ? [{ title, url, kind }] : [];
}
function defaultSources() {
    return [{ title: "Bundestag live XML OpenAPI wrapper", url: OPENAPI_URL, kind: "openapi_reference" }, { title: "Deutscher Bundestag Open Data", url: OPEN_DATA_URL, kind: "official_context" }, { title: "Bundestag website terms/imprint", url: IMPRINT_URL, kind: "terms" }, { title: "Bundestag audio/video terms", url: MEDIA_TERMS_URL, kind: "terms" }, { title: "Bundestag privacy policy", url: PRIVACY_URL, kind: "privacy" }];
}
function defaultWarnings() {
    return ["No exact public rate limit for these Bundestag XML feeds was found; use small limits and avoid repeated broad index pulls.", "This live/site XML surface is not the full parliamentary archive. Use dipctl for complete proceedings, printed papers, and plenary protocol research.", "Official Bundestag profile/disclosure data can include self-reported fields; preserve source URLs and timestamps in final citations.", "Website, image, and video materials may have separate usage terms; inspect the relevant source page/terms before republication."];
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
function isHelp(value) {
    return value === "--help" || value === "-h" || value === "help";
}
function matches(argv, ...expected) {
    return expected.every((value, index) => argv[index] === value);
}
function firstNonEmpty(...values) {
    for (const value of values)
        if (value !== undefined && value !== null && String(value).trim())
            return String(value).trim();
    return "";
}
function sourceKind(url) {
    if (url.includes("/xml/"))
        return "api_endpoint";
    if (url.includes("webtv.bundestag.de"))
        return "media_feed";
    if (url.includes("/mediathek"))
        return "media_page";
    if (url.includes("/abgeordnete/"))
        return "public_profile";
    return "public_page";
}
function articleIdFromUrl(value) {
    return /(\d{5,})(?:\.xml)?\/?(?:$|[?#])/.exec(value)?.[1] ?? "";
}
function stripHtml(value) {
    return stripSpace(decodeEntities(String(value ?? "").replace(/<(script|style)[^>]*>.*?<\/(script|style)>/gis, " ").replace(/<[^>]+>/g, " ")));
}
function stripSpace(value) {
    return String(value ?? "").replace(/\s+/g, " ").trim();
}
function decodeEntities(value) {
    return String(value ?? "").replace(/<!\[CDATA\[/g, "").replace(/\]\]>/g, "").replace(/&nbsp;/g, " ").replace(/&#160;/g, " ").replace(/&amp;/g, "&").replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&quot;/g, '"');
}
function truncate(value, maxLen) {
    return value.length <= maxLen ? value : `${value.slice(0, maxLen)}...`;
}
function grepSnippets(text, grep, limit, maxLen) {
    text = stripSpace(text);
    if (!text)
        return [];
    if (!grep)
        return [{ text: truncate(text, maxLen) }];
    const lower = text.toLowerCase();
    const needle = grep.toLowerCase().trim();
    const out = [];
    const seen = new Set();
    let startFrom = 0;
    while (out.length < limit) {
        const idx = lower.indexOf(needle, startFrom);
        if (idx < 0)
            break;
        const start = Math.max(0, idx - Math.floor(maxLen / 2));
        const end = Math.min(text.length, start + maxLen);
        const snippet = text.slice(start, end).trim();
        const key = snippet.slice(0, 180);
        if (!seen.has(key)) {
            out.push({ grep, text: snippet });
            seen.add(key);
        }
        startFrom = idx + needle.length;
    }
    return out;
}
function htmlTitle(value) {
    return stripHtml(/<title[^>]*>([\s\S]*?)<\/title>/i.exec(value ?? "")?.[1] ?? "");
}
function escapeRegExp(value) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
main(process.argv.slice(2)).then((code) => {
    process.exitCode = code;
});
