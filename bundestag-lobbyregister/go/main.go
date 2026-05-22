package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	appName     = "bundestag-lobbyregister"
	baseURL     = "https://api.lobbyregister.bundestag.de/rest/v2"
	publicURL   = "https://www.lobbyregister.bundestag.de"
	rawV1URL = "https://www.lobbyregister.bundestag.de/sucheDetailJson"
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

type cliError struct {
	exitCode int
	code     string
	message  string
}

func (e cliError) Error() string {
	return e.message
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
	case args[0] == "statistics":
		err = runStatistics(args[1:])
	case args[0] == "search":
		err = runSearch(args[1:])
	case match(args, "entry", "get"):
		err = runEntryGet(args[2:])
	case match(args, "entry", "source"):
		err = runEntrySource(args[2:])
	case match(args, "entry", "dossier"):
		err = runEntryDossier(args[2:])
	case match(args, "financial", "summary"):
		err = runFinancialSummary(args[2:])
	case match(args, "statements", "list"):
		err = runStatementsList(args[2:])
	case match(args, "raw", "search"):
		err = runV1Search(args[2:])
	default:
		err = cliError{2, "unknown_command", "unknown command path: " + strings.Join(args, " ")}
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
	fmt.Println(`bundestag-lobbyregister -- Bundestag Lobbyregister research CLI

Purpose
  Search and cite public lobby-register data for interests represented
  toward the German Bundestag and Federal Government.

Use this when
  - you need official lobby-register entries, finance ranges, statements,
    regulatory projects, clients, donors, contracts, or public source URLs
  - you need a compact evidence trail for one registered organization/person
  - you need to inspect political-finance or lobbying relationships

Do not use this when
  - you need official parliamentary proceedings; use DIP/Bundestag tools
  - you need company ownership records outside the register
  - you want to infer misconduct from registration data alone

Fast paths
  Check auth and endpoint health:
    bundestag-lobbyregister doctor

  Search safely:
    bundestag-lobbyregister search --term "Musterverband" --limit 3

  Build an evidence bundle:
    bundestag-lobbyregister entry dossier --register-number <register-number> --grep "Foerderung"

  Normalize financial data:
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
  raw search    Preserves the old unauthenticated sucheDetailJson wrapper.

Auth
  Prefer LOBBYREGISTER_API_KEY from the environment.
  --apikey still works for local compatibility and is redacted from output.

Output
  Research commands emit JSON envelopes with status, request, retrievedAt,
  summary/items, sources, warnings, and nextActions. Broad commands default
  to small limits.`)
}

