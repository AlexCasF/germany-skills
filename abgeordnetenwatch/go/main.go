package main

import (
	"context"
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
	appName = "abgeordnetenwatch"
	baseURL = "https://www.abgeordnetenwatch.de/api/v2"
	rootURL = "https://www.abgeordnetenwatch.de"
)

type parsedArgs struct {
	flags       map[string]string
	params      url.Values
	positionals []string
}

type apiResponse struct {
	statusCode  int
	contentType string
	body        []byte
	requestURL  string
}

var rawEntities = map[string]string{
	"parliaments":             "Parliaments",
	"parliament-periods":      "Parliament periods, legislatures, and elections",
	"politicians":             "Politicians and candidate/person profile data",
	"candidacies-mandates":    "Candidacies and mandates",
	"polls":                   "Named votes / poll metadata",
	"sidejobs":                "Side jobs and disclosed outside income",
	"sidejob-organizations":   "Organizations connected to side jobs",
	"votes":                   "Individual vote records",
	"parties":                 "Parties",
	"committees":              "Committees",
	"committee-memberships":   "Committee memberships",
	"fractions":               "Parliamentary groups/fractions",
	"electoral-lists":         "Electoral lists",
	"constituencies":          "Constituencies",
	"election-programs":       "Election programs",
	"topics":                  "Topics",
	"cities":                  "Cities used in side-job data",
	"countries":               "Countries used in side-job data",
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
		err = runDoctor()
	case match(args, "politicians", "search"):
		err = runPoliticianSearch(args[2:])
	case match(args, "politicians", "page"):
		err = runPoliticianPage(args[2:])
	case match(args, "politicians", "source"):
		err = runPoliticianSource(args[2:])
	case match(args, "politicians", "dossier"):
		err = runPoliticianDossier(args[2:])
	case match(args, "mandates", "for-politician"):
		err = runMandatesForPolitician(args[2:])
	case match(args, "sidejobs", "for-politician"):
		err = runSidejobsForPolitician(args[2:])
	case args[0] == "page":
		err = runPoliticianPage(args[1:])
	case args[0] == "source":
		err = runPoliticianSource(args[1:])
	default:
		err = runRaw(args)
	}
	if err != nil {
		var cliErr cliError
		if errors.As(err, &cliErr) {
			fail(cliErr.exitCode, cliErr.code, cliErr.message)
		}
		fail(1, "unexpected_error", err.Error())
	}
}

type cliError struct {
	exitCode int
	code     string
	message  string
}

func (e cliError) Error() string {
	return e.message
}

