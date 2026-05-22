package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName          = "tagesschau"
	baseURL          = "https://www.tagesschau.de"
	homepageURL      = "https://www.tagesschau.de/api2u/homepage"
	newsURL          = "https://www.tagesschau.de/api2u/news"
	channelsURL      = "https://www.tagesschau.de/api2u/channels"
	searchURL        = "https://www.tagesschau.de/api2u/search"
	apiDocsURL       = "https://github.com/bundesAPI/tagesschau-api"
	openAPIURL       = "https://github.com/bundesAPI/tagesschau-api/raw/refs/heads/main/openapi.yaml"
	ccURL            = "https://www.tagesschau.de/multimedia/video/creative-commons-index-100.html"
	rssInfoURL       = "https://www.tagesschau.de/infoservices/rssfeeds"
	defaultUserAgent = "germany-skills/tagesschau-2.0"
	defaultLimit     = 10
	maxLimit         = 30
	defaultTimeout   = 35 * time.Second
)

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
	return fmt.Sprintf("upstream status %d from %s: %s", e.statusCode, e.url, truncate(stripSpace(e.body), 260))
}

type contentBlock struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Video any    `json:"video"`
	Audio any    `json:"audio"`
	Image any    `json:"image"`
	Extra map[string]any
}

func (c *contentBlock) UnmarshalJSON(raw []byte) error {
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	c.Extra = values
	if value, ok := values["type"].(string); ok {
		c.Type = value
	}
	if value, ok := values["value"].(string); ok {
		c.Value = value
	}
	c.Video = values["video"]
	c.Audio = values["audio"]
	c.Image = values["image"]
	return nil
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
	case args[0] == "source":
		err = runSource(args[1:])
	case args[0] == "fields":
		err = runFields(args[1:])
	case args[0] == "homepage":
		err = runFeed("homepage", homepageURL, args[1:])
	case args[0] == "news":
		err = runFeed("news", newsURL, args[1:])
	case args[0] == "channels":
		err = runFeed("channels", channelsURL, args[1:])
	case args[0] == "search":
		err = runSearch(args[1:])
	case match(args, "article", "get"):
		err = runArticle("article get", args[2:], false)
	case match(args, "article", "source"):
		err = runArticleSource(args[2:])
	case match(args, "article", "dossier"):
		err = runArticle("article dossier", args[2:], true)
	default:
		err = cliError{2, "unknown_command", "unknown command: " + strings.Join(args, " ")}
	}
	if err != nil {
		fail(err)
	}
}

func printRootHelp() {
	fmt.Println(`tagesschau -- Tagesschau public JSON feed research CLI

Usage:
  tagesschau doctor
  tagesschau homepage --limit 5
  tagesschau news --ressort inland --limit 5
  tagesschau channels --limit 5
  tagesschau search --text "Bundestag" --limit 5
  tagesschau article get --url "https://www.tagesschau.de/...-100.html" --grep "Bundestag"
  tagesschau article dossier --url "https://www.tagesschau.de/...-100.html"

Research commands:
  doctor          Check endpoint health, auth, 60/hour request limit, and usage warnings.
  source          Print canonical API, usage, and Creative Commons references.
  fields          Explain filters, regions, ressorts, URLs, and content fields.
  homepage        Compact homepage feed with article next actions.
  news            Compact news feed. Supports --ressort, --regions, --param key=value, --raw.
  channels        Compact channels feed.
  search          Compact search. Supports --text/--searchText, --limit, --result-page, --raw.
  article get     Fetch one API/detail URL or public detailsweb URL and return bounded snippets.
  article source  Return source metadata for one article URL.
  article dossier Bundle metadata, snippets, links, caveats, and source references.

Tagesschau is a current-news context source. Do not treat it as the sole official evidence for parliamentary, legal, fiscal, or statistical claims.`)
}