func printHelp(path []string) {
	joined := strings.Join(path, " ")
	switch joined {
	case "entry dossier":
		fmt.Println(`bundestag-lobbyregister entry dossier

Builds a compact evidence bundle for one register entry.

Inputs
  --register-number <register-number>    Exact register number
  --name "Musterverband ..."   Search first, then use the first match
  --grep "term"                Return snippets from embedded statement text
  --include-raw                Include the full upstream detail JSON

Examples
  bundestag-lobbyregister entry dossier --register-number <register-number> --grep "Laerm"
  bundestag-lobbyregister entry dossier --name "Musterverband"`)
	case "search":
		fmt.Println(`bundestag-lobbyregister search

Safe free-text search over register entries. The upstream endpoint returns
large full-detail records, so this command defaults to compact summaries and a
small --limit.

Examples
  bundestag-lobbyregister search --term "Energie" --limit 5
  bundestag-lobbyregister search --term "\"Musterverband\"" --include-raw`)
	case "entry get":
		fmt.Println(`bundestag-lobbyregister entry get

Fetch one official register entry by register number.

Examples
  bundestag-lobbyregister entry get --register-number <register-number>
  bundestag-lobbyregister entry get --register-number <register-number> --version 6 --include-raw`)
	case "financial summary":
		fmt.Println(`bundestag-lobbyregister financial summary

Fetch one register entry and normalize financial expense ranges, donations,
membership fees, public allowances, annual-report links, and disclosure caveats.

Example
  bundestag-lobbyregister financial summary --register-number <register-number>`)
	default:
		printRootHelp()
	}
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	key := apiKey(parsed)
	payload := envelope("doctor", baseURL+"/statistics/registerentries?format=json")
	payload["summary"] = map[string]any{
		"authRequired":           true,
		"apiKeyConfigured":       key != "",
		"apiKeySource":           keySource(parsed),
		"baseUrl":                baseURL,
		"publicRegisterUrl":      publicURL,
		"openApiYaml":            baseURL + "/R2.21-de.yaml",
		"swaggerUi":              baseURL + "/swagger-ui/",
		"termsAndOpenDataPage":   publicURL + "/informationen-und-hilfe/open-data-1049716",
		"publishedRateLimit":     "not found in official docs reviewed; use small limits and retry politely",
		"recommendedDefaultLimit": 5,
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = standardWarnings()
	if key == "" {
		payload["warnings"] = appendAny(payload["warnings"], "LOBBYREGISTER_API_KEY is not configured; live API calls will fail.")
		payload["nextActions"] = []string{"Set LOBBYREGISTER_API_KEY, then run: bundestag-lobbyregister statistics"}
		emit(payload)
		return nil
	}
	resp, err := apiJSON("/statistics/registerentries", url.Values{"format": {"json"}}, key)
	if err != nil {
		payload["status"] = "error"
		payload["summary"].(map[string]any)["health"] = map[string]any{"ok": false, "error": err.Error()}
		emit(payload)
		return nil
	}
	stats := asMap(resp["json"])
	payload["summary"].(map[string]any)["health"] = map[string]any{
		"ok":                true,
		"statusCode":        resp["statusCode"],
		"sourceDate":        stringAt(stats, "sourceDate"),
		"totalLobbyists":    anyAt(stats, "lobbyists", "totalNumber"),
		"activeLobbyists":   anyAt(stats, "lobbyists", "active", "number"),
		"inactiveLobbyists": anyAt(stats, "lobbyists", "inactive", "number"),
	}
	payload["nextActions"] = []string{
		`bundestag-lobbyregister search --term "Musterverband" --limit 3`,
		"bundestag-lobbyregister entry dossier --register-number <register-number>",
	}
	emit(payload)
	return nil
}

func runStatistics(argv []string) error {
	parsed := parseArgs(argv)
	key, err := requireKey(parsed)
	if err != nil {
		return err
	}
	resp, err := apiJSON("/statistics/registerentries", url.Values{"format": {"json"}}, key)
	if err != nil {
		return err
	}
	stats := asMap(resp["json"])
	payload := envelope("statistics", resp["requestURL"].(string))
	payload["summary"] = map[string]any{
		"source":            stringAt(stats, "source"),
		"sourceDate":        stringAt(stats, "sourceDate"),
		"totalLobbyists":    anyAt(stats, "lobbyists", "totalNumber"),
		"activeLobbyists":   anyAt(stats, "lobbyists", "active", "number"),
		"inactiveLobbyists": anyAt(stats, "lobbyists", "inactive", "number"),
		"peopleInvolved":    anyAt(stats, "lobbyists", "peopleInvolvedInLobbyistWork", "totalNumber"),
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = stats
	}
	payload["sources"] = defaultSources()
	payload["nextActions"] = []string{`bundestag-lobbyregister search --term "Energie" --limit 5`}
	emit(payload)
	return nil
}

func runSearch(argv []string) error {
	parsed := parseArgs(argv)
	key, err := requireKey(parsed)
	if err != nil {
		return err
	}
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], parsed.flags["name"])
	if term == "" {
		return cliError{2, "missing_term", "search requires --term, --q, or --name"}
	}
	limit := limitFlag(parsed, 5, 25)
	params := url.Values{"format": {"json"}, "q": {term}}
	if cursor := parsed.flags["cursor"]; cursor != "" {
		params.Set("cursor", cursor)
	}
	resp, err := apiJSON("/registerentries", params, key)
	if err != nil {
		return err
	}
	data := asMap(resp["json"])
	results := asSlice(data["results"])
	items := []any{}
	for i, entry := range results {
		if i >= limit {
			break
		}
		items = append(items, summarizeEntry(asMap(entry)))
	}
	payload := envelope("search", resp["requestURL"].(string))
	payload["summary"] = map[string]any{
		"query":            term,
		"returnedByApi":    anyAt(data, "resultCount"),
		"totalResultCount": anyAt(data, "totalResultCount"),
		"limitApplied":     limit,
		"cursorPresent":    stringAt(data, "cursor") != "",
		"sourceDate":       stringAt(data, "sourceDate"),
	}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = searchNextActions(items)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runEntryGet(argv []string) error {
	parsed := parseArgs(argv)
	entry, respURL, err := getEntryFromArgs(parsed)
	if err != nil {
		return err
	}
	payload := envelope("entry get", respURL)
	payload["summary"] = summarizeEntry(entry)
	payload["sources"] = entrySources(entry)
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = nextActionsForEntry(entry)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = entry
	}
	emit(payload)
	return nil
}