func printRootHelp() {
	fmt.Println(`abgeordnetenwatch -- abgeordnetenwatch.de public transparency data

Purpose
  Search and cite public politician, mandate, voting, profile, and side-job
  data from abgeordnetenwatch.de.

Use this when
  - you need public politician profiles, mandates, candidacies, or side jobs
  - you need profile-page source text and canonical profile URLs
  - you need context from abgeordnetenwatch before checking official records

Do not use this when
  - you need an official parliamentary archive; use DIP/Bundestag tools instead
  - you want to infer misconduct from side-job data alone

Fast paths
  Check API health and usage facts:
    abgeordnetenwatch doctor

  Search for a politician:
    abgeordnetenwatch politicians search --name "Mustername" --limit 3

  Build a source-backed person bundle:
    abgeordnetenwatch politicians dossier --name "Mustername" --grep Suchbegriff

Raw endpoint commands
  <entity> list|get

  Raw entities include:
    parliaments, parliament-periods, politicians, candidacies-mandates, polls,
    sidejobs, sidejob-organizations, votes, parties, committees,
    committee-memberships, fractions, electoral-lists, constituencies,
    election-programs, topics, cities, countries

Research commands
  doctor
  politicians search
  politicians page
  politicians source
  politicians dossier
  mandates for-politician
  sidejobs for-politician

Output guarantees
  Research commands emit JSON with status, request, retrievedAt, sources,
  warnings, and nextActions. Raw endpoint commands return upstream JSON.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "politicians dossier":
		fmt.Println(`abgeordnetenwatch politicians dossier

What it does
  Builds a compact evidence bundle for one politician with API profile data,
  mandates, side jobs, source URLs, page metadata, optional profile-page
  snippets, warnings, and next actions.

Inputs
  --id       Politician ID, e.g. <politician-id>
  --name     Search name, e.g. "Mustername"
  --url      Public profile URL, e.g. https://www.abgeordnetenwatch.de/profile/example
  --grep     Optional page-text snippet term
  --limit    Max records per related collection, default 10

Examples
  abgeordnetenwatch politicians dossier --name "Mustername" --grep Suchbegriff
  abgeordnetenwatch politicians dossier --id <politician-id> --limit 5

Next action
  Cross-check official parliamentary claims with DIP/Bundestag tools when needed.`)
	case "politicians page":
		fmt.Println(`abgeordnetenwatch politicians page

What it does
  Fetches a public abgeordnetenwatch profile page and extracts canonical URL,
  title, description, OpenData/profile ID hints, text preview, and grep snippets.

Inputs
  --id       Politician ID
  --name     Politician search name
  --url      Public profile URL
  --grep     Optional snippet term

Example
  abgeordnetenwatch politicians page --name "Mustername" --grep Suchbegriff`)
	case "politicians search":
		fmt.Println(`abgeordnetenwatch politicians search

What it does
  Searches politicians by name with a small default limit and normalized source URLs.

Inputs
  --name        Full or partial name
  --first-name  Optional first-name contains filter
  --last-name   Optional last-name contains filter
  --party       Optional party label/short-name contains filter
  --limit       Default 5

Example
  abgeordnetenwatch politicians search --name "Mustername" --limit 3`)
	default:
		printRootHelp()
	}
}

func runDoctor() error {
	resp, err := apiJSON("/politicians", url.Values{"range_end": []string{"1"}})
	if err != nil {
		return err
	}
	meta := object(resp["meta"])
	apiInfo := object(meta["abgeordnetenwatch_api"])
	result := object(meta["result"])
	payload := envelope("doctor", "https://www.abgeordnetenwatch.de/api/v2/politicians?range_end=1")
	payload["summary"] = map[string]any{
		"authRequired": false,
		"baseUrl":      baseURL,
		"apiVersion":   apiInfo["version"],
		"licence":      apiInfo["licence"],
		"licenceLink":  apiInfo["licence_link"],
		"documentation": []string{
			"https://www.abgeordnetenwatch.de/api",
			"https://www.abgeordnetenwatch.de/api/response",
			"https://www.abgeordnetenwatch.de/api/version-changelog/aktuell",
		},
		"publishedRateLimit": "not found in official API docs",
		"resultLimit":        "default 100; range_end/pager_limit up to 1000 per official docs",
		"health": map[string]any{
			"status":      meta["status"],
			"count":       result["count"],
			"sampleTotal": result["total"],
		},
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = []string{
		"No exact request-per-minute rate limit was found; use conservative limits.",
		"abgeordnetenwatch is a transparency platform, not an official parliamentary archive.",
		"Side-job data is evidence of disclosures, not proof of misconduct by itself.",
	}
	payload["nextActions"] = []string{
		`abgeordnetenwatch politicians search --name "Mustername" --limit 3`,
		`abgeordnetenwatch politicians dossier --id <politician-id> --grep Suchbegriff`,
	}
	emit(payload)
	return nil
}

func runRaw(args []string) error {
	if len(args) < 2 {
		return cliError{2, "unknown_command", "expected <entity> list|get"}
	}
	entity := args[0]
	action := args[1]
	if _, ok := rawEntities[entity]; !ok {
		return cliError{2, "unknown_entity", "unknown entity: " + entity}
	}
	parsed := parseArgs(args[2:])
	params := normalizeParams(parsed)
	switch action {
	case "list":
		resp, err := apiGet("/"+entity, params)
		if err != nil {
			return err
		}
		fmt.Println(string(resp.body))
		return nil
	case "get":
		id := parsed.flags["id"]
		if id == "" && len(parsed.positionals) > 0 {
			id = parsed.positionals[0]
		}
		if id == "" {
			return cliError{2, "missing_id", entity + " get requires --id"}
		}
		resp, err := apiGet("/"+entity+"/"+url.PathEscape(id), params)
		if err != nil {
			return err
		}
		fmt.Println(string(resp.body))
		return nil
	default:
		return cliError{2, "unknown_action", "unknown action for " + entity + ": " + action}
	}
}

func runPoliticianSearch(args []string) error {
	parsed := parseArgs(args)
	params := normalizeParams(parsed)
	limit := limitFlag(parsed, 5, 50)
	params.Set("range_end", strconv.Itoa(limit))
	if name := strings.TrimSpace(parsed.flags["name"]); name != "" {
		params.Set("label[cn]", name)
	}
	if first := strings.TrimSpace(parsed.flags["first-name"]); first != "" {
		params.Set("first_name[cn]", first)
	}
	if last := strings.TrimSpace(parsed.flags["last-name"]); last != "" {
		params.Set("last_name[cn]", last)
	}
	if party := strings.TrimSpace(parsed.flags["party"]); party != "" {
		params.Set("party[entity.label][cn]", party)
	}
	data, err := apiJSON("/politicians", params)
	if err != nil {
		return err
	}
	items := summarizeList(data, limit)
	payload := envelope("politicians search", baseURL+"/politicians?"+params.Encode())
	payload["summary"] = map[string]any{
		"search":      searchSummary(parsed),
		"returned":    len(items),
		"total":       total(data),
		"clientLimit": limit,
	}
	payload["items"] = items
	payload["sources"] = []map[string]string{{"kind": "api", "title": "Politicians endpoint", "url": baseURL + "/politicians"}}
	payload["warnings"] = []string{"Search results are public transparency data; verify official parliamentary records separately when needed."}
	payload["nextActions"] = nextForPoliticianItems(items)
	emit(payload)
	return nil
}

func runPoliticianSource(args []string) error {
	record, _, err := resolvePolitician(args)
	if err != nil {
		return err
	}
	summary := map[string]any{
		"record":  summarizePolitician(record),
		"sources": politicianSources(record),
	}
	payload := envelope("politicians source", apiURLFromRecord(record))
	payload["summary"] = summary
	payload["sources"] = politicianSources(record)
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf(`abgeordnetenwatch politicians page --id %v`, record["id"]),
		fmt.Sprintf(`abgeordnetenwatch politicians dossier --id %v`, record["id"]),
	}
	emit(payload)
	return nil
}

func runPoliticianPage(args []string) error {
	record, rawURL, err := resolvePolitician(args)
	if err != nil {
		return err
	}
	parsed := parseArgs(args)
	profileURL := rawURL
	if profileURL == "" {
		profileURL = stringValue(record["abgeordnetenwatch_url"])
	}
	if profileURL == "" {
		return cliError{1, "missing_profile_url", "politician record has no profile URL"}
	}
	page, err := fetchPage(profileURL, parsed.flags["grep"])
	if err != nil {
		return err
	}
	payload := envelope("politicians page", page["url"].(string))
	payload["summary"] = page
	payload["sources"] = []map[string]string{{"kind": "profile", "title": "Public profile page", "url": page["url"].(string)}}
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = []string{fmt.Sprintf(`abgeordnetenwatch politicians dossier --id %v`, record["id"])}
	emit(payload)
	return nil
}

func runPoliticianDossier(args []string) error {
	parsed := parseArgs(args)
	limit := limitFlag(parsed, 10, 50)
	record, _, err := resolvePolitician(args)
	if err != nil {
		return err
	}
	id := intString(record["id"])
	mandates, err := fetchCollection("/candidacies-mandates", url.Values{
		"politician": []string{id},
		"range_end":  []string{strconv.Itoa(limit)},
	}, limit)
	if err != nil {
		return err
	}
	sidejobs, err := sidejobsForMandates(mandates, limit)
	if err != nil {
		return err
	}
	var page map[string]any
	if profileURL := stringValue(record["abgeordnetenwatch_url"]); profileURL != "" {
		page, _ = fetchPage(profileURL, parsed.flags["grep"])
	}
	summary := map[string]any{
		"politician":       summarizePolitician(record),
		"mandateCount":     len(mandates),
		"mandates":         summarizeRecords(mandates, limit),
		"sidejobCount":     len(sidejobs),
		"sidejobs":         summarizeRecords(sidejobs, limit),
		"sidejobIncomeSum": sumNumeric(sidejobs, "income"),
		"profilePage":      page,
		"sourceCategories": []string{"api", "public-profile-page", "mandates", "sidejobs"},
	}
	payload := envelope("politicians dossier", apiURLFromRecord(record))
	payload["summary"] = summary
	payload["sources"] = politicianSources(record)
	payload["warnings"] = append(standardWarnings(),
		"Side-job income fields may be partial and depend on disclosed Bundestag data as processed by abgeordnetenwatch.",
		"Do not equate outside income or mandates with corruption without independent evidence.")
	payload["nextActions"] = []string{
		fmt.Sprintf(`abgeordnetenwatch sidejobs for-politician --id %s --limit %d`, id, limit),
		fmt.Sprintf(`abgeordnetenwatch politicians page --id %s --grep Suchbegriff`, id),
	}
	emit(payload)
	return nil
}

func runMandatesForPolitician(args []string) error {
	parsed := parseArgs(args)
	limit := limitFlag(parsed, 10, 50)
	record, _, err := resolvePolitician(args)
	if err != nil {
		return err
	}
	id := intString(record["id"])
	mandates, err := fetchCollection("/candidacies-mandates", url.Values{
		"politician": []string{id},
		"range_end":  []string{strconv.Itoa(limit)},
	}, limit)
	if err != nil {
		return err
	}
	payload := envelope("mandates for-politician", baseURL+"/candidacies-mandates?politician="+url.QueryEscape(id))
	payload["summary"] = map[string]any{
		"politician": summarizePolitician(record),
		"returned":   len(mandates),
	}
	payload["items"] = summarizeRecords(mandates, limit)
	payload["sources"] = []map[string]string{{"kind": "api", "title": "Candidacies/mandates endpoint", "url": baseURL + "/candidacies-mandates"}}
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = []string{fmt.Sprintf(`abgeordnetenwatch sidejobs for-politician --id %s`, id)}
	emit(payload)
	return nil
}

func runSidejobsForPolitician(args []string) error {
	parsed := parseArgs(args)
	limit := limitFlag(parsed, 10, 50)
	record, _, err := resolvePolitician(args)
	if err != nil {
		return err
	}
	id := intString(record["id"])
	mandates, err := fetchCollection("/candidacies-mandates", url.Values{
		"politician": []string{id},
		"range_end":  []string{strconv.Itoa(limit)},
	}, limit)
	if err != nil {
		return err
	}
	sidejobs, err := sidejobsForMandates(mandates, limit)
	if err != nil {
		return err
	}
	payload := envelope("sidejobs for-politician", baseURL+"/sidejobs")
	payload["summary"] = map[string]any{
		"politician":  summarizePolitician(record),
		"mandates":    len(mandates),
		"returned":    len(sidejobs),
		"incomeSum":   sumNumeric(sidejobs, "income"),
		"clientLimit": limit,
	}
	payload["items"] = summarizeRecords(sidejobs, limit)
	payload["sources"] = []map[string]string{{"kind": "api", "title": "Sidejobs endpoint", "url": baseURL + "/sidejobs"}}
	payload["warnings"] = append(standardWarnings(), "Side-job data is disclosure data; interpret categories and income fields cautiously.")
	payload["nextActions"] = []string{fmt.Sprintf(`abgeordnetenwatch politicians dossier --id %s --grep Suchbegriff`, id)}
	emit(payload)
	return nil
}

func resolvePolitician(args []string) (map[string]any, string, error) {
	parsed := parseArgs(args)
	if rawURL := strings.TrimSpace(parsed.flags["url"]); rawURL != "" {
		id := idFromProfileURL(rawURL)
		if id == "" {
			id = idFromShortURL(rawURL)
		}
		if id == "" {
			page, err := fetchPage(rawURL, "")
			if err == nil {
				if pageID := stringValue(page["politicianId"]); pageID != "" {
					id = pageID
				}
			}
		}
		if id != "" {
			rec, err := getPolitician(id)
			return rec, rawURL, err
		}
		return nil, rawURL, cliError{2, "unsupported_profile_url", "could not infer politician ID from URL; use --id or --name"}
	}
	if id := strings.TrimSpace(parsed.flags["id"]); id != "" {
		rec, err := getPolitician(id)
		return rec, "", err
	}
	if name := strings.TrimSpace(parsed.flags["name"]); name != "" {
		params := url.Values{"label[cn]": []string{name}, "range_end": []string{"1"}}
		data, err := apiJSON("/politicians", params)
		if err != nil {
			return nil, "", err
		}
		rows := dataList(data)
		if len(rows) == 0 {
			return nil, "", cliError{1, "not_found", "no politician found for name: " + name}
		}
		id := intString(rows[0]["id"])
		rec, err := getPolitician(id)
		return rec, "", err
	}
	return nil, "", cliError{2, "missing_input", "provide --id, --name, or --url"}
}

func getPolitician(id string) (map[string]any, error) {
	data, err := apiJSON("/politicians/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	rec := object(data["data"])
	if len(rec) == 0 {
		return nil, cliError{1, "not_found", "politician not found: " + id}
	}
	return rec, nil
}

func fetchCollection(path string, params url.Values, limit int) ([]map[string]any, error) {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("range_end") == "" && params.Get("pager_limit") == "" {
		params.Set("range_end", strconv.Itoa(limit))
	}
	data, err := apiJSON(path, params)
	if err != nil {
		return nil, err
	}
	rows := dataList(data)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func sidejobsForMandates(mandates []map[string]any, limit int) ([]map[string]any, error) {
	var out []map[string]any
	seen := map[string]bool{}
	for _, mandate := range mandates {
		if len(out) >= limit {
			break
		}
		id := intString(mandate["id"])
		if id == "" {
			continue
		}
		rows, err := fetchCollection("/sidejobs", url.Values{
			"mandates":  []string{id},
			"range_end": []string{strconv.Itoa(limit)},
		}, limit)
		if err != nil {
			continue
		}
		for _, row := range rows {
			rid := intString(row["id"])
			if rid != "" && seen[rid] {
				continue
			}
			seen[rid] = true
			out = append(out, row)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func fetchPage(rawURL string, grep string) (map[string]any, error) {
	u, err := validateAWURL(rawURL)
	if err != nil {
		return nil, err
	}
	resp, err := httpGet(u, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if err != nil {
		return nil, err
	}
	body := string(resp.body)
	text := stripHTML(body)
	page := map[string]any{
		"url":          resp.requestURL,
		"title":        firstMatch(body, `(?is)<title[^>]*>(.*?)</title>`),
		"canonical":    attrMatch(body, "link", "rel", "canonical", "href"),
		"shortlink":    attrMatch(body, "link", "rel", "shortlink", "href"),
		"description":  metaContent(body, "description"),
		"politicianId": politicianIDFromHTML(body),
		"textLength":   len(text),
		"textPreview":  trim(text, 1200),
	}
	if grep != "" {
		page["grep"] = grep
		page["snippets"] = snippets(text, grep, 10)
	}
	return page, nil
}

func apiJSON(path string, params url.Values) (map[string]any, error) {
	resp, err := apiGet(path, params)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(resp.body, &data); err != nil {
		return nil, cliError{1, "invalid_json", "API did not return JSON: " + err.Error()}
	}
	return data, nil
}

func apiGet(path string, params url.Values) (*apiResponse, error) {
	u := baseURL + path
	if params != nil && len(params) > 0 {
		u += "?" + params.Encode()
	}
	return httpGet(u, "application/json")
}

func httpGet(rawURL string, accept string) (*apiResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, cliError{2, "bad_url", err.Error()}
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", appName+" (+https://github.com/AlexCasF/germany-skills)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, cliError{1, "request_failed", err.Error()}
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if readErr != nil {
		return nil, cliError{1, "read_failed", readErr.Error()}
	}
	if resp.StatusCode >= 400 {
		return nil, cliError{1, "request_failed", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, trim(string(body), 500))}
	}
	return &apiResponse{statusCode: resp.StatusCode, contentType: resp.Header.Get("Content-Type"), body: body, requestURL: rawURL}, nil
}

func parseArgs(args []string) parsedArgs {
	out := parsedArgs{flags: map[string]string{}, params: url.Values{}}
	for i := 0; i < len(args); {
		token := args[i]
		if token == "--param" || token == "--query" {
			if i+1 >= len(args) {
				out.flags["__bad_param__"] = token
				i++
				continue
			}
			addParam(out.params, args[i+1])
			i += 2
			continue
		}
		if strings.HasPrefix(token, "--") {
			key := strings.TrimPrefix(token, "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				out.flags[key] = args[i+1]
				i += 2
			} else {
				out.flags[key] = "true"
				i++
			}
			continue
		}
		out.positionals = append(out.positionals, token)
		i++
	}
	return out
}

func normalizeParams(parsed parsedArgs) url.Values {
	params := url.Values{}
	for key, values := range parsed.params {
		for _, value := range values {
			params.Add(key, value)
		}
	}
	if parsed.flags["limit"] != "" && params.Get("range_end") == "" && params.Get("pager_limit") == "" {
		params.Set("range_end", parsed.flags["limit"])
	}
	if parsed.flags["page"] != "" {
		params.Set("page", parsed.flags["page"])
	}
	if parsed.flags["pager-limit"] != "" {
		params.Set("pager_limit", parsed.flags["pager-limit"])
	}
	if parsed.flags["related-data"] != "" {
		params.Set("related_data", parsed.flags["related-data"])
	}
	return params
}

func addParam(params url.Values, raw string) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return
	}
	params.Add(parts[0], parts[1])
}

func limitFlag(parsed parsedArgs, def int, max int) int {
	raw := parsed.flags["limit"]
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return def
	}
	if value > max {
		return max
	}
	return value
}

func summarizeList(data map[string]any, limit int) []map[string]any {
	return summarizeRecords(dataList(data), limit)
}

func summarizeRecords(rows []map[string]any, limit int) []map[string]any {
	var out []map[string]any
	for _, row := range rows {
		out = append(out, summarizeRecord(row))
		if len(out) >= limit {
			break
		}
	}
	return out
}

func summarizeRecord(row map[string]any) map[string]any {
	if stringValue(row["entity_type"]) == "politician" {
		return summarizePolitician(row)
	}
	out := map[string]any{}
	for _, key := range []string{"id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "type", "start_date", "end_date", "income", "income_level", "income_total", "interval", "data_change_date", "job_title_extra", "additional_information"} {
		if value, ok := row[key]; ok {
			out[key] = value
		}
	}
	if org := object(row["sidejob_organization"]); len(org) > 0 {
		out["sidejob_organization"] = summarizeReference(org)
	}
	if party := object(row["party"]); len(party) > 0 {
		out["party"] = summarizeReference(party)
	}
	if period := object(row["parliament_period"]); len(period) > 0 {
		out["parliament_period"] = summarizeReference(period)
	}
	if politician := object(row["politician"]); len(politician) > 0 {
		out["politician"] = summarizeReference(politician)
	}
	if url := stringValue(row["api_url"]); url != "" {
		out["sources"] = []map[string]string{{"kind": "api", "title": "API record", "url": url}}
	}
	return out
}

func summarizePolitician(row map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "entity_type", "label", "api_url", "abgeordnetenwatch_url", "first_name", "last_name", "sex", "year_of_birth", "education", "occupation", "statistic_questions", "statistic_questions_answered", "ext_id_bundestagsverwaltung", "qid_wikidata"} {
		if value, ok := row[key]; ok {
			out[key] = value
		}
	}
	if party := object(row["party"]); len(party) > 0 {
		out["party"] = summarizeReference(party)
	}
	out["sources"] = politicianSources(row)
	return out
}

func summarizeReference(row map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "entity_type", "label", "api_url", "abgeordnetenwatch_url"} {
		if value, ok := row[key]; ok {
			out[key] = value
		}
	}
	return out
}

func politicianSources(row map[string]any) []map[string]string {
	var sources []map[string]string
	if api := stringValue(row["api_url"]); api != "" {
		sources = append(sources, map[string]string{"kind": "api", "title": "API record", "url": api})
	}
	if profile := stringValue(row["abgeordnetenwatch_url"]); profile != "" {
		sources = append(sources, map[string]string{"kind": "profile", "title": "Public profile", "url": profile})
	}
	if id := intString(row["id"]); id != "" {
		sources = append(sources, map[string]string{"kind": "api", "title": "Mandates for politician", "url": baseURL + "/candidacies-mandates?politician=" + url.QueryEscape(id)})
	}
	return sources
}

func dataList(data map[string]any) []map[string]any {
	raw, ok := data["data"].([]any)
	if !ok {
		return nil
	}
	var rows []map[string]any
	for _, item := range raw {
		if row := object(item); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func total(data map[string]any) any {
	meta := object(data["meta"])
	result := object(meta["result"])
	return result["total"]
}

func searchSummary(parsed parsedArgs) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"name", "first-name", "last-name", "party", "limit"} {
		if parsed.flags[key] != "" {
			out[key] = parsed.flags[key]
		}
	}
	return out
}

func nextForPoliticianItems(items []map[string]any) []string {
	var out []string
	for _, item := range items {
		id := intString(item["id"])
		if id == "" {
			continue
		}
		out = append(out,
			fmt.Sprintf("abgeordnetenwatch politicians dossier --id %s", id),
			fmt.Sprintf("abgeordnetenwatch politicians page --id %s --grep Suchbegriff", id),
		)
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func envelope(command string, requestURL string) map[string]any {
	return map[string]any{
		"tool":        appName,
		"command":     command,
		"status":      "ok",
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request": map[string]any{
			"method":     "GET",
			"url":        requestURL,
			"redactions": []string{},
		},
		"summary":     map[string]any{},
		"sources":     []map[string]string{},
		"warnings":    []string{},
		"nextActions": []string{},
	}
}

func defaultSources() []map[string]string {
	return []map[string]string{
		{"kind": "documentation", "title": "API documentation", "url": "https://www.abgeordnetenwatch.de/api"},
		{"kind": "documentation", "title": "API response format", "url": "https://www.abgeordnetenwatch.de/api/response"},
		{"kind": "documentation", "title": "API changelog", "url": "https://www.abgeordnetenwatch.de/api/version-changelog/aktuell"},
		{"kind": "license", "title": "CC0 1.0", "url": "https://creativecommons.org/publicdomain/zero/1.0/deed.de"},
	}
}

func standardWarnings() []string {
	return []string{
		"abgeordnetenwatch is a transparency platform, not an official parliamentary archive.",
		"Use official Bundestag/DIP records when the exact official parliamentary record matters.",
		"No exact API rate limit was found in official docs; keep requests bounded.",
	}
}

func validateAWURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", cliError{2, "bad_url", err.Error()}
	}
	if parsed.Host != "www.abgeordnetenwatch.de" && parsed.Host != "abgeordnetenwatch.de" {
		return "", cliError{2, "unsupported_url", "URL must belong to abgeordnetenwatch.de"}
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	if parsed.Host == "abgeordnetenwatch.de" {
		parsed.Host = "www.abgeordnetenwatch.de"
	}
	return parsed.String(), nil
}

func idFromProfileURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(parsed.Path, "/politician/") {
		return strings.TrimPrefix(parsed.Path, "/politician/")
	}
	return ""
}

func idFromShortURL(raw string) string {
	re := regexp.MustCompile(`/politician/([0-9]+)`)
	match := re.FindStringSubmatch(raw)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func politicianIDFromHTML(body string) string {
	for _, pattern := range []string{
		`currentPath":"politician/([0-9]+)"`,
		`/politician/([0-9]+)`,
		`view_args":"([0-9]+)"`,
	} {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(body)
		if len(match) == 2 {
			return match[1]
		}
	}
	return ""
}