func printHelp(args []string) {
	switch {
	case len(args) == 0:
		printRootHelp()
	case args[0] == "search":
		fmt.Println("search flags: --text/--searchText TERM --limit 1-30 --result-page N --include-raw --raw --param key=value")
	case args[0] == "news":
		fmt.Println("news flags: --ressort inland|ausland|wirtschaft|sport|video|investigativ|wissen --regions 1,2 --limit 1-30 --include-raw --raw --param key=value")
	case args[0] == "homepage":
		fmt.Println("homepage flags: --limit 1-30 --include-regional --include-raw --raw")
	case match(args, "article", "get"):
		fmt.Println("article get flags: --url URL --grep TERM --limit 1-30 --include-raw --raw")
	case match(args, "article", "source"):
		fmt.Println("article source flags: --url URL")
	case match(args, "article", "dossier"):
		fmt.Println("article dossier flags: --url URL --grep TERM --limit 1-30 --include-raw")
	default:
		printRootHelp()
	}
}

func printExamples() {
	fmt.Println(`Examples:
  tagesschau doctor
  tagesschau homepage --limit 5
  tagesschau news --ressort inland --limit 5
  tagesschau search --text "Bundestag" --limit 5
  tagesschau search --param searchText=Bundestag --param pageSize=5
  tagesschau article get --url "https://www.tagesschau.de/inland/example-100.html" --grep "Bundestag"
  tagesschau article dossier --url "https://www.tagesschau.de/api2u/inland/example-100.json"`)
}

func runDoctor(argv []string) error {
	checks := []map[string]any{}
	for _, check := range []struct {
		name string
		url  string
	}{
		{"homepage", homepageURL},
		{"news", withParams(newsURL, url.Values{"ressort": {"inland"}})},
		{"channels", channelsURL},
		{"search", withParams(searchURL, url.Values{"searchText": {"Bundestag"}, "pageSize": {"1"}})},
	} {
		status, contentType, raw, err := fetchRaw(check.url, "application/json")
		item := map[string]any{"name": check.name, "url": check.url}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
		} else {
			item["ok"] = status >= 200 && status < 300
			item["statusCode"] = status
			item["contentType"] = contentType
			item["bodyBytes"] = len(raw)
		}
		checks = append(checks, item)
	}
	payload := envelope("doctor", "GET", "multiple", map[string]any{})
	payload["summary"] = map[string]any{
		"authRequired":       false,
		"documentedLimit":    "The published API documentation states that more than 60 requests per hour are not allowed.",
		"usageRestrictions":  "Private, non-commercial use is allowed; publication is not allowed except for content explicitly released under a Creative Commons license.",
		"recommendedRole":    "Use as a current-news context layer, not as the sole official source for institutional or statistical claims.",
		"endpointHealth":     checks,
		"copyrightSensitive": true,
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		"tagesschau search --text \"Bundestag\" --limit 5",
		"tagesschau homepage --limit 5",
		"tagesschau source",
	}
	emit(payload)
	return nil
}

func runSource(argv []string) error {
	payload := envelope("source", "GET", apiDocsURL, map[string]any{})
	payload["summary"] = map[string]any{
		"publisher":         "Tagesschau / ARD-aktuell; API documentation mirrored by bundesAPI.",
		"authRequired":      false,
		"documentedLimit":   "No more than 60 requests per hour.",
		"reuseRestriction":  "Private, non-commercial use only; no publication except explicitly CC-licensed offers.",
		"primaryEndpoints":  []string{homepageURL, newsURL, channelsURL, searchURL},
		"articleURLPattern": "Public detailsweb URLs can be converted to /api2u/...json detail URLs.",
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"tagesschau fields", "tagesschau search --text \"Bundestag\" --limit 5"}
	emit(payload)
	return nil
}

