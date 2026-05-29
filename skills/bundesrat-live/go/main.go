package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	appName          = "bundesrat-live"
	baseURL          = "https://www.bundesrat.de"
	openAPIURL       = "https://github.com/bundesAPI/bundesrat-api"
	serviceBundURL   = "https://www.service.bund.de/Content/DE/DEBehoerden/B/BR/Bundesrat.html"
	imprintURL       = baseURL + "/DE/service-navi/impressum/impressum-node.html"
	privacyURL       = baseURL + "/DE/service-navi/datenschutz/datenschutz-node.html"
	robotsURL        = baseURL + "/robots.txt"
	defaultLimit     = 10
	safeLimit        = 100
	defaultTimeout   = 45 * time.Second
	defaultUserAgent = "germany-skills/bundesrat-live"
)

var endpoints = map[string]string{
	"startlist":            baseURL + "/iOS/v3/startlist_table.xml",
	"news":                 baseURL + "/iOS/v3/01_Aktuelles/aktuelles_table.xml",
	"dates":                baseURL + "/iOS/v3/02_Termine/termine_table.xml",
	"plenum compact":       baseURL + "/iOS/v3/03_Plenum/plenum_kompakt_table.xml",
	"plenum current":       baseURL + "/iOS/SharedDocs/3_Plenum/plenum_aktuelleSitzung_table.xml",
	"plenum chronological": baseURL + "/iOS/SharedDocs/3_Plenum/plenum_toChronologisch_table.xml",
	"plenum next":          baseURL + "/iOS/SharedDocs/3_Plenum/plenum_naechsteSitzungen.xml",
	"members":              baseURL + "/iOS/SharedDocs/2_Mitglieder/mitglieder_table.xml",
	"votes":                baseURL + "/iOS/v3/06_Stimmen/stimmverteilung.xml",
	"presidium":            baseURL + "/iOS/v3/05_Bundesrat/Praesidium/bundesrat_praesidium.xml",
}

type parsedArgs struct {
	flags       map[string]string
	params      url.Values
	positionals []string
}

type cliError struct {
	exitCode int
	code     string
	message  string
}

func (e cliError) Error() string { return e.message }

type httpError struct {
	statusCode int
	body       string
	url        string
}