func runEntrySource(argv []string) error {
	parsed := parseArgs(argv)
	entry, respURL, err := getEntryFromArgs(parsed)
	if err != nil {
		return err
	}
	payload := envelope("entry source", respURL)
	payload["summary"] = map[string]any{
		"registerNumber": stringAt(entry, "registerNumber"),
		"name":           stringAt(entry, "lobbyistIdentity", "name"),
		"version":        anyAt(entry, "registerEntryDetails", "version"),
		"sourceDate":     stringAt(entry, "sourceDate"),
	}
	payload["sources"] = entrySources(entry)
	payload["nextActions"] = nextActionsForEntry(entry)
	emit(payload)
	return nil
}

func runEntryDossier(argv []string) error {
	parsed := parseArgs(argv)
	entry, respURL, err := getEntryFromArgs(parsed)
	if err != nil {
		return err
	}
	payload := envelope("entry dossier", respURL)
	payload["summary"] = summarizeEntry(entry)
	payload["financial"] = financialBlock(entry)
	payload["regulatoryProjects"] = compactProjects(entry, limitFlag(parsed, 5, 20))
	payload["statements"] = compactStatements(entry, parsed.flags["grep"], limitFlag(parsed, 5, 20))
	payload["sources"] = entrySources(entry)
	payload["warnings"] = standardWarnings()
	payload["nextActions"] = nextActionsForEntry(entry)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = entry
	}
	emit(payload)
	return nil
}

func runFinancialSummary(argv []string) error {
	parsed := parseArgs(argv)
	entry, respURL, err := getEntryFromArgs(parsed)
	if err != nil {
		return err
	}
	payload := envelope("financial summary", respURL)
	payload["summary"] = map[string]any{
		"registerNumber": stringAt(entry, "registerNumber"),
		"name":           stringAt(entry, "lobbyistIdentity", "name"),
		"sourceDate":     stringAt(entry, "sourceDate"),
	}
	payload["financial"] = financialBlock(entry)
	payload["sources"] = entrySources(entry)
	payload["warnings"] = append(standardWarnings(), "Financial ranges are register disclosures, not audited findings by this tool.")
	payload["nextActions"] = nextActionsForEntry(entry)
	emit(payload)
	return nil
}

func runStatementsList(argv []string) error {
	parsed := parseArgs(argv)
	entry, respURL, err := getEntryFromArgs(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, 10, 50)
	payload := envelope("statements list", respURL)
	payload["summary"] = map[string]any{
		"registerNumber":   stringAt(entry, "registerNumber"),
		"name":             stringAt(entry, "lobbyistIdentity", "name"),
		"statementsPresent": anyAt(entry, "statements", "statementsPresent"),
		"statementsCount":   anyAt(entry, "statements", "statementsCount"),
		"limitApplied":      limit,
	}
	payload["items"] = compactStatements(entry, parsed.flags["grep"], limit)
	payload["sources"] = entrySources(entry)
	payload["warnings"] = append(standardWarnings(), "Statement text may include copyrighted material; quote only short excerpts.")
	payload["nextActions"] = nextActionsForEntry(entry)
	emit(payload)
	return nil
}