func stripHTML(raw string) string {
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<svg[^>]*>.*?</svg>`)
	raw = reScript.ReplaceAllString(raw, " ")
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	raw = reTags.ReplaceAllString(raw, " ")
	return clean(html.UnescapeString(raw))
}

func snippets(text string, term string, limit int) []string {
	if term == "" {
		return nil
	}
	lower := strings.ToLower(text)
	needle := strings.ToLower(term)
	var out []string
	start := 0
	for len(out) < limit {
		idx := strings.Index(lower[start:], needle)
		if idx < 0 {
			break
		}
		idx += start
		left := maxInt(0, idx-240)
		right := minInt(len(text), idx+len(term)+240)
		out = append(out, clean(text[left:right]))
		start = idx + len(term)
	}
	return out
}

func firstMatch(raw string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return clean(html.UnescapeString(match[1]))
}

func attrMatch(raw string, tag string, attrName string, attrValue string, wanted string) string {
	re := regexp.MustCompile(`(?is)<` + tag + `[^>]*>`)
	for _, m := range re.FindAllString(raw, -1) {
		if strings.Contains(strings.ToLower(m), strings.ToLower(attrName+`="`+attrValue+`"`)) || strings.Contains(strings.ToLower(m), strings.ToLower(attrName+`='`+attrValue+`'`)) {
			if value := attrValueFromTag(m, wanted); value != "" {
				return html.UnescapeString(value)
			}
		}
	}
	return ""
}

func metaContent(raw string, name string) string {
	re := regexp.MustCompile(`(?is)<meta[^>]*>`)
	for _, m := range re.FindAllString(raw, -1) {
		if strings.Contains(strings.ToLower(m), `name="`+strings.ToLower(name)+`"`) || strings.Contains(strings.ToLower(m), `property="`+strings.ToLower(name)+`"`) {
			return clean(html.UnescapeString(attrValueFromTag(m, "content")))
		}
	}
	return ""
}

func attrValueFromTag(tag string, attr string) string {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(attr) + `\s*=\s*["']([^"']+)["']`)
	match := re.FindStringSubmatch(tag)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func apiURLFromRecord(row map[string]any) string {
	if value := stringValue(row["api_url"]); value != "" {
		return value
	}
	if id := intString(row["id"]); id != "" {
		return baseURL + "/politicians/" + id
	}
	return baseURL
}

func object(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func intString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func sumNumeric(rows []map[string]any, key string) float64 {
	var sum float64
	for _, row := range rows {
		switch v := row[key].(type) {
		case float64:
			sum += v
		case int:
			sum += float64(v)
		case json.Number:
			if f, err := v.Float64(); err == nil {
				sum += f
			}
		}
	}
	return sum
}

func clean(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func trim(value string, n int) string {
	value = clean(value)
	if len(value) <= n {
		return value
	}
	return value[:n] + "..."
}

func match(args []string, parts ...string) bool {
	if len(args) < len(parts) {
		return false
	}
	for i := range parts {
		if args[i] != parts[i] {
			return false
		}
	}
	return true
}

func isHelp(value string) bool {
	return value == "--help" || value == "-h" || value == "help"
}

func emit(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(value)
}

func fail(exitCode int, code string, message string) {
	payload := map[string]any{
		"tool":        appName,
		"status":      "error",
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	emit(payload)
	os.Exit(exitCode)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