func runFields(argv []string) error {
	payload := envelope("fields", "GET", apiDocsURL, map[string]any{})
	payload["summary"] = map[string]any{
		"feeds": []map[string]any{
			{"command": "homepage", "meaning": "Selected current and breaking items shown in the app homepage."},
			{"command": "news", "meaning": "Current news feed; filterable by ressort and region."},
			{"command": "channels", "meaning": "Current livestream/program channels."},
			{"command": "search", "meaning": "Search feed with searchText, resultPage, and pageSize."},
		},
		"ressorts": []string{"inland", "ausland", "wirtschaft", "sport", "video", "investigativ", "wissen"},
		"regions": map[string]string{
			"1": "Baden-WÃ¼rttemberg", "2": "Bayern", "3": "Berlin", "4": "Brandenburg", "5": "Bremen", "6": "Hamburg", "7": "Hessen", "8": "Mecklenburg-Vorpommern",
			"9": "Niedersachsen", "10": "Nordrhein-Westfalen", "11": "Rheinland-Pfalz", "12": "Saarland", "13": "Sachsen", "14": "Sachsen-Anhalt", "15": "Schleswig-Holstein", "16": "ThÃ¼ringen",
		},
		"coreArticleFields": []string{"title", "topline", "date", "details", "detailsweb", "shareURL", "firstSentence", "ressort", "type", "tags"},
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"tagesschau search --text \"Bundestag\" --limit 5", "tagesschau homepage --limit 5"}
	emit(payload)
	return nil
}