func (e httpError) Error() string {
	return fmt.Sprintf("upstream status %d from %s: %s", e.statusCode, e.url, truncate(stripSpace(e.body), 300))
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 || isHelp(args[0]) {
		printRootHelp()
		return
	}
	if isHelp(args[len(args)-1]) {
		printHelp(args[:len(args)-1])
		return
	}

	var err error
	switch {
	case args[0] == "doctor":
		err = runDoctor(args[1:])
	case args[0] == "examples":
		printExamples()
	case args[0] == "startlist":
		err = runFeed("startlist", args[1:])
	case args[0] == "news" && len(args) > 1 && args[1] == "search":
		err = runFeedSearch("news", args[2:])
	case args[0] == "news" && len(args) > 1 && args[1] == "page":
		err = runPage("news page", args[2:])
	case args[0] == "news":
		err = runFeed("news", args[1:])
	case args[0] == "dates" && len(args) > 1 && args[1] == "search":
		err = runFeedSearch("dates", args[2:])
	case args[0] == "dates" && len(args) > 1 && args[1] == "page":
		err = runPage("dates page", args[2:])
	case args[0] == "dates":
		err = runFeed("dates", args[1:])
	case match(args, "plenum", "compact"):
		err = runPlenum("plenum compact", args[2:])
	case match(args, "plenum", "current"):
		err = runPlenum("plenum current", args[2:])
	case match(args, "plenum", "chronological"):
		err = runPlenum("plenum chronological", args[2:])
	case match(args, "plenum", "next"):
		err = runPlenumNext(args[2:])
	case match(args, "plenum", "dossier"):
		err = runPlenum("plenum compact", args[2:])
	case args[0] == "members" && len(args) > 1 && args[1] == "search":
		err = runMembersSearch(args[2:])
	case args[0] == "members" && len(args) > 1 && args[1] == "dossier":
		err = runMemberDossier(args[2:])
	case args[0] == "members":
		err = runMembers(args[1:])
	case args[0] == "votes" && len(args) > 1 && args[1] == "summary":
		err = runFeed("votes", args[2:])
	case args[0] == "votes":
		err = runFeed("votes", args[1:])
	case args[0] == "presidium":
		err = runFeed("presidium", args[1:])
	case args[0] == "page":
		err = runPage("page", args[1:])
	case args[0] == "source":
		err = runSource(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run bundesrat-live --help"}
	}

	if err != nil {
		var ce cliError
		if errors.As(err, &ce) {
			fail(ce.exitCode, ce.code, ce.message)
		}
		fail(1, "unexpected_error", err.Error())
	}
}

func printRootHelp() {
	fmt.Println(`bundesrat-live -- Bundesrat live/app XML research CLI

Purpose
  Discover and normalize public Bundesrat app XML feeds for news, dates,
  plenary-session summaries and agenda items, members, vote distribution,
  presidium/context pages, and source URLs.

Use this when
  - you need current Bundesrat public information from bundesrat.de
  - you need BundesratKOMPAKT plenary summaries, agenda TOPs, documents, or DIP links
  - you need current Bundesrat members, state affiliations, roles, or profile URLs
  - you need Bundesrat news/events with official public source pages

Do not use this when
  - you need the full parliamentary archive or Bundestag proceedings; use dip-bundestag
  - you need federal law text; use rechtsinformationen-bund
  - you need statistical evidence; use the relevant statistical CLI

Fast paths
  bundesrat-live doctor
  bundesrat-live news --limit 5
  bundesrat-live news search --term "Suchbegriff" --limit 3
  bundesrat-live dates --limit 5
  bundesrat-live members search --name "Ã–zdemir" --limit 3
  bundesrat-live members dossier --name "Ã–zdemir" --grep "Bundesrat"
  bundesrat-live plenum compact --limit 1 --top-limit 3
  bundesrat-live plenum current --limit 1 --top-limit 5
  bundesrat-live plenum next
  bundesrat-live page --url "https://www.bundesrat.de/SharedDocs/pm/2026/example.html" --grep "Suchbegriff"

Endpoint-compatible commands
  startlist
  news
  dates
  plenum compact
  plenum current
  plenum chronological
  plenum next
  members
  votes
  presidium

Research commands
  doctor
  examples
  news search
  news page
  dates search
  dates page
  members search
  members dossier
  plenum dossier
  votes summary
  page
  source

Output guarantees
  Commands emit JSON envelopes with status, request, summary/items, sources,
  warnings, and nextActions. Pass --raw on endpoint commands for raw XML.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "news search", "dates search":
		fmt.Println(`bundesrat-live news search

Search a Bundesrat feed and return compact source-rich rows.

Examples
  bundesrat-live news search --term "Suchbegriff" --limit 3
  bundesrat-live dates search --term "Ausschuss" --limit 5

Flags
  --term <text>       Search title, abstract, body, detail text, URL, date
  --limit <n>         Result cap, defaults to 10, safe max 100
  --include-raw       Include raw XML block for matching records`)
	case "members search":
		fmt.Println(`bundesrat-live members search

Search the current Bundesrat member feed.

Examples
  bundesrat-live members search --name "Ã–zdemir" --limit 3
  bundesrat-live members search --term "Bremen" --limit 5

Flags
  --name <text>       Search member name
  --term <text>       Search across name, party, state, roles, biography
  --limit <n>         Result cap, defaults to 10, safe max 100
  --include-raw       Include raw XML block for matching records`)
	case "members dossier":
		fmt.Println(`bundesrat-live members dossier

Build a compact dossier for one Bundesrat member from the official member feed.

Examples
  bundesrat-live members dossier --name "Mustername"
  bundesrat-live members dossier --url "https://www.bundesrat.de/SharedDocs/example/DE/example.html" --grep "Suchbegriff"

Flags
  --name <text>       Resolve by member name
  --url <url>         Resolve by official profile URL
  --grep <term>       Return matching snippets from role/contact/biography text
  --include-raw       Include raw XML block`)
	case "plenum compact", "plenum current", "plenum dossier":
		fmt.Println(`bundesrat-live plenum compact

Normalize Bundesrat plenary summary or agenda TOPs.

Examples
  bundesrat-live plenum compact --limit 1 --top-limit 3
  bundesrat-live plenum current --limit 1 --top-limit 5 --grep "Drucksache"
  bundesrat-live plenum dossier --grep "Sozialleistungsbetrug" --top-limit 5

Flags
  --limit <n>         Header/item cap, defaults to 10
  --top-limit <n>     Agenda TOP cap, defaults to 10
  --grep <term>       Filter/snippet agenda text
  --include-raw       Include raw TOP blocks
  --raw               Print original XML`)
	case "page", "news page", "dates page":
		fmt.Println(`bundesrat-live page

Fetch and normalize a public bundesrat.de source URL emitted by a feed.

Examples
  bundesrat-live page --url "https://www.bundesrat.de/SharedDocs/pm/2026/example.html" --grep "Suchbegriff"

Flags
  --url <url>         Public bundesrat.de URL
  --grep <term>       Return matching source snippets`)
	default:
		printRootHelp()
	}
}

func printExamples() {
	fmt.Println(`bundesrat-live examples

1. Check endpoint health and fair-use hints:
   bundesrat-live doctor

2. List the app feed catalog:
   bundesrat-live startlist --limit 12

3. Read latest Bundesrat news as compact JSON:
   bundesrat-live news --limit 5

4. Search news and expand a returned source page:
   bundesrat-live news search --term "Suchbegriff" --limit 3
   bundesrat-live news page --url "https://www.bundesrat.de/SharedDocs/pm/2026/example.html" --grep "Suchbegriff"

5. Inspect scheduled Bundesrat events:
   bundesrat-live dates --limit 5

6. Search current members:
   bundesrat-live members search --name "Ã–zdemir" --limit 3

7. Build a member dossier:
   bundesrat-live members dossier --name "Ã–zdemir" --grep "Bundesrat"

8. Inspect BundesratKOMPAKT plenary TOPs:
   bundesrat-live plenum compact --limit 1 --top-limit 3

9. Inspect current agenda/Drucksachen:
   bundesrat-live plenum current --limit 1 --top-limit 5

10. Fetch raw XML if the normalized summary is insufficient:
   bundesrat-live plenum compact --raw`)
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, 5, 10)
	checkNames := []string{"startlist", "news", "dates", "plenum compact", "members", "votes"}
	payload := envelope("doctor", baseURL, map[string]any{"limit": limit})
	summary := map[string]any{
		"authRequired":       false,
		"publishedRateLimit": "No exact public request quota was found in the OpenAPI wrapper or Bundesrat website material. The site robots.txt currently publishes Crawl-delay: 30; use small limits, cache repeated feed calls, and back off on 429/5xx responses.",
		"fairUseHints": []string{
			"Prefer search and dossier commands before broad source-page expansion.",
			"Respect robots.txt Crawl-delay: 30 for crawling-like workflows.",
			"Use --limit and --top-limit on broad feeds.",
			"Preserve source URLs, retrieval timestamps, and image/media copyright fields.",
		},
		"endpoints": []any{},
	}
	status := "ok"
	for _, name := range checkNames[:minInt(limit, len(checkNames))] {
		code, contentType, body, err := fetchRaw(withDefaultView(endpoints[name], nil))
		item := map[string]any{
			"name":        name,
			"url":         endpoints[name],
			"statusCode":  code,
			"contentType": contentType,
			"bodyPreview": truncate(stripSpace(string(body)), 180),
		}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
			status = "degraded"
		} else {
			item["ok"] = true
		}
		summary["endpoints"] = append(summary["endpoints"].([]any), item)
	}
	payload["status"] = status
	payload["summary"] = summary
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		"bundesrat-live news --limit 5",
		`bundesrat-live members search --name "Ã–zdemir" --limit 3`,
		"bundesrat-live plenum compact --limit 1 --top-limit 3",
	}
	emit(payload)
	return nil
}

func runFeed(key string, argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchEndpoint(key, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	grep := firstNonEmpty(parsed.flags["grep"], parsed.flags["term"], parsed.flags["q"])
	items := compactItems(raw, key, limit, grep, flagBool(parsed, "include-raw"))
	payload := envelope(key, requestURL, map[string]any{"limit": limit, "grep": grep})
	payload["summary"] = map[string]any{"totalItems": countItemLike(raw), "returned": len(items), "grep": grep}
	payload["items"] = items
	payload["sources"] = sources("Bundesrat "+key+" XML feed", requestURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromItems(items, key)
	emit(payload)
	return nil
}

func runFeedSearch(key string, argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], parsed.flags["name"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", key + " search requires --term"}
	}
	parsed.flags["grep"] = term
	return runFeed(key, rebuildArgs(parsed))
}

func runMembers(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchEndpoint("members", parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	items := compactEmployees(raw, limit, "", flagBool(parsed, "include-raw"))
	payload := envelope("members", requestURL, map[string]any{"limit": limit})
	payload["summary"] = map[string]any{"totalMembers": len(blocks(string(raw), "employee")), "returned": len(items)}
	payload["items"] = items
	payload["sources"] = sources("Bundesrat member XML feed", requestURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{`bundesrat-live members search --name "Ã–zdemir" --limit 3`}
	emit(payload)
	return nil
}

func runMembersSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["name"], parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "members search requires --name or --term"}
	}
	raw, requestURL, err := fetchEndpoint("members", nil)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	items := compactEmployees(raw, limit, term, flagBool(parsed, "include-raw"))
	payload := envelope("members search", requestURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"term": term, "totalMembers": len(blocks(string(raw), "employee")), "matchesReturned": len(items)}
	payload["items"] = items
	payload["sources"] = sources("Bundesrat member XML feed", requestURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromEmployees(items)
	emit(payload)
	return nil
}

func runMemberDossier(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["name"], parsed.flags["url"], parsed.flags["term"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_member", "members dossier requires --name or --url"}
	}
	raw, requestURL, err := fetchEndpoint("members", nil)
	if err != nil {
		return err
	}
	grep := parsed.flags["grep"]
	matches := compactEmployees(raw, safeLimit, term, flagBool(parsed, "include-raw"))
	if len(matches) == 0 {
		return cliError{2, "member_not_found", "member not found in current Bundesrat feed: " + term}
	}
	item := matches[0]
	text := fmt.Sprint(item["evidenceText"])
	payload := envelope("members dossier", requestURL, map[string]any{"term": term, "grep": grep})
	payload["summary"] = map[string]any{
		"name":         item["name"],
		"party":        item["party"],
		"state":        item["state"],
		"profileUrl":   item["url"],
		"snippetCount": len(grepSnippets(text, grep, 8, 650)),
	}
	payload["items"] = []any{map[string]any{
		"profile":  item,
		"snippets": grepSnippets(text, grep, 8, 650),
	}}
	payload["sources"] = []map[string]any{{"title": "Bundesrat member XML feed", "url": requestURL, "kind": "api_endpoint"}}
	if sourceURL := fmt.Sprint(item["url"]); sourceURL != "" {
		payload["sources"] = append(payload["sources"].([]map[string]any), map[string]any{"title": "Bundesrat member profile", "url": sourceURL, "kind": "public_profile"})
		payload["nextActions"] = []string{fmt.Sprintf("bundesrat-live page --url %q --grep %q", sourceURL, firstNonEmpty(grep, "Bundesrat"))}
	}
	payload["warnings"] = defaultWarnings()
	emit(payload)
	return nil
}

func runPlenum(key string, argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchEndpoint(key, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	topLimit := limitFlagName(parsed, "top-limit", defaultLimit, safeLimit)
	grep := firstNonEmpty(parsed.flags["grep"], parsed.flags["term"], parsed.flags["q"])
	items := compactPlenum(raw, key, limit, topLimit, grep, flagBool(parsed, "include-raw"))
	payload := envelope(key, requestURL, map[string]any{"limit": limit, "topLimit": topLimit, "grep": grep})
	payload["summary"] = plenumSummary(raw, key, len(items), grep)
	payload["items"] = items
	payload["sources"] = sources("Bundesrat "+key+" XML feed", requestURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromItems(items, key)
	emit(payload)
	return nil
}

func runPlenumNext(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchEndpoint("plenum next", parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	itemBlocks := blocks(string(raw), "item")
	items := compactItems(raw, "plenum next", limit, parsed.flags["grep"], flagBool(parsed, "include-raw"))
	var sessions []map[string]string
	if len(itemBlocks) > 0 {
		sessions = tableRows(tag(itemBlocks[0], "detail"))
	}
	payload := envelope("plenum next", requestURL, map[string]any{"limit": limit})
	payload["summary"] = map[string]any{"returned": len(items), "upcomingSessions": sessions}
	payload["items"] = items
	payload["sources"] = sources("Bundesrat next plenary sessions XML feed", requestURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundesrat-live plenum current --limit 1 --top-limit 5", "bundesrat-live plenum compact --limit 1 --top-limit 5"}
	emit(payload)
	return nil
}

func runPage(command string, argv []string) error {
	parsed := parseArgs(argv)
	sourceURL := firstNonEmpty(parsed.flags["url"], parsed.flags["source-url"], strings.Join(parsed.positionals, " "))
	if sourceURL == "" {
		return cliError{2, "missing_url", command + " requires --url"}
	}
	if !strings.HasPrefix(sourceURL, baseURL+"/") {
		return cliError{2, "unsafe_url", "page only accepts https://www.bundesrat.de URLs"}
	}
	code, contentType, body, err := fetchRaw(sourceURL)
	if err != nil {
		return err
	}
	grep := parsed.flags["grep"]
	text := stripHTML(string(body))
	payload := envelope(command, sourceURL, map[string]any{"url": sourceURL, "grep": grep})
	payload["summary"] = map[string]any{
		"url":          sourceURL,
		"statusCode":   code,
		"contentType":  contentType,
		"title":        htmlTitle(string(body)),
		"textLength":   len(text),
		"snippetCount": len(grepSnippets(text, grep, 8, 650)),
	}
	payload["items"] = grepSnippets(text, grep, 8, 650)
	payload["sources"] = sources("Bundesrat public source page", sourceURL, "public_page")
	payload["warnings"] = append(defaultWarnings(), "Public HTML extraction is best-effort; prefer XML feed fields for structured metadata.")
	payload["nextActions"] = []string{fmt.Sprintf("bundesrat-live source --url %q", sourceURL)}
	emit(payload)
	return nil
}

func runSource(argv []string) error {
	parsed := parseArgs(argv)
	sourceURL := firstNonEmpty(parsed.flags["url"], strings.Join(parsed.positionals, " "))
	if sourceURL == "" {
		return cliError{2, "missing_url", "source requires --url"}
	}
	payload := envelope("source", sourceURL, map[string]any{"url": sourceURL})
	payload["summary"] = map[string]any{
		"url":      sourceURL,
		"kind":     sourceKind(sourceURL),
		"citation": "Bundesrat, " + sourceURL,
	}
	payload["sources"] = sources("Bundesrat source", sourceURL, sourceKind(sourceURL))
	payload["warnings"] = defaultWarnings()
	if strings.HasPrefix(sourceURL, baseURL+"/") {
		payload["nextActions"] = []string{fmt.Sprintf("bundesrat-live page --url %q", sourceURL)}
	}
	emit(payload)
	return nil
}

func fetchEndpoint(key string, params url.Values) ([]byte, string, error) {
	endpoint, ok := endpoints[key]
	if !ok {
		return nil, "", cliError{2, "unknown_endpoint", "unknown endpoint: " + key}
	}
	requestURL := withDefaultView(endpoint, params)
	status, _, body, err := fetchRaw(requestURL)
	if err != nil {
		return nil, requestURL, err
	}
	if status < 200 || status >= 300 {
		return nil, requestURL, httpError{status, string(body), requestURL}
	}
	return body, requestURL, nil
}

func fetchRaw(requestURL string) (int, string, []byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp.StatusCode, resp.Header.Get("Content-Type"), body, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, resp.Header.Get("Content-Type"), body, httpError{resp.StatusCode, string(body), requestURL}
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func withDefaultView(base string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("view") == "" {
		params.Set("view", "renderXml")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func compactItems(raw []byte, key string, limit int, grep string, includeRaw bool) []map[string]any {
	var out []map[string]any
	for _, block := range blocks(string(raw), "item") {
		searchText := itemSearchText(block)
		if grep != "" && !strings.Contains(strings.ToLower(searchText), strings.ToLower(grep)) {
			continue
		}
		item := compactItem(block, key, grep, includeRaw)
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactItem(block string, key string, grep string, includeRaw bool) map[string]any {
	detail := tag(block, "detail")
	text := firstNonEmpty(tag(block, "bodyText"), tag(block, "description"), tag(block, "abstract"), stripHTML(detail))
	sourceURL := tag(block, "url")
	item := map[string]any{
		"type":         tag(block, "type"),
		"id":           tag(block, "id"),
		"name":         tag(block, "name"),
		"title":        firstNonEmpty(tag(block, "title"), tag(block, "name")),
		"url":          sourceURL,
		"date":         tag(block, "date"),
		"dateOfIssue":  tag(block, "dateOfIssue"),
		"startDate":    tag(block, "startdate"),
		"stopDate":     tag(block, "stopdate"),
		"summary":      truncate(stripHTML(text), 700),
		"imageUrl":     tag(block, "imagePath"),
		"imageDate":    tag(block, "imageDate"),
		"imageCaption": tag(block, "imageCaption"),
		"sources":      sources("Bundesrat source", sourceURL, sourceKind(sourceURL)),
		"links":        extractLinks(detail, 10),
		"snippets":     grepSnippets(stripHTML(detail+" "+text), grep, 4, 650),
		"nextActions":  nextActionsForURL(sourceURL, key),
	}
	if includeRaw {
		item["raw"] = block
	}
	return item
}

func compactEmployees(raw []byte, limit int, term string, includeRaw bool) []map[string]any {
	var out []map[string]any
	for _, block := range blocks(string(raw), "employee") {
		text := employeeSearchText(block)
		if term != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(term)) {
			continue
		}
		firstName := tag(block, "firstname")
		lastName := tag(block, "name")
		sourceURL := tag(block, "url")
		evidence := stripHTML(strings.Join([]string{tag(block, "detail1"), tag(block, "detail2"), tag(block, "detail3")}, " "))
		item := map[string]any{
			"name":              stripSpace(firstName + " " + lastName),
			"firstName":         firstName,
			"lastName":          lastName,
			"party":             tag(block, "party"),
			"state":             tag(block, "state"),
			"isBundesratMember": tag(block, "brmitglied"),
			"isMember":          tag(block, "mitglied"),
			"isBevollmaechtigt": tag(block, "bv"),
			"url":               sourceURL,
			"imageUrl":          tag(block, "imagePath"),
			"roles":             truncate(stripHTML(tag(block, "detail1")), 1000),
			"biography":         truncate(stripHTML(tag(block, "detail2")), 1000),
			"contact":           truncate(stripHTML(tag(block, "detail3")), 1000),
			"evidenceText":      evidence,
			"sources":           sources("Bundesrat member profile", sourceURL, "public_profile"),
			"nextActions":       nextActionsForURL(sourceURL, "members"),
		}
		if includeRaw {
			item["raw"] = block
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactPlenum(raw []byte, key string, limit int, topLimit int, grep string, includeRaw bool) []map[string]any {
	var out []map[string]any
	xmlText := string(raw)
	header := block(xmlText, "header")
	if header != "" && strings.Contains(header, "<url>") {
		out = append(out, map[string]any{
			"kind":        "header",
			"url":         tag(header, "url"),
			"title":       firstNonEmpty(tag(header, "titel2"), tag(header, "title")),
			"subtitle":    stripSpace(strings.Join([]string{tag(header, "titel1"), tag(header, "titel3"), tag(header, "titelAlt")}, " ")),
			"detailType":  tag(header, "detailTyp"),
			"summary":     truncate(stripHTML(firstNonEmpty(tag(header, "vorschautext"), tag(header, "detail"))), 1000),
			"sources":     sources("Bundesrat plenary page", tag(header, "url"), "public_page"),
			"links":       extractLinks(tag(header, "detail"), 10),
			"snippets":    grepSnippets(stripHTML(tag(header, "detail")), grep, 4, 650),
			"nextActions": nextActionsForURL(tag(header, "url"), key),
		})
	}
	for _, top := range blocks(xmlText, "top") {
		searchText := stripHTML(top)
		if grep != "" && !strings.Contains(strings.ToLower(searchText), strings.ToLower(grep)) {
			continue
		}
		detail := firstNonEmpty(tag(top, "detail"), tag(top, "topdetail"))
		sourceURL := tag(top, "url")
		item := map[string]any{
			"kind":        "top",
			"top":         firstNonEmpty(tag(top, "nr"), tag(top, "toptitle")),
			"printMatter": tag(top, "topdrucksache"),
			"filter":      tag(top, "filter"),
			"title":       firstNonEmpty(tag(top, "title"), tag(top, "topheader")),
			"url":         sourceURL,
			"summary":     truncate(stripHTML(firstNonEmpty(tag(top, "topheader"), detail)), 900),
			"links":       extractLinks(detail, 14),
			"snippets":    grepSnippets(stripHTML(detail), grep, 5, 650),
			"sources":     sources("Bundesrat plenary TOP", sourceURL, sourceKind(sourceURL)),
			"nextActions": nextActionsForURL(sourceURL, key),
		}
		if includeRaw {
			item["raw"] = top
		}
		out = append(out, item)
		if len(out) >= limit+topLimit {
			break
		}
	}
	if len(out) == 0 {
		return compactItems(raw, key, limit, grep, includeRaw)
	}
	return out[:minInt(len(out), limit+topLimit)]
}

func plenumSummary(raw []byte, key string, returned int, grep string) map[string]any {
	xmlText := string(raw)
	return map[string]any{
		"title":     tag(xmlText, "title"),
		"header":    truncate(stripHTML(block(xmlText, "header")), 900),
		"topCount":  len(blocks(xmlText, "top")),
		"itemCount": len(blocks(xmlText, "item")),
		"returned":  returned,
		"grep":      grep,
		"sourceUrl": firstNonEmpty(tag(block(xmlText, "header"), "url"), endpoints[key]),
	}
}

func itemSearchText(block string) string {
	return stripHTML(strings.Join([]string{
		tag(block, "type"), tag(block, "id"), tag(block, "name"), tag(block, "title"),
		tag(block, "url"), tag(block, "date"), tag(block, "dateOfIssue"), tag(block, "bodyText"),
		tag(block, "description"), tag(block, "abstract"), tag(block, "detail"),
	}, " "))
}

func employeeSearchText(block string) string {
	return stripHTML(strings.Join([]string{
		tag(block, "firstname"), tag(block, "name"), tag(block, "party"), tag(block, "state"),
		tag(block, "url"), tag(block, "detail1"), tag(block, "detail2"), tag(block, "detail3"),
	}, " "))
}

func nextActionsFromItems(items []map[string]any, key string) []string {
	var actions []string
	for _, item := range items {
		for _, action := range nextActionsForURL(fmt.Sprint(item["url"]), key) {
			actions = append(actions, action)
			if len(actions) >= 4 {
				return actions
			}
		}
	}
	if key == "news" {
		return []string{`bundesrat-live news search --term "Suchbegriff" --limit 3`}
	}
	if key == "dates" {
		return []string{`bundesrat-live dates search --term "Ausschuss" --limit 5`}
	}
	return []string{"bundesrat-live plenum compact --limit 1 --top-limit 3"}
}

func nextActionsFromEmployees(items []map[string]any) []string {
	var actions []string
	for _, item := range items {
		if name := fmt.Sprint(item["name"]); name != "" {
			actions = append(actions, fmt.Sprintf("bundesrat-live members dossier --name %q", name))
		}
		if len(actions) >= 3 {
			break
		}
	}
	if len(actions) == 0 {
		return []string{`bundesrat-live members search --name "Ã–zdemir" --limit 3`}
	}
	return actions
}

func nextActionsForURL(sourceURL string, key string) []string {
	if sourceURL == "" {
		return nil
	}
	var actions []string
	if strings.HasPrefix(sourceURL, baseURL+"/") {
		actions = append(actions, fmt.Sprintf("bundesrat-live page --url %q", sourceURL))
	}
	if key == "news" {
		actions = append(actions, fmt.Sprintf("bundesrat-live news page --url %q", sourceURL))
	}
	if key == "dates" {
		actions = append(actions, fmt.Sprintf("bundesrat-live dates page --url %q", sourceURL))
	}
	return actions
}

func defaultSources() []map[string]any {
	return []map[string]any{
		{"title": "bundesAPI Bundesrat OpenAPI wrapper", "url": openAPIURL, "kind": "openapi_reference"},
		{"title": "service.bund.de Bundesrat profile", "url": serviceBundURL, "kind": "official_context"},
		{"title": "Bundesrat robots.txt", "url": robotsURL, "kind": "fair_use"},
		{"title": "Bundesrat Impressum", "url": imprintURL, "kind": "terms"},
		{"title": "Bundesrat DatenschutzerklÃ¤rung", "url": privacyURL, "kind": "privacy"},
	}
}

func defaultWarnings() []string {
	return []string{
		"No exact public rate limit for these Bundesrat XML feeds was found; robots.txt publishes Crawl-delay: 30, so avoid crawling-style rapid page expansion.",
		"This app/live XML surface is current-publication oriented, not a complete historical archive.",
		"Bundesrat public pages can include image/media copyright notices; preserve source URLs and copyright fields in final artifacts.",
		"Votes by individual Land are generally not always recorded by the Bundesrat itself; inspect plenary records and linked state pages where the distinction matters.",
	}
}

func parseArgs(args []string) parsedArgs {
	parsed := parsedArgs{flags: map[string]string{}, params: url.Values{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			parsed.positionals = append(parsed.positionals, arg)
			continue
		}
		keyValue := strings.TrimPrefix(arg, "--")
		key := keyValue
		value := "true"
		if idx := strings.Index(keyValue, "="); idx >= 0 {
			key = keyValue[:idx]
			value = keyValue[idx+1:]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "param" {
			if idx := strings.Index(value, "="); idx > 0 {
				parsed.params.Add(value[:idx], value[idx+1:])
			}
			continue
		}
		parsed.flags[key] = value
	}
	return parsed
}

func rebuildArgs(parsed parsedArgs) []string {
	var args []string
	for key, value := range parsed.flags {
		args = append(args, "--"+key, value)
	}
	for key, values := range parsed.params {
		for _, value := range values {
			args = append(args, "--param", key+"="+value)
		}
	}
	args = append(args, parsed.positionals...)
	return args
}

func limitFlag(parsed parsedArgs, fallback int, maxValue int) int {
	return limitFlagName(parsed, "limit", fallback, maxValue)
}

func limitFlagName(parsed parsedArgs, name string, fallback int, maxValue int) int {
	value := fallback
	if raw := parsed.flags[name]; raw != "" {
		if parsedValue, err := strconv.Atoi(raw); err == nil && parsedValue > 0 {
			value = parsedValue
		}
	}
	if value > maxValue && !flagBool(parsed, "allow-large-output") {
		fail(2, "limit_exceeds_safe_max", fmt.Sprintf("%s %d exceeds safe max %d; pass --allow-large-output to override", name, value, maxValue))
	}
	return value
}

func flagBool(parsed parsedArgs, key string) bool {
	value := strings.ToLower(parsed.flags[key])
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func envelope(command string, requestURL string, request any) map[string]any {
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request":     map[string]any{"method": "GET", "url": requestURL, "params": request},
		"summary":     map[string]any{},
		"items":       []any{},
		"sources":     []any{},
		"warnings":    []string{},
		"nextActions": []string{},
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func fail(exitCode int, code string, message string) {
	emit(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error":       map[string]any{"code": code, "message": message},
	})
	os.Exit(exitCode)
}

func isHelp(value string) bool { return value == "--help" || value == "-h" || value == "help" }

func match(args []string, expected ...string) bool {
	if len(args) < len(expected) {
		return false
	}
	for index, value := range expected {
		if args[index] != value {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func countItemLike(raw []byte) int {
	text := string(raw)
	count := len(blocks(text, "item"))
	if count == 0 {
		count = len(blocks(text, "employee")) + len(blocks(text, "top"))
	}
	return count
}

func sourceKind(sourceURL string) string {
	switch {
	case sourceURL == "":
		return "unknown"
	case strings.Contains(sourceURL, "dip.bundestag.de"):
		return "dip_reference"
	case strings.Contains(sourceURL, "/SharedDocs/personen/"):
		return "public_profile"
	case strings.Contains(sourceURL, "/SharedDocs/drucksachen/") || strings.Contains(sourceURL, "/drs.html"):
		return "official_document"
	case strings.Contains(sourceURL, "/DE/plenum/"):
		return "plenary_page"
	case strings.Contains(sourceURL, "/SharedDocs/pm/"):
		return "press_release"
	default:
		return "public_page"
	}
}

func sources(title string, sourceURL string, kind string) []map[string]any {
	if sourceURL == "" {
		return nil
	}
	return []map[string]any{{"title": title, "url": sourceURL, "kind": kind}}
}

var (
	scriptStylePattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTagPattern     = regexp.MustCompile(`(?s)<[^>]+>`)
	spacePattern       = regexp.MustCompile(`\s+`)
	cdataStartPattern  = regexp.MustCompile(`^<!\[CDATA\[`)
	cdataEndPattern    = regexp.MustCompile(`\]\]>$`)
	titlePattern       = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	linkPattern        = regexp.MustCompile(`(?is)<a\s+[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	tableRowPattern    = regexp.MustCompile(`(?is)<tr>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*</tr>`)
)

func block(xmlText string, name string) string {
	matches := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(name) + `(?:\s[^>]*)?>(.*?)</` + regexp.QuoteMeta(name) + `>`).FindStringSubmatch(xmlText)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func blocks(xmlText string, name string) []string {
	matches := regexp.MustCompile(`(?is)<`+regexp.QuoteMeta(name)+`(?:\s[^>]*)?>.*?</`+regexp.QuoteMeta(name)+`>`).FindAllString(xmlText, -1)
	if matches == nil {
		return []string{}
	}
	return matches
}

func tag(xmlText string, name string) string {
	value := block(xmlText, name)
	value = cdataStartPattern.ReplaceAllString(value, "")
	value = cdataEndPattern.ReplaceAllString(value, "")
	return strings.TrimSpace(html.UnescapeString(value))
}

func stripHTML(value string) string {
	value = cdataStartPattern.ReplaceAllString(value, "")
	value = cdataEndPattern.ReplaceAllString(value, "")
	value = scriptStylePattern.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "<br/>", " ")
	value = strings.ReplaceAll(value, "<br>", " ")
	value = strings.ReplaceAll(value, "<br />", " ")
	value = htmlTagPattern.ReplaceAllString(value, " ")
	return stripSpace(html.UnescapeString(value))
}

func stripSpace(value string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func truncate(value string, maxLen int) string {
	value = stripSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}

func grepSnippets(text string, grep string, limit int, maxLen int) []map[string]any {
	text = stripSpace(text)
	if text == "" {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(grep))
	if needle == "" {
		return []map[string]any{{"text": truncate(text, maxLen)}}
	}
	lower := strings.ToLower(text)
	var out []map[string]any
	seen := map[string]bool{}
	searchFrom := 0
	for len(out) < limit {
		idx := strings.Index(lower[searchFrom:], needle)
		if idx < 0 {
			break
		}
		idx += searchFrom
		start := idx - maxLen/2
		if start < 0 {
			start = 0
		}
		end := start + maxLen
		if end > len(text) {
			end = len(text)
		}
		snippet := strings.TrimSpace(text[start:end])
		key := snippet
		if len(key) > 180 {
			key = key[:180]
		}
		if !seen[key] {
			out = append(out, map[string]any{"grep": grep, "text": snippet})
			seen[key] = true
		}
		searchFrom = idx + len(needle)
	}
	return out
}

func extractLinks(value string, limit int) []map[string]any {
	var out []map[string]any
	seen := map[string]bool{}
	for _, match := range linkPattern.FindAllStringSubmatch(value, -1) {
		rawURL := strings.TrimSpace(html.UnescapeString(match[1]))
		if rawURL == "" || strings.HasPrefix(rawURL, "mailto:") || strings.HasPrefix(rawURL, "tel:") {
			continue
		}
		if strings.HasPrefix(rawURL, "/") {
			rawURL = baseURL + rawURL
		}
		if !strings.HasPrefix(rawURL, "http") {
			rawURL = baseURL + "/" + strings.TrimPrefix(rawURL, "./")
		}
		if seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		out = append(out, map[string]any{"title": truncate(stripHTML(match[2]), 160), "url": rawURL, "kind": sourceKind(rawURL)})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func tableRows(value string) []map[string]string {
	var rows []map[string]string
	for _, match := range tableRowPattern.FindAllStringSubmatch(value, -1) {
		rows = append(rows, map[string]string{"date": stripHTML(match[1]), "time": stripHTML(match[2])})
	}
	return rows
}

func htmlTitle(value string) string {
	matches := titlePattern.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return stripHTML(matches[1])
}