func runV1Search(argv []string) error {
	parsed := parseArgs(argv)
	params := parsed.params
	for k, v := range parsed.flags {
		if k != "include-raw" && k != "timeout" {
			params.Set(k, v)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutFlag(parsed))
	defer cancel()
	u, _ := url.Parse(rawV1URL)
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return cliError{1, "http_error", fmt.Sprintf("raw V1 endpoint returned HTTP %d", resp.StatusCode)}
	}
	os.Stdout.Write(body)
	if len(body) == 0 || body[len(body)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func getEntryFromArgs(parsed parsedArgs) (map[string]any, string, error) {
	key, err := requireKey(parsed)
	if err != nil {
		return nil, "", err
	}
	registerNumber := firstNonEmpty(parsed.flags["register-number"], parsed.flags["registerNumber"], parsed.flags["id"])
	if registerNumber == "" && parsed.flags["name"] != "" {
		found, _, err := searchFirst(parsed.flags["name"], key)
		if err != nil {
			return nil, "", err
		}
		registerNumber = stringAt(found, "registerNumber")
	}
	if registerNumber == "" {
		return nil, "", cliError{2, "missing_register_number", "requires --register-number or --name"}
	}
	if !regexp.MustCompile(`^R[0-9]{6}$`).MatchString(registerNumber) {
		return nil, "", cliError{2, "invalid_register_number", "register number must look like <register-number>"}
	}
	path := "/registerentries/" + url.PathEscape(registerNumber)
	if version := parsed.flags["version"]; version != "" {
		path += "/" + url.PathEscape(version)
	}
	resp, err := apiJSON(path, url.Values{"format": {"json"}}, key)
	if err != nil {
		return nil, "", err
	}
	return asMap(resp["json"]), resp["requestURL"].(string), nil
}

func searchFirst(term string, key string) (map[string]any, string, error) {
	resp, err := apiJSON("/registerentries", url.Values{"format": {"json"}, "q": {term}}, key)
	if err != nil {
		return nil, "", err
	}
	data := asMap(resp["json"])
	results := asSlice(data["results"])
	if len(results) == 0 {
		return nil, "", cliError{1, "not_found", "no register entry found for name: " + term}
	}
	return asMap(results[0]), resp["requestURL"].(string), nil
}

func summarizeEntry(entry map[string]any) map[string]any {
	return map[string]any{
		"registerNumber":          stringAt(entry, "registerNumber"),
		"name":                    stringAt(entry, "lobbyistIdentity", "name"),
		"identity":                stringAt(entry, "lobbyistIdentity", "identity"),
		"legalForm":               firstNonEmpty(stringAt(entry, "lobbyistIdentity", "legalForm", "de"), stringAt(entry, "lobbyistIdentity", "legalForm", "en")),
		"activeLobbyist":          anyAt(entry, "accountDetails", "activeLobbyist"),
		"firstPublicationDate":    stringAt(entry, "accountDetails", "firstPublicationDate"),
		"lastUpdateDate":          stringAt(entry, "accountDetails", "lastUpdateDate"),
		"version":                 anyAt(entry, "registerEntryDetails", "version"),
		"detailsPageUrl":          stringAt(entry, "registerEntryDetails", "detailsPageUrl"),
		"pdfUrl":                  stringAt(entry, "registerEntryDetails", "pdfUrl"),
		"financialExpensesEuro":   anyAt(entry, "financialExpenses", "financialExpensesEuro"),
		"financialFiscalYear":     fiscalYear(entry, "financialExpenses"),
		"employeeFTE":             anyAt(entry, "employeesInvolvedInLobbying", "employeeFTE"),
		"fieldsOfInterest":        labelsFromArray(sliceAt(entry, "activitiesAndInterests", "fieldsOfInterest"), 10),
		"activityDescriptionHint": truncate(stringAt(entry, "activitiesAndInterests", "activityDescription"), 280),
		"mainFundingSources":      labelsFromArray(sliceAt(entry, "mainFundingSources", "mainFundingSources"), 8),
		"totalDonationsEuro":      anyAt(entry, "donators", "totalDonationsEuro"),
		"totalMembershipFees":     anyAt(entry, "membershipFees", "totalMembershipFees"),
		"publicAllowancesPresent": anyAt(entry, "publicAllowances", "publicAllowancesPresent"),
		"regulatoryProjectsCount": anyAt(entry, "regulatoryProjects", "regulatoryProjectsCount"),
		"statementsCount":         anyAt(entry, "statements", "statementsCount"),
		"contractsCount":          anyAt(entry, "contracts", "contractsCount"),
	}
}

func financialBlock(entry map[string]any) map[string]any {
	return map[string]any{
		"financialExpenses": map[string]any{
			"fiscalYear": fiscalYear(entry, "financialExpenses"),
			"rangeEuro":  anyAt(entry, "financialExpenses", "financialExpensesEuro"),
		},
		"mainFundingSources": labelsFromArray(sliceAt(entry, "mainFundingSources", "mainFundingSources"), 20),
		"publicAllowances":   anyAt(entry, "publicAllowances"),
		"donations": map[string]any{
			"fiscalYear": fiscalYear(entry, "donators"),
			"totalEuro":  anyAt(entry, "donators", "totalDonationsEuro"),
			"items":      compactNamedItems(sliceAt(entry, "donators", "donators"), 20),
		},
		"membershipFees": map[string]any{
			"fiscalYear":             fiscalYear(entry, "membershipFees"),
			"totalEuro":              anyAt(entry, "membershipFees", "totalMembershipFees"),
			"individualContributors": compactNamedItems(sliceAt(entry, "membershipFees", "individualContributors"), 20),
		},
		"annualReport": map[string]any{
			"exists": anyAt(entry, "annualReports", "annualReportLastFiscalYearExists"),
			"pdfUrl": stringAt(entry, "annualReports", "annualReportPdfUrl"),
		},
	}
}

func compactProjects(entry map[string]any, limit int) []any {
	projects := sliceAt(entry, "regulatoryProjects", "regulatoryProjects")
	out := []any{}
	for i, p := range projects {
		if i >= limit {
			break
		}
		pm := asMap(p)
		out = append(out, map[string]any{
			"number":           stringAt(pm, "regulatoryProjectNumber"),
			"title":            stringAt(pm, "title"),
			"descriptionHint":  truncate(stringAt(pm, "description"), 320),
			"affectedLaws":     labelsFromArray(sliceAt(pm, "affectedLaws"), 8),
			"fieldsOfInterest": labelsFromArray(sliceAt(pm, "fieldsOfInterest"), 8),
			"projectUrl":       stringAt(pm, "projectUrl"),
		})
	}
	return out
}

func compactStatements(entry map[string]any, grep string, limit int) []any {
	statements := sliceAt(entry, "statements", "statements")
	out := []any{}
	for _, s := range statements {
		if len(out) >= limit {
			break
		}
		sm := asMap(s)
		text := stringAt(sm, "text", "text")
		item := map[string]any{
			"regulatoryProjectNumber": stringAt(sm, "regulatoryProjectNumber"),
			"regulatoryProjectTitle":  stringAt(sm, "regulatoryProjectTitle"),
			"pdfUrl":                  stringAt(sm, "pdfUrl"),
			"pdfPageCount":            anyAt(sm, "pdfPageCount"),
			"recipientGroups":         anyAt(sm, "recipientGroups"),
			"textPreview":             truncate(text, 420),
		}
		if grep != "" {
			snips := snippets(text, grep, 3)
			if len(snips) == 0 {
				continue
			}
			item["snippets"] = snips
		}
		out = append(out, item)
	}
	return out
}

func entrySources(entry map[string]any) []any {
	sources := defaultSources()
	if u := stringAt(entry, "registerEntryDetails", "detailsPageUrl"); u != "" {
		sources = append(sources, map[string]any{"title": "Public detail page", "url": u, "kind": "public-page"})
	}
	if u := stringAt(entry, "registerEntryDetails", "pdfUrl"); u != "" {
		sources = append(sources, map[string]any{"title": "Public PDF export", "url": u, "kind": "pdf"})
	}
	if u := stringAt(entry, "annualReports", "annualReportPdfUrl"); u != "" {
		sources = append(sources, map[string]any{"title": "Annual report PDF", "url": u, "kind": "pdf"})
	}
	for _, st := range sliceAt(entry, "statements", "statements") {
		sm := asMap(st)
		if u := stringAt(sm, "pdfUrl"); u != "" {
			sources = append(sources, map[string]any{"title": "Statement PDF: " + stringAt(sm, "regulatoryProjectTitle"), "url": u, "kind": "statement-pdf"})
		}
	}
	return sources
}

func defaultSources() []any {
	return []any{
		map[string]any{"title": "Bundestag Lobbyregister", "url": publicURL, "kind": "official-register"},
		map[string]any{"title": "Open Data/API page", "url": publicURL + "/informationen-und-hilfe/open-data-1049716", "kind": "terms"},
		map[string]any{"title": "Swagger UI", "url": baseURL + "/swagger-ui/", "kind": "api-docs"},
		map[string]any{"title": "OpenAPI YAML", "url": baseURL + "/R2.21-de.yaml", "kind": "openapi"},
	}
}

func standardWarnings() []string {
	return []string{
		"API calls require an API key; this tool redacts keys from normalized output.",
		"Register disclosures describe published self-reported register data; corroborate contentious claims with additional official sources.",
		"Use small limits for broad searches; the upstream search endpoint returns full-detail records.",
	}
}

func nextActionsForEntry(entry map[string]any) []string {
	rn := stringAt(entry, "registerNumber")
	if rn == "" {
		return []string{}
	}
	return []string{
		"bundestag-lobbyregister entry source --register-number " + rn,
		"bundestag-lobbyregister financial summary --register-number " + rn,
		"bundestag-lobbyregister statements list --register-number " + rn + " --grep <term>",
	}
}

func searchNextActions(items []any) []string {
	out := []string{}
	for _, item := range items {
		m := asMap(item)
		if rn := stringAt(m, "registerNumber"); rn != "" {
			out = append(out, "bundestag-lobbyregister entry dossier --register-number "+rn)
		}
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func apiJSON(path string, params url.Values, key string) (map[string]any, error) {
	resp, err := apiGet(path, params, key)
	if err != nil {
		return nil, err
	}
	var data any
	dec := json.NewDecoder(bytes.NewReader(resp.body))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return map[string]any{
		"json":        data,
		"statusCode":  resp.statusCode,
		"contentType": resp.contentType,
		"requestURL":  resp.requestURL,
	}, nil
}

func apiGet(path string, params url.Values, key string) (apiResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("format") == "" {
		params.Set("format", "json")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	u, err := url.Parse(baseURL + path)
	if err != nil {
		return apiResponse{}, err
	}
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Authorization", "ApiKey "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return apiResponse{}, cliError{1, "http_error", fmt.Sprintf("HTTP %d from Lobbyregister API: %s", resp.StatusCode, truncate(string(body), 300))}
	}
	return apiResponse{statusCode: resp.StatusCode, contentType: resp.Header.Get("Content-Type"), body: body, requestURL: sanitizeURL(u.String())}, nil
}

func parseArgs(argv []string) parsedArgs {
	out := parsedArgs{flags: map[string]string{}, params: url.Values{}, positionals: []string{}}
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--param" && i+1 < len(argv) {
			parts := strings.SplitN(argv[i+1], "=", 2)
			if len(parts) == 2 {
				out.params.Add(parts[0], parts[1])
			}
			i++
			continue
		}
		if strings.HasPrefix(arg, "--param=") {
			parts := strings.SplitN(strings.TrimPrefix(arg, "--param="), "=", 2)
			if len(parts) == 2 {
				out.params.Add(parts[0], parts[1])
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			nameValue := strings.TrimPrefix(arg, "--")
			if strings.Contains(nameValue, "=") {
				parts := strings.SplitN(nameValue, "=", 2)
				out.flags[parts[0]] = parts[1]
				continue
			}
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				out.flags[nameValue] = argv[i+1]
				i++
			} else {
				out.flags[nameValue] = "true"
			}
			continue
		}
		out.positionals = append(out.positionals, arg)
	}
	return out
}

func requireKey(parsed parsedArgs) (string, error) {
	key := apiKey(parsed)
	if key == "" {
		return "", cliError{2, "missing_api_key", "set LOBBYREGISTER_API_KEY or pass --apikey"}
	}
	return key, nil
}

func apiKey(parsed parsedArgs) string {
	if parsed.flags["apikey"] != "" {
		return parsed.flags["apikey"]
	}
	return os.Getenv("LOBBYREGISTER_API_KEY")
}

func keySource(parsed parsedArgs) string {
	if parsed.flags["apikey"] != "" {
		return "flag:redacted"
	}
	if os.Getenv("LOBBYREGISTER_API_KEY") != "" {
		return "env:LOBBYREGISTER_API_KEY"
	}
	return "missing"
}

func envelope(command string, requestURL string) map[string]any {
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request": map[string]any{
			"method":            "GET",
			"url":               sanitizeURL(requestURL),
			"authConfigured":    true,
			"redactedHeaders":   []string{"Authorization"},
			"redactedQueryKeys": []string{"apikey"},
		},
	}
}

