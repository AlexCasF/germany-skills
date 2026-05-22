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
	appName = "destatis"
	baseURL = "https://www-genesis.destatis.de/genesisWS/rest/2020"
	uiURL   = "https://www-genesis.destatis.de/datenbank/online"
	docsURL = "https://www.destatis.de/DE/Service/OpenData/genesis-api-webservice-oberflaeche.html"
)

var legacyPaths = map[string]string{
	"catalogue statistics":  "/catalogue/statistics",
	"catalogue tables":      "/catalogue/tables",
	"catalogue variables":   "/catalogue/variables",
	"metadata table":        "/metadata/table",
	"metadata timeseries":   "/metadata/timeseries",
	"data table":            "/data/table",
	"data timeseries":       "/data/timeseries",
	"find search":           "/find/find",
}

type parsedArgs struct {
	flags       map[string]string
	params      url.Values
	positionals []string
}

type credentials struct {
	username string
	password string
	source   string
	guest    bool
}

type apiResponse struct {
	statusCode int
	body       []byte
	requestURL string
}

type cliError struct {
	exitCode int
	code     string
	message  string
}

func (e cliError) Error() string { return e.message }

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
	case args[0] == "search":
		err = runSearch(args[1:])
	case match(args, "table", "source"):
		err = runTableSource(args[2:])
	case match(args, "table", "dossier"):
		err = runTableDossier(args[2:])
	case match(args, "table", "sample"):
		err = runTableSample(args[2:])
	case match(args, "timeseries", "dossier"):
		err = runTimeseriesDossier(args[2:])
	case match(args, "variables", "explain"):
		err = runVariablesExplain(args[2:])
	default:
		err = runLegacy(args)
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
	fmt.Println(`destatis -- Destatis GENESIS-Online statistics CLI

Purpose
  Search and retrieve official German statistics from Destatis GENESIS-Online.

Use this when
  - you need official German statistical tables, time series, variables, or metadata
  - you need to find table/statistic codes before requesting data
  - you need source-aware statistical evidence with labels, caveats, and small samples

Do not use this when
  - you need regional ArcGIS indicator layers; use Deutschlandatlas or Regionalatlas tools
  - you need parliamentary records; use DIP/Bundestag tools
  - you only have a claim but no statistical concept yet; search first, then inspect metadata

Fast paths
  Check credentials and endpoint behavior:
    destatis doctor

  Search official table/statistic catalogue:
    destatis search --term "Arbeitslose" --limit 5

  Show source/citation metadata for a known table:
    destatis table source --name 12211-0900

  Build a cautious table bundle:
    destatis table dossier --name 12211-0900

Legacy endpoint commands
  destatis catalogue statistics
  destatis catalogue tables
  destatis catalogue variables
  destatis metadata table
  destatis metadata timeseries
  destatis data table
  destatis data timeseries
  destatis find search

Research commands
  doctor
  search
  table source
  table dossier
  table sample
  timeseries dossier
  variables explain

Auth
  Prefer DESTATIS_USERNAME and DESTATIS_PASSWORD from the environment.
  --username and --password still work and are redacted from output.
  If no credentials are configured, the CLI uses GAST/GAST for public discovery.

Output
  Research commands emit JSON envelopes with status, request, retrievedAt,
  summary/items, sources, warnings, and nextActions. Legacy commands return
  upstream JSON on success and structured JSON errors on failure.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "table dossier":
		fmt.Println(`destatis table dossier

Build a cautious evidence bundle for one GENESIS table code. With full
credentials it tries metadata and a small data sample; with guest credentials it
returns source metadata and structured warnings if protected endpoints return 401.

Examples
  destatis table dossier --name 12211-0900
  destatis table dossier --name 12211-0900 --include-raw`)
	case "search":
		fmt.Println(`destatis search

Friendly alias for the GENESIS find endpoint. Uses GAST/GAST by default when no
credentials are configured and keeps output compact.

Example
  destatis search --term "Arbeitslose" --limit 5`)
	default:
		printRootHelp()
	}
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	payload := envelope("doctor", "/helloworld/logincheck", nil, cred)
	payload["summary"] = map[string]any{
		"baseUrl":              baseURL,
		"webUi":                uiURL,
		"docs":                 docsURL,
		"authConfigured":       !cred.guest || os.Getenv("DESTATIS_USERNAME") != "" || os.Getenv("DESTATIS_PASSWORD") != "",
		"credentialSource":     cred.source,
		"guestFallbackEnabled": cred.guest,
		"publishedRateLimit":   "not found in official Destatis docs reviewed; use small pagelength values and avoid parallel broad requests",
		"license":              "Datenlizenz Deutschland - Namensnennung - Version 2.0 for GENESIS-Online usage per Destatis Open Data page",
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = standardWarnings(cred)

	login, err := apiPost("/helloworld/logincheck", url.Values{}, cred)
	if err != nil {
		payload["status"] = "error"
		payload["summary"].(map[string]any)["health"] = map[string]any{"ok": false, "error": redact(err.Error())}
		emit(payload)
		return nil
	}
	loginJSON, _ := parseJSON(login.body)
	payload["summary"].(map[string]any)["health"] = map[string]any{
		"ok":         true,
		"statusCode": login.statusCode,
		"message":    stringAt(loginJSON, "Status"),
		"username":   redactUsername(stringAt(loginJSON, "Username")),
	}

	findResp, err := apiPost("/find/find", url.Values{"term": {"Arbeitslose"}, "category": {"all"}, "pagelength": {"1"}, "language": {"de"}}, cred)
	if err == nil {
		findJSON, _ := parseJSON(findResp.body)
		payload["summary"].(map[string]any)["findCheck"] = map[string]any{
			"ok":          true,
			"status":      anyAt(findJSON, "Status"),
			"tablesFound": len(asSlice(findJSON["Tables"])),
		}
	} else {
		payload["summary"].(map[string]any)["findCheck"] = map[string]any{"ok": false, "error": redact(err.Error())}
	}
	payload["nextActions"] = []string{
		`destatis search --term "Arbeitslose" --limit 5`,
		"destatis table source --name 12211-0900",
	}
	emit(payload)
	return nil
}

func runSearch(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], parsed.flags["selection"])
	if term == "" {
		return cliError{2, "missing_term", "search requires --term"}
	}
	limit := limitFlag(parsed, 5, 25)
	params := cloneValues(parsed.params)
	params.Set("term", term)
	params.Set("category", firstNonEmpty(parsed.flags["category"], params.Get("category"), "all"))
	params.Set("pagelength", strconv.Itoa(limit))
	params.Set("language", firstNonEmpty(parsed.flags["language"], params.Get("language"), "de"))
	resp, err := apiPost("/find/find", params, cred)
	if err != nil {
		return err
	}
	data, err := parseJSON(resp.body)
	if err != nil {
		return err
	}
	items := compactFind(data, limit)
	payload := envelope("search", "/find/find", params, cred)
	payload["summary"] = map[string]any{
		"term":         term,
		"limitApplied": limit,
		"status":       anyAt(data, "Status"),
		"statistics":   len(asSlice(data["Statistics"])),
		"tables":       len(asSlice(data["Tables"])),
		"timeseries":   len(asSlice(data["Timeseries"])),
	}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = standardWarnings(cred)
	payload["nextActions"] = nextActionsForFind(items)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runTableSource(argv []string) error {
	parsed := parseArgs(argv)
	name, err := requiredName(parsed)
	if err != nil {
		return err
	}
	payload := envelope("table source", "/metadata/table", url.Values{"name": {name}}, resolveCredentials(parsed))
	payload["summary"] = tableSourceSummary(name)
	payload["sources"] = sourcesForTable(name)
	payload["warnings"] = []string{"Source URLs identify official GENESIS locations; table availability and metadata detail can depend on credentials."}
	payload["nextActions"] = []string{
		"destatis table dossier --name " + name,
		"destatis metadata table --param name=" + name,
	}
	emit(payload)
	return nil
}

func runTableDossier(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	name, err := requiredName(parsed)
	if err != nil {
		return err
	}
	payload := envelope("table dossier", "/metadata/table", url.Values{"name": {name}}, cred)
	payload["summary"] = tableSourceSummary(name)
	payload["sources"] = sourcesForTable(name)
	payload["warnings"] = standardWarnings(cred)
	payload["nextActions"] = []string{
		"destatis table sample --name " + name,
		"destatis variables explain --table " + name,
	}

	metaParams := url.Values{"name": {name}, "language": {firstNonEmpty(parsed.flags["language"], "de")}}
	metaResp, metaErr := apiPost("/metadata/table", metaParams, cred)
	if metaErr != nil {
		payload["metadata"] = map[string]any{"available": false, "error": redact(metaErr.Error())}
		payload["warnings"] = appendAny(payload["warnings"], "Metadata request failed; guest credentials can be insufficient for metadata/data endpoints.")
	} else {
		metaJSON, _ := parseJSON(metaResp.body)
		payload["metadata"] = summarizeDestatisPayload(metaJSON)
		if flagBool(parsed, "include-raw") {
			payload["rawMetadata"] = metaJSON
		}
	}

	if flagBool(parsed, "sample") {
		sampleParams := url.Values{"name": {name}, "area": {"all"}, "compress": {"true"}, "transpose": {"false"}, "format": {"ffcsv"}, "language": {firstNonEmpty(parsed.flags["language"], "de")}}
		sampleResp, sampleErr := apiPost("/data/table", sampleParams, cred)
		if sampleErr != nil {
			payload["sample"] = map[string]any{"available": false, "error": redact(sampleErr.Error())}
			payload["warnings"] = appendAny(payload["warnings"], "Data sample request failed; use personal GENESIS credentials for protected data endpoints.")
		} else {
			payload["sample"] = map[string]any{"available": true, "preview": truncate(string(sampleResp.body), 1200)}
		}
	}
	emit(payload)
	return nil
}

func runTableSample(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	name, err := requiredName(parsed)
	if err != nil {
		return err
	}
	params := cloneValues(parsed.params)
	params.Set("name", name)
	params.Set("area", firstNonEmpty(parsed.flags["area"], params.Get("area"), "all"))
	params.Set("format", firstNonEmpty(parsed.flags["format"], params.Get("format"), "ffcsv"))
	params.Set("compress", firstNonEmpty(parsed.flags["compress"], params.Get("compress"), "true"))
	params.Set("transpose", firstNonEmpty(parsed.flags["transpose"], params.Get("transpose"), "false"))
	params.Set("language", firstNonEmpty(parsed.flags["language"], params.Get("language"), "de"))
	resp, err := apiPost("/data/table", params, cred)
	payload := envelope("table sample", "/data/table", params, cred)
	payload["summary"] = tableSourceSummary(name)
	payload["sources"] = sourcesForTable(name)
	payload["warnings"] = standardWarnings(cred)
	if err != nil {
		payload["status"] = "partial"
		payload["sample"] = map[string]any{"available": false, "error": redact(err.Error())}
		emit(payload)
		return nil
	}
	payload["sample"] = map[string]any{"available": true, "preview": truncate(string(resp.body), 1600)}
	emit(payload)
	return nil
}

func runTimeseriesDossier(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	name, err := requiredName(parsed)
	if err != nil {
		return err
	}
	params := url.Values{"name": {name}, "language": {firstNonEmpty(parsed.flags["language"], "de")}}
	payload := envelope("timeseries dossier", "/metadata/timeseries", params, cred)
	payload["summary"] = map[string]any{"name": name, "kind": "timeseries", "webUi": uiURL + "/timeseries/" + url.PathEscape(name)}
	payload["sources"] = defaultSources()
	payload["warnings"] = standardWarnings(cred)
	resp, err := apiPost("/metadata/timeseries", params, cred)
	if err != nil {
		payload["status"] = "partial"
		payload["metadata"] = map[string]any{"available": false, "error": redact(err.Error())}
	} else {
		data, _ := parseJSON(resp.body)
		payload["metadata"] = summarizeDestatisPayload(data)
		if flagBool(parsed, "include-raw") {
			payload["rawMetadata"] = data
		}
	}
	emit(payload)
	return nil
}

func runVariablesExplain(argv []string) error {
	parsed := parseArgs(argv)
	cred := resolveCredentials(parsed)
	table := firstNonEmpty(parsed.flags["table"], parsed.flags["name"], parsed.flags["code"])
	if table == "" {
		return cliError{2, "missing_table", "variables explain requires --table"}
	}
	params := url.Values{"name": {table}, "language": {firstNonEmpty(parsed.flags["language"], "de")}}
	payload := envelope("variables explain", "/catalogue/tables2variable", params, cred)
	payload["summary"] = map[string]any{"table": table, "purpose": "discover variables/dimensions connected to a GENESIS table"}
	payload["sources"] = sourcesForTable(table)
	payload["warnings"] = standardWarnings(cred)
	resp, err := apiPost("/catalogue/tables2variable", params, cred)
	if err != nil {
		payload["status"] = "partial"
		payload["variables"] = map[string]any{"available": false, "error": redact(err.Error())}
	} else {
		data, _ := parseJSON(resp.body)
		payload["variables"] = summarizeDestatisPayload(data)
		if flagBool(parsed, "include-raw") {
			payload["rawVariables"] = data
		}
	}
	payload["nextActions"] = []string{"destatis table dossier --name " + table}
	emit(payload)
	return nil
}

func runLegacy(args []string) error {
	if len(args) < 2 {
		return cliError{2, "unknown_command", "expected command group and action"}
	}
	key := args[0] + " " + args[1]
	path, ok := legacyPaths[key]
	if !ok {
		return cliError{2, "unknown_command", "unknown command path: " + strings.Join(args, " ")}
	}
	parsed := parseArgs(args[2:])
	cred := resolveCredentials(parsed)
	params := cloneValues(parsed.params)
	for k, v := range parsed.flags {
		if !reservedFlag(k) {
			params.Set(k, v)
		}
	}
	if key == "find search" && params.Get("term") == "" && parsed.flags["selection"] != "" {
		params.Set("term", parsed.flags["selection"])
	}
	if params.Get("language") == "" {
		params.Set("language", "de")
	}
	if params.Get("pagelength") == "" {
		params.Set("pagelength", strconv.Itoa(limitFlag(parsed, 10, 100)))
	}
	resp, err := apiPost(path, params, cred)
	if err != nil {
		return err
	}
	os.Stdout.Write(resp.body)
	if len(resp.body) == 0 || resp.body[len(resp.body)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func apiPost(path string, params url.Values, cred credentials) (apiResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("username", cred.username)
	params.Set("password", cred.password)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, strings.NewReader(params.Encode()))
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return apiResponse{}, cliError{1, "http_error", fmt.Sprintf("HTTP %d from Destatis GENESIS API: %s", resp.StatusCode, truncate(string(body), 280))}
	}
	return apiResponse{statusCode: resp.StatusCode, body: body, requestURL: baseURL + path}, nil
}

func compactFind(data map[string]any, limit int) []any {
	items := []any{}
	add := func(kind string, rows []any) {
		for _, row := range rows {
			if len(items) >= limit {
				return
			}
			m := asMap(row)
			code := stringAt(m, "Code")
			item := map[string]any{
				"kind":    kind,
				"code":    code,
				"title":   stringAt(m, "Content"),
				"time":    stringAt(m, "Time"),
				"cubes":   stringAt(m, "Cubes"),
				"sources": sourceLinks(kind, code),
			}
			items = append(items, item)
		}
	}
	add("statistic", asSlice(data["Statistics"]))
	add("table", asSlice(data["Tables"]))
	add("timeseries", asSlice(data["Timeseries"]))
	add("cube", asSlice(data["Cubes"]))
	return items
}

func summarizeDestatisPayload(data map[string]any) map[string]any {
	return map[string]any{
		"status":     anyAt(data, "Status"),
		"ident":      anyAt(data, "Ident"),
		"parameters": redactParamMap(asMap(data["Parameter"])),
		"objectKeys": objectKeys(data),
		"preview":    truncate(mustJSON(data), 1400),
	}
}

func tableSourceSummary(name string) map[string]any {
	return map[string]any{
		"name":       name,
		"kind":       "table",
		"apiBaseUrl": baseURL,
		"webUi":      uiURL + "/table/" + url.PathEscape(name),
		"license":    "Datenlizenz Deutschland - Namensnennung - Version 2.0 per Destatis Open Data page",
	}
}

func sourceLinks(kind string, code string) []any {
	if code == "" {
		return defaultSources()
	}
	if kind == "table" {
		return sourcesForTable(code)
	}
	if kind == "statistic" {
		return []any{
			map[string]any{"title": "GENESIS statistic page", "url": uiURL + "/statistic/" + url.PathEscape(code), "kind": "web-ui"},
			map[string]any{"title": "GENESIS REST API", "url": baseURL, "kind": "api"},
		}
	}
	return defaultSources()
}

func sourcesForTable(name string) []any {
	return []any{
		map[string]any{"title": "GENESIS table page", "url": uiURL + "/table/" + url.PathEscape(name), "kind": "web-ui"},
		map[string]any{"title": "GENESIS metadata endpoint", "url": baseURL + "/metadata/table", "kind": "api"},
		map[string]any{"title": "GENESIS data endpoint", "url": baseURL + "/data/table", "kind": "api"},
		map[string]any{"title": "Destatis GENESIS API/Webservice page", "url": docsURL, "kind": "docs"},
	}
}

func defaultSources() []any {
	return []any{
		map[string]any{"title": "Destatis GENESIS API/Webservices page", "url": docsURL, "kind": "docs"},
		map[string]any{"title": "GENESIS-Online database", "url": uiURL, "kind": "web-ui"},
		map[string]any{"title": "GENESIS REST base URL", "url": baseURL, "kind": "api"},
	}
}

func standardWarnings(cred credentials) []string {
	warnings := []string{
		"Use small pagelength values for discovery; inspect metadata before requesting data.",
		"Preserve table/statistic codes, units, time periods, and source dates in final answers.",
		"Credentials are redacted from normalized output and errors.",
	}
	if cred.guest {
		warnings = append(warnings, "Using GAST/GAST fallback: discovery works, but metadata/data endpoints may return 401; configure DESTATIS_USERNAME and DESTATIS_PASSWORD for full access.")
	}
	return warnings
}

func nextActionsForFind(items []any) []string {
	actions := []string{}
	for _, item := range items {
		m := asMap(item)
		code := stringAt(m, "code")
		if code == "" {
			continue
		}
		switch stringAt(m, "kind") {
		case "table":
			actions = append(actions, "destatis table dossier --name "+code)
		case "timeseries":
			actions = append(actions, "destatis timeseries dossier --name "+code)
		case "statistic":
			actions = append(actions, "destatis catalogue tables --param name="+code)
		}
		if len(actions) >= 5 {
			break
		}
	}
	return actions
}

func parseArgs(argv []string) parsedArgs {
	out := parsedArgs{flags: map[string]string{}, params: url.Values{}, positionals: []string{}}
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--param" && i+1 < len(argv) {
			addParam(out.params, argv[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "--param=") {
			addParam(out.params, strings.TrimPrefix(arg, "--param="))
			continue
		}
		if strings.HasPrefix(arg, "--") {
			nameValue := strings.TrimPrefix(arg, "--")
			if strings.Contains(nameValue, "=") {
				parts := strings.SplitN(nameValue, "=", 2)
				out.flags[parts[0]] = parts[1]
			} else if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
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

func addParam(values url.Values, raw string) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) == 2 {
		values.Add(parts[0], parts[1])
	}
}

func resolveCredentials(parsed parsedArgs) credentials {
	username := firstNonEmpty(parsed.flags["username"], os.Getenv("DESTATIS_USERNAME"), "GAST")
	password := firstNonEmpty(parsed.flags["password"], os.Getenv("DESTATIS_PASSWORD"), "GAST")
	source := "guest:GAST"
	if parsed.flags["username"] != "" || parsed.flags["password"] != "" {
		source = "flags:redacted"
	} else if os.Getenv("DESTATIS_USERNAME") != "" || os.Getenv("DESTATIS_PASSWORD") != "" {
		source = "env:DESTATIS_USERNAME/DESTATIS_PASSWORD"
	}
	return credentials{username: username, password: password, source: source, guest: username == "GAST" && password == "GAST"}
}

func envelope(command string, path string, params url.Values, cred credentials) map[string]any {
	request := map[string]any{
		"method":           "POST",
		"url":              baseURL + path,
		"credentialSource": cred.source,
		"redactedFields":   []string{"username", "password"},
	}
	if params != nil {
		request["params"] = redactParamValues(params)
	}
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request":     request,
	}
}

func requiredName(parsed parsedArgs) (string, error) {
	name := firstNonEmpty(parsed.flags["name"], parsed.flags["code"], parsed.flags["table"])
	if name == "" && len(parsed.positionals) > 0 {
		name = parsed.positionals[0]
	}
	if name == "" {
		return "", cliError{2, "missing_name", "requires --name, --code, or --table"}
	}
	return name, nil
}

func parseJSON(body []byte) (map[string]any, error) {
	var data any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return asMap(data), nil
}

func redactParamValues(values url.Values) map[string]any {
	out := map[string]any{}
	for key, vals := range values {
		if isSecretKey(key) {
			out[key] = "REDACTED"
		} else if len(vals) == 1 {
			out[key] = vals[0]
		} else {
			out[key] = vals
		}
	}
	return out
}

func redactParamMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		if isSecretKey(key) {
			out[key] = "REDACTED"
		} else {
			out[key] = value
		}
	}
	return out
}

func isSecretKey(key string) bool {
	k := strings.ToLower(key)
	return k == "username" || k == "password" || k == "passwort" || strings.Contains(k, "token")
}

func reservedFlag(key string) bool {
	switch key {
	case "username", "password", "limit", "include-raw", "sample":
		return true
	default:
		return false
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

func isHelp(arg string) bool { return arg == "-h" || arg == "--help" || arg == "help" }

func match(args []string, parts ...string) bool {
	if len(args) < len(parts) {
		return false
	}
	for i, part := range parts {
		if args[i] != part {
			return false
		}
	}
	return true
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

func limitFlag(parsed parsedArgs, def int, max int) int {
	raw := parsed.flags["limit"]
	if raw == "" {
		raw = parsed.params.Get("pagelength")
	}
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

func objectKeys(m map[string]any) []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys
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

func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	if max < 4 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func redactUsername(username string) string {
	if username == "" || username == "GAST" {
		return username
	}
	return "REDACTED"
}

func redact(s string) string {
	re := regexp.MustCompile(`(?i)(username|password|passwort|token)=([^&\s]+)|(--(?:username|password|token)\s+)([^\s]+)`)
	return re.ReplaceAllStringFunc(s, func(part string) string {
		if strings.HasPrefix(part, "--") {
			prefix := strings.SplitN(part, " ", 2)[0]
			return prefix + " REDACTED"
		}
		if strings.Contains(part, "=") {
			return strings.SplitN(part, "=", 2)[0] + "=REDACTED"
		}
		return "REDACTED"
	})
}