func runFeed(command string, endpoint string, argv []string) error {
	parsed := parseArgs(argv)
	params := cloneValues(parsed.params)
	for _, key := range []string{"ressort", "regions"} {
		if value := parsed.flags[key]; value != "" {
			params.Set(key, value)
		}
	}
	requestURL := withParams(endpoint, params)
	status, _, raw, err := fetchRaw(requestURL, "application/json")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return httpError{status, string(raw), requestURL}
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, maxLimit)
	items := compactFeedItems(data, limit, flagBool(parsed, "include-regional"))
	payload := envelope(command, "GET", requestURL, paramsToMap(params))
	payload["summary"] = map[string]any{
		"type":          data["type"],
		"itemsReturned": len(items),
		"rawCounts":     feedCounts(data),
	}
	payload["items"] = items
	payload["sources"] = []map[string]string{{"kind": "api_request", "title": "Tagesschau API request", "url": requestURL}}
	payload["sources"] = append(payload["sources"].([]map[string]string), defaultSources()...)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromItems(items)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runSearch(argv []string) error {
	parsed := parseArgs(argv)
	params := cloneValues(parsed.params)
	text := firstNonEmpty(parsed.flags["text"], parsed.flags["searchText"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if text != "" {
		params.Set("searchText", text)
	}
	limit := limitFlag(parsed, defaultLimit, maxLimit)
	if params.Get("pageSize") == "" {
		params.Set("pageSize", strconv.Itoa(limit))
	}
	if value := firstNonEmpty(parsed.flags["page-size"], parsed.flags["pageSize"]); value != "" {
		params.Set("pageSize", value)
	}
	if value := firstNonEmpty(parsed.flags["result-page"], parsed.flags["resultPage"], parsed.flags["page"]); value != "" {
		params.Set("resultPage", value)
	}
	requestURL := withParams(searchURL, params)
	status, _, raw, err := fetchRaw(requestURL, "application/json")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return httpError{status, string(raw), requestURL}
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	items := compactArray(data["searchResults"], limit)
	payload := envelope("search", "GET", requestURL, paramsToMap(params))
	payload["summary"] = map[string]any{
		"searchText":      data["searchText"],
		"totalItemCount":  data["totalItemCount"],
		"pageSize":        data["pageSize"],
		"resultPage":      data["resultPage"],
		"itemsReturned":   len(items),
		"copyrightNotice": "Do not republish Tagesschau article text unless content is explicitly CC-licensed.",
	}
	payload["items"] = items
	payload["sources"] = []map[string]string{{"kind": "api_request", "title": "Tagesschau API request", "url": requestURL}}
	payload["sources"] = append(payload["sources"].([]map[string]string), defaultSources()...)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromItems(items)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runArticle(command string, argv []string, dossier bool) error {
	parsed := parseArgs(argv)
	inputURL := firstNonEmpty(parsed.flags["url"], parsed.params.Get("url"), strings.Join(parsed.positionals, " "))
	if inputURL == "" {
		return cliError{2, "missing_url", command + " requires --url"}
	}
	apiURL, publicURL, err := articleURLs(inputURL)
	if err != nil {
		return err
	}
	status, _, raw, err := fetchRaw(apiURL, "application/json")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return httpError{status, string(raw), apiURL}
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, maxLimit)
	grep := firstNonEmpty(parsed.flags["grep"], parsed.flags["term"], parsed.flags["q"])
	snippets := articleSnippets(data, grep, limit)
	item := compactArticle(data)
	item["snippetCount"] = len(snippets)
	item["snippets"] = snippets
	payload := envelope(command, "GET", apiURL, map[string]any{"url": inputURL, "grep": grep, "limit": limit})
	payload["summary"] = item
	payload["items"] = snippets
	payload["sources"] = []map[string]string{
		{"kind": "api_request", "title": "Tagesschau article JSON", "url": apiURL},
		{"kind": "public_article", "title": "Tagesschau public article", "url": publicURL},
	}
	payload["sources"] = append(payload["sources"].([]map[string]string), defaultSources()...)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("tagesschau article source --url %q", publicURL),
	}
	if dossier {
		payload["summary"].(map[string]any)["dossierUse"] = "Use as current-news context; verify institutional/statistical claims against primary official sources."
		payload["nextActions"] = append(payload["nextActions"].([]string), "tagesschau source")
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runArticleSource(argv []string) error {
	parsed := parseArgs(argv)
	inputURL := firstNonEmpty(parsed.flags["url"], parsed.params.Get("url"), strings.Join(parsed.positionals, " "))
	if inputURL == "" {
		return cliError{2, "missing_url", "article source requires --url"}
	}
	apiURL, publicURL, err := articleURLs(inputURL)
	if err != nil {
		return err
	}
	payload := envelope("article source", "GET", apiURL, map[string]any{"url": inputURL})
	payload["summary"] = map[string]any{
		"apiUrl":             apiURL,
		"publicUrl":          publicURL,
		"sourceType":         "news_context",
		"reuseRestriction":   "Do not republish article text except where explicitly CC-licensed.",
		"recommendedUse":     "Cite headline/date/public URL; use short snippets only as needed for analysis.",
		"primaryEvidenceUse": false,
	}
	payload["sources"] = []map[string]string{
		{"kind": "api_request", "title": "Tagesschau article JSON", "url": apiURL},
		{"kind": "public_article", "title": "Tagesschau public article", "url": publicURL},
	}
	payload["sources"] = append(payload["sources"].([]map[string]string), defaultSources()...)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{fmt.Sprintf("tagesschau article get --url %q --limit 5", publicURL)}
	emit(payload)
	return nil
}

func compactFeedItems(data map[string]any, limit int, includeRegional bool) []map[string]any {
	var out []map[string]any
	out = append(out, compactArray(data["news"], limit)...)
	if includeRegional && len(out) < limit {
		out = append(out, compactArray(data["regional"], limit-len(out))...)
	}
	if len(out) == 0 {
		out = append(out, compactArray(data["channels"], limit)...)
	}
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func compactArray(value any, limit int) []map[string]any {
	var out []map[string]any
	items, ok := value.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, compactArticle(obj))
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactArticle(obj map[string]any) map[string]any {
	details := stringField(obj, "details")
	publicURL := firstNonEmpty(stringField(obj, "detailsweb"), stringField(obj, "detailsWeb"), stringField(obj, "shareURL"))
	if publicURL == "" && details != "" {
		_, publicURL, _ = articleURLs(details)
	}
	item := map[string]any{
		"title":         stringField(obj, "title"),
		"topline":       stringField(obj, "topline"),
		"date":          stringField(obj, "date"),
		"type":          stringField(obj, "type"),
		"firstSentence": stripHTML(stringField(obj, "firstSentence")),
		"sophoraId":     stringField(obj, "sophoraId"),
		"externalId":    stringField(obj, "externalId"),
		"details":       details,
		"detailsweb":    publicURL,
		"shareURL":      stringField(obj, "shareURL"),
		"ressort":       stringField(obj, "ressort"),
		"tags":          tagStrings(obj["tags"]),
	}
	if publicURL != "" {
		item["sourceUrl"] = publicURL
		item["nextActions"] = []string{
			fmt.Sprintf("tagesschau article get --url %q --limit 5", publicURL),
			fmt.Sprintf("tagesschau article source --url %q", publicURL),
		}
	}
	return item
}

func articleSnippets(data map[string]any, grep string, limit int) []map[string]any {
	rawContent, ok := data["content"].([]any)
	if !ok {
		return nil
	}
	var blocks []contentBlock
	for _, raw := range rawContent {
		buf, _ := json.Marshal(raw)
		var block contentBlock
		if err := json.Unmarshal(buf, &block); err == nil {
			blocks = append(blocks, block)
		}
	}
	needle := strings.ToLower(grep)
	var out []map[string]any
	for index, block := range blocks {
		if block.Type != "text" && block.Type != "headline" {
			continue
		}
		text := stripHTML(block.Value)
		if text == "" {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(text), needle) {
			continue
		}
		out = append(out, map[string]any{
			"index":   index,
			"type":    block.Type,
			"text":    truncate(text, 520),
			"matched": needle == "" || strings.Contains(strings.ToLower(text), needle),
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func articleURLs(input string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(input))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", cliError{2, "invalid_url", "expected an absolute Tagesschau URL"}
	}
	if !strings.HasSuffix(parsed.Host, "tagesschau.de") {
		return "", "", cliError{2, "invalid_url", "expected a tagesschau.de URL"}
	}
	path := parsed.Path
	if strings.HasPrefix(path, "/api2u/") {
		api := parsed.String()
		publicPath := strings.TrimPrefix(path, "/api2u")
		publicPath = strings.TrimSuffix(publicPath, ".json") + ".html"
		public := (&url.URL{Scheme: "https", Host: "www.tagesschau.de", Path: publicPath}).String()
		return api, public, nil
	}
	publicURL := (&url.URL{Scheme: "https", Host: "www.tagesschau.de", Path: path}).String()
	apiPath := strings.TrimSuffix(path, ".html") + ".json"
	api := (&url.URL{Scheme: "https", Host: "www.tagesschau.de", Path: "/api2u" + apiPath}).String()
	return api, publicURL, nil
}

func fetchRaw(requestURL, accept string) (int, string, []byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	var lastStatus int
	var lastContentType string
	var lastBody []byte
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(750 * time.Millisecond)
		}
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return 0, "", nil, err
		}
		req.Header.Set("User-Agent", defaultUserAgent)
		req.Header.Set("Accept", accept)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastStatus = resp.StatusCode
		lastContentType = resp.Header.Get("Content-Type")
		lastBody = body
		if readErr != nil {
			return lastStatus, lastContentType, lastBody, readErr
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusGatewayTimeout {
			return lastStatus, lastContentType, lastBody, nil
		}
	}
	return lastStatus, lastContentType, lastBody, lastErr
}

func envelope(command, method, requestURL string, params any) map[string]any {
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request": map[string]any{
			"method": method,
			"url":    requestURL,
			"params": params,
		},
		"summary":     map[string]any{},
		"items":       []any{},
		"sources":     []map[string]string{},
		"warnings":    []string{},
		"nextActions": []string{},
	}
}

func defaultSources() []map[string]string {
	return []map[string]string{
		{"kind": "api_docs", "title": "bundesAPI Tagesschau API documentation", "url": apiDocsURL},
		{"kind": "openapi", "title": "Tagesschau OpenAPI YAML", "url": openAPIURL},
		{"kind": "public_service", "title": "tagesschau.de", "url": baseURL + "/"},
		{"kind": "usage", "title": "Tagesschau RSS and reuse notice", "url": rssInfoURL},
		{"kind": "license", "title": "Creative Commons videos", "url": ccURL},
	}
}

func defaultWarnings() []string {
	return []string{
		"Published API documentation says not to make more than 60 requests per hour.",
		"Tagesschau content use is private/non-commercial; publication is not allowed except for content explicitly under Creative Commons.",
		"Use this as current-news context, not as the only evidence for official parliamentary, legal, fiscal, or statistical claims.",
		"Avoid reproducing long article text; cite the public article URL and use short snippets only when needed.",
	}
}

func feedCounts(data map[string]any) map[string]int {
	out := map[string]int{}
	for _, key := range []string{"news", "regional", "channels", "searchResults"} {
		if arr, ok := data[key].([]any); ok {
			out[key] = len(arr)
		}
	}
	return out
}

func nextActionsFromItems(items []map[string]any) []string {
	var actions []string
	for _, item := range items {
		if urlValue, ok := item["sourceUrl"].(string); ok && urlValue != "" {
			actions = append(actions, fmt.Sprintf("tagesschau article get --url %q --limit 5", urlValue))
		}
		if len(actions) >= 3 {
			break
		}
	}
	if len(actions) == 0 {
		actions = append(actions, "tagesschau source")
	}
	return actions
}

func parseArgs(argv []string) parsedArgs {
	parsed := parsedArgs{flags: map[string]string{}, params: url.Values{}}
	for i := 0; i < len(argv); i++ {
		token := argv[i]
		if strings.HasPrefix(token, "--") {
			key := strings.TrimPrefix(token, "--")
			if key == "param" {
				i++
				if i >= len(argv) || !strings.Contains(argv[i], "=") {
					fail(cliError{2, "invalid_param", "--param requires key=value"})
				}
				parts := strings.SplitN(argv[i], "=", 2)
				parsed.params.Add(parts[0], parts[1])
				continue
			}
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				parsed.flags[key] = argv[i+1]
				i++
			} else {
				parsed.flags[key] = "true"
			}
		} else {
			parsed.positionals = append(parsed.positionals, token)
		}
	}
	return parsed
}

func cloneValues(values url.Values) url.Values {
	out := url.Values{}
	for key, vals := range values {
		for _, value := range vals {
			out.Add(key, value)
		}
	}
	return out
}

func paramsToMap(values url.Values) map[string]any {
	out := map[string]any{}
	for key, vals := range values {
		if len(vals) == 1 {
			out[key] = vals[0]
		} else {
			out[key] = vals
		}
	}
	return out
}

func withParams(base string, params url.Values) string {
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				query.Set(key, value)
			}
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func stringField(obj map[string]any, key string) string {
	if value, ok := obj[key].(string); ok {
		return value
	}
	return ""
}

func tagStrings(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	var tags []string
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			if tag, ok := obj["tag"].(string); ok && tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	sort.Strings(tags)
	return tags
}

var tagPattern = regexp.MustCompile(`<[^>]+>`)
var whitespacePattern = regexp.MustCompile(`\s+`)

func stripHTML(value string) string {
	value = strings.ReplaceAll(value, "<br />", " ")
	value = strings.ReplaceAll(value, "<br/>", " ")
	value = tagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return stripSpace(value)
}

func stripSpace(value string) string {
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(value, " "))
}

func truncate(value string, max int) string {
	if len([]rune(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max]) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func flagBool(parsed parsedArgs, key string) bool {
	value := strings.ToLower(parsed.flags[key])
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func limitFlag(parsed parsedArgs, fallback int, max int) int {
	value := parsed.flags["limit"]
	if value == "" {
		return fallback
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		fail(cliError{2, "invalid_limit", "--limit must be an integer"})
	}
	if parsedValue < 0 {
		return 0
	}
	if parsedValue > max {
		return max
	}
	return parsedValue
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

func emit(payload any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fail(err)
	}
}

func fail(err error) {
	exitCode := 1
	code := "unexpected_error"
	message := err.Error()
	if typed, ok := err.(cliError); ok {
		exitCode = typed.exitCode
		code = typed.code
		message = typed.message
	}
	payload := map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
	os.Exit(exitCode)
}