func emit(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func fail(exitCode int, code string, message string) {
	emit(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error": map[string]any{
			"code":    code,
			"message": redact(message),
		},
	})
	os.Exit(exitCode)
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
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

func limitFlag(parsed parsedArgs, def int, max int) int {
	raw := parsed.flags["limit"]
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func timeoutFlag(parsed parsedArgs) time.Duration {
	raw := parsed.flags["timeout"]
	if raw == "" {
		return 60 * time.Second
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 60 * time.Second
	}
	return time.Duration(n) * time.Second
}

func flagBool(parsed parsedArgs, name string) bool {
	v := strings.ToLower(parsed.flags[name])
	return v == "true" || v == "1" || v == "yes"
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{}
}

func anyAt(m map[string]any, path ...string) any {
	var cur any = m
	for _, p := range path {
		cm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = cm[p]
	}
	return cur
}

func stringAt(m map[string]any, path ...string) string {
	v := anyAt(m, path...)
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}

func sliceAt(m map[string]any, path ...string) []any {
	return asSlice(anyAt(m, path...))
}

func fiscalYear(entry map[string]any, block string) map[string]any {
	return map[string]any{
		"finished": anyAt(entry, block, "relatedFiscalYearFinished"),
		"start":    stringAt(entry, block, "relatedFiscalYearStart"),
		"end":      stringAt(entry, block, "relatedFiscalYearEnd"),
	}
}

func labelsFromArray(items []any, limit int) []string {
	out := []string{}
	for _, item := range items {
		if len(out) >= limit {
			break
		}
		m := asMap(item)
		label := firstNonEmpty(stringAt(m, "de"), stringAt(m, "title"), stringAt(m, "name"), stringAt(m, "en"), stringAt(m, "code"))
		if label != "" {
			out = append(out, label)
		}
	}
	return out
}

func compactNamedItems(items []any, limit int) []any {
	out := []any{}
	for _, item := range items {
		if len(out) >= limit {
			break
		}
		m := asMap(item)
		out = append(out, map[string]any{
			"name":     firstNonEmpty(stringAt(m, "name"), stringAt(m, "lastName")),
			"rangeEuro": firstNonEmpty(stringAt(m, "amount", "from"), stringAt(m, "totalAmount", "from")),
			"rawHint":  truncate(mustJSON(m), 240),
		})
	}
	return out
}

func snippets(text string, term string, limit int) []string {
	if text == "" || term == "" {
		return []string{}
	}
	lower := strings.ToLower(text)
	needle := strings.ToLower(term)
	out := []string{}
	start := 0
	for len(out) < limit {
		idx := strings.Index(lower[start:], needle)
		if idx < 0 {
			break
		}
		idx += start
		from := idx - 160
		if from < 0 {
			from = 0
		}
		to := idx + len(term) + 160
		if to > len(text) {
			to = len(text)
		}
		out = append(out, strings.TrimSpace(collapseSpace(text[from:to])))
		start = idx + len(term)
	}
	return out
}

func truncate(s string, max int) string {
	s = collapseSpace(s)
	if len(s) <= max {
		return s
	}
	if max < 4 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return redact(raw)
	}
	q := u.Query()
	if q.Has("apikey") {
		q.Set("apikey", "REDACTED")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func redact(s string) string {
	re := regexp.MustCompile(`(?i)(apikey=)[^&\s]+|ApiKey\s+[A-Za-z0-9._-]+|(--apikey\s+)[A-Za-z0-9._-]+`)
	return re.ReplaceAllStringFunc(s, func(part string) string {
		if strings.Contains(strings.ToLower(part), "apikey=") {
			return "apikey=REDACTED"
		}
		if strings.HasPrefix(strings.ToLower(part), "--apikey") {
			return "--apikey REDACTED"
		}
		return "ApiKey REDACTED"
	})
}

func appendAny(v any, item any) []any {
	out := []any{}
	switch t := v.(type) {
	case []any:
		out = append(out, t...)
	case []string:
		for _, s := range t {
			out = append(out, s)
		}
	}
	out = append(out, item)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
