package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	appName              = "deutschlandatlas"
	portalSearchBase     = "https://www.karto365.de/portal/sharing/rest/search"
	hostingBase          = "https://www.karto365.de/hosting/rest/services"
	officialHomeURL      = "https://www.deutschlandatlas.bund.de/DE/Home/home_node.html"
	officialDownloadsURL = "https://www.deutschlandatlas.bund.de/DE/Service/Downloads/downloads_node.html"
	githubSpecURL        = "https://github.com/bundesAPI/deutschlandatlas-api"
	defaultTimeout       = 30 * time.Second
	defaultLimit         = 10
	safeLimit            = 100
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
	case match(args, "tables", "search"):
		err = runTablesSearch(args[2:])
	case match(args, "table", "query"):
		err = runTableQuery(args[2:])
	case match(args, "table", "fields"):
		err = runTableFields(args[2:])
	case match(args, "table", "sample"):
		err = runTableSample(args[2:])
	case match(args, "table", "source"):
		err = runTableSource(args[2:])
	case match(args, "indicator", "dossier"):
		err = runIndicatorDossier(args[2:])
	case args[0] == "query-builder":
		err = runQueryBuilder(args[1:])
	case args[0] == "explain-field":
		err = runExplainField(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run deutschlandatlas --help"}
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
	fmt.Println(`deutschlandatlas -- Deutschlandatlas ArcGIS research CLI

Purpose
  Discover and query public Deutschlandatlas indicator map services. The
  Deutschlandatlas describes regional living conditions in Germany through
  indicators on housing, work, mobility, health, education, infrastructure, and
  related policy fields.

Fast paths
  Check endpoint health and usage hints:
    deutschlandatlas doctor

  Find candidate indicator tables:
    deutschlandatlas tables search --term "Indikator" --limit 5

  Inspect one table before querying values:
    deutschlandatlas table fields --table alq_HA2023
    deutschlandatlas table sample --table alq_HA2023 --limit 5

  Build a compact evidence bundle:
    deutschlandatlas indicator dossier --table alq_HA2023

Raw endpoint command
  table query --table <table> [--param key=value] [--layer auto|0|5]
    Returns the upstream ArcGIS JSON. The CLI defaults to discovering the
    service's feature layer because many current services are not on layer 0.
    Use --layer 0 or --raw-layer-zero for exact layer-zero probing.

Research commands
  doctor
  tables search
  table fields
  table sample
  table source
  indicator dossier
  query-builder
  explain-field

Safety defaults
  - JSON output only.
  - Geometry is disabled unless --geometry true is passed.
  - Broad result counts default to 10 and are capped at 100 unless
    --allow-large-output is set.
  - No authentication is required for the public endpoints tested here.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "table sample":
		fmt.Println(`deutschlandatlas table sample

Fetch a small bounded sample from one Deutschlandatlas ArcGIS table.

Examples
  deutschlandatlas table sample --table alq_HA2023 --limit 5
  deutschlandatlas table sample --table alq_HA2023 --fields name,alq --where "alq > 10"
  deutschlandatlas table sample --table alq_HA2023 --geometry true --limit 2

Notes
  Geometry is off by default to avoid very large outputs. Use table fields first
  when you do not know the indicator field name.`)
	case "indicator dossier":
		fmt.Println(`deutschlandatlas indicator dossier

Bundle service metadata, selected feature layer, field descriptions, a tiny
sample, source URLs, warnings, and next actions for one indicator table.

Example
  deutschlandatlas indicator dossier --table alq_HA2023 --limit 3`)
	case "tables search":
		fmt.Println(`deutschlandatlas tables search

Search the public ArcGIS portal for Deutschlandatlas table services.

Examples
  deutschlandatlas tables search --term "Apotheken" --limit 5
  deutschlandatlas tables search --term "Indikator" --limit 5`)
	default:
		printRootHelp()
	}
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, 1, 10)
	searchURL := portalSearchURL("", limit, 1)
	searchData, searchErr := fetchJSON(searchURL)
	serviceData, serviceErr := fetchJSON(serviceURL("alq_HA2023"))

	payload := envelope("doctor", searchURL, nil)
	payload["summary"] = map[string]any{
		"portalSearchReachable":  searchErr == nil,
		"sampleServiceReachable": serviceErr == nil,
		"authRequired":           false,
		"publishedRateLimit":     "No exact public rate limit found in the reviewed Deutschlandatlas/API materials. Use small limits, cache stable indicator metadata, and avoid parallel broad ArcGIS queries.",
		"fairUseHints": []string{
			"Prefer tables search, fields, and small samples before broader queries.",
			"Do not request geometry unless map geometry is actually needed.",
			"Respect ArcGIS transfer limits and back off on 429, 5xx, or slow responses.",
		},
	}
	if searchErr == nil {
		payload["summary"].(map[string]any)["portalTotal"] = intAt(searchData, "total")
	}
	if serviceErr == nil {
		payload["summary"].(map[string]any)["sampleService"] = serviceSummary(serviceData)
	}
	if searchErr != nil || serviceErr != nil {
		payload["status"] = "degraded"
		payload["warnings"] = append(defaultWarnings(), compactErr("portalSearch", searchErr), compactErr("sampleService", serviceErr))
	} else {
		payload["warnings"] = defaultWarnings()
	}
	payload["sources"] = defaultSources()
	payload["nextActions"] = []string{
		`deutschlandatlas tables search --term "Indikator" --limit 5`,
		"deutschlandatlas indicator dossier --table alq_HA2023",
	}
	emit(payload)
	return nil
}

func runTablesSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "tables search requires --term"}
	}
	limit := limitFlag(parsed, 5, 25)
	start := intFlag(parsed, "start", 1)
	requestURL := portalSearchURL(term, limit, start)
	data, err := fetchJSON(requestURL)
	if err != nil {
		return err
	}

	items := compactPortalResults(data, limit)
	payload := envelope("tables search", requestURL, map[string]any{"term": term, "limit": limit, "start": start})
	payload["summary"] = map[string]any{
		"term":         term,
		"total":        intAt(data, "total"),
		"returned":     len(items),
		"limitApplied": limit,
		"nextStart":    intAt(data, "nextStart"),
	}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForTables(items)
	emit(payload)
	return nil
}

func runTableQuery(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	layer, _, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("f", firstNonEmpty(parsed.params.Get("f"), "json"))
	params.Set("where", firstNonEmpty(parsed.params.Get("where"), parsed.flags["where"], "1=1"))
	params.Set("outFields", firstNonEmpty(parsed.params.Get("outFields"), parsed.params.Get("outfields"), parsed.flags["fields"], "*"))
	params.Set("returnGeometry", firstNonEmpty(parsed.params.Get("returnGeometry"), parsed.params.Get("returngeometry"), boolString(flagBool(parsed, "geometry"))))
	for key, values := range parsed.params {
		for _, value := range values {
			params.Set(key, value)
		}
	}
	if params.Get("resultRecordCount") == "" && params.Get("resultrecordcount") == "" {
		params.Set("resultRecordCount", strconv.Itoa(limitFlag(parsed, defaultLimit, safeLimit)))
	}
	if !flagBool(parsed, "allow-large-output") {
		count := firstNonEmpty(params.Get("resultRecordCount"), params.Get("resultrecordcount"))
		if count != "" {
			n, _ := strconv.Atoi(count)
			if n > safeLimit {
				return cliError{2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override"}
			}
		}
	}
	requestURL := queryURL(table, layer, params)
	data, err := fetchJSON(requestURL)
	if err != nil {
		return err
	}
	emit(data)
	return nil
}

func runTableFields(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	layer, layerSource, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	layerURL := layerURL(table, layer)
	layerData, err := fetchJSON(layerURL)
	if err != nil {
		return err
	}
	fields := compactFields(layerData)

	payload := envelope("table fields", layerURL, map[string]any{"table": table, "layer": layer})
	payload["summary"] = map[string]any{
		"table":                 table,
		"layer":                 layer,
		"layerSource":           layerSource,
		"fieldCount":            len(fields),
		"displayField":          stringAt(layerData, "displayField"),
		"objectIdField":         stringAt(layerData, "objectIdField"),
		"geometryType":          stringAt(layerData, "geometryType"),
		"maxRecordCount":        intAt(layerData, "maxRecordCount"),
		"likelyIndicatorFields": likelyIndicatorFields(fields),
	}
	payload["items"] = fields
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table sample --table %s --fields name,%s --limit 5", table, firstLikelyIndicator(fields)),
		fmt.Sprintf("deutschlandatlas indicator dossier --table %s", table),
	}
	emit(payload)
	return nil
}

func runTableSample(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	layer, layerSource, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("f", "json")
	params.Set("where", firstNonEmpty(parsed.params.Get("where"), parsed.flags["where"], "1=1"))
	params.Set("outFields", firstNonEmpty(parsed.params.Get("outFields"), parsed.params.Get("outfields"), parsed.flags["fields"], "*"))
	params.Set("returnGeometry", boolString(flagBool(parsed, "geometry")))
	params.Set("resultRecordCount", strconv.Itoa(limit))
	for key, values := range parsed.params {
		for _, value := range values {
			params.Set(key, value)
		}
	}
	requestURL := queryURL(table, layer, params)
	data, err := fetchJSON(requestURL)
	if err != nil {
		return err
	}
	items := compactFeatures(data, flagBool(parsed, "geometry"))
	warnings := defaultWarnings()
	if boolAt(data, "exceededTransferLimit") {
		warnings = append(warnings, "ArcGIS reported exceededTransferLimit=true; narrow the where clause or paginate deliberately.")
	}
	if flagBool(parsed, "geometry") {
		warnings = append(warnings, "Geometry was requested intentionally; outputs can grow quickly.")
	}

	payload := envelope("table sample", requestURL, map[string]any{"table": table, "layer": layer, "limit": limit})
	payload["summary"] = map[string]any{
		"table":                 table,
		"layer":                 layer,
		"layerSource":           layerSource,
		"returned":              len(items),
		"limitApplied":          limit,
		"returnGeometry":        flagBool(parsed, "geometry"),
		"exceededTransferLimit": boolAt(data, "exceededTransferLimit"),
		"displayField":          stringAt(data, "displayFieldName"),
	}
	payload["items"] = items
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table fields --table %s", table),
		fmt.Sprintf("deutschlandatlas query-builder --table %s --where \"name LIKE '%%Berlin%%'\" --fields name,* --limit 10", table),
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runTableSource(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	layer := 0
	layerSource := "raw_default"
	if !flagBool(parsed, "skip-layer-discovery") {
		if discovered, source, err := resolveLayer(table, parsed); err == nil {
			layer = discovered
			layerSource = source
		}
	}
	payload := envelope("table source", serviceURL(table), map[string]any{"table": table, "layer": layer})
	payload["summary"] = map[string]any{
		"table":          table,
		"selectedLayer":  layer,
		"layerSource":    layerSource,
		"authRequired":   false,
		"apiStyle":       "ArcGIS REST MapServer query endpoint",
		"rateLimitFound": false,
	}
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table fields --table %s", table),
		fmt.Sprintf("deutschlandatlas table sample --table %s --limit 5", table),
	}
	emit(payload)
	return nil
}

func runIndicatorDossier(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, 5, 25)
	layer, layerSource, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	serviceData, serviceErr := fetchJSON(serviceURL(table))
	layerData, layerErr := fetchJSON(layerURL(table, layer))

	params := url.Values{}
	params.Set("f", "json")
	params.Set("where", "1=1")
	params.Set("outFields", "*")
	params.Set("returnGeometry", "false")
	params.Set("resultRecordCount", strconv.Itoa(limit))
	sampleData, sampleErr := fetchJSON(queryURL(table, layer, params))

	payload := envelope("indicator dossier", serviceURL(table), map[string]any{"table": table, "layer": layer, "limit": limit})
	payload["summary"] = map[string]any{
		"table":         table,
		"selectedLayer": layer,
		"layerSource":   layerSource,
		"limitApplied":  limit,
		"authRequired":  false,
	}
	warnings := defaultWarnings()
	if serviceErr == nil {
		payload["service"] = serviceSummary(serviceData)
	} else {
		warnings = append(warnings, compactErr("serviceMetadata", serviceErr))
	}
	if layerErr == nil {
		fields := compactFields(layerData)
		payload["fields"] = fields
		payload["summary"].(map[string]any)["likelyIndicatorFields"] = likelyIndicatorFields(fields)
	} else {
		warnings = append(warnings, compactErr("layerMetadata", layerErr))
	}
	if sampleErr == nil {
		payload["sample"] = map[string]any{
			"items":                 compactFeatures(sampleData, false),
			"exceededTransferLimit": boolAt(sampleData, "exceededTransferLimit"),
		}
		if boolAt(sampleData, "exceededTransferLimit") {
			warnings = append(warnings, "Sample query reports exceededTransferLimit=true; this is expected for broad ArcGIS layers and means pagination/filtering is needed for full extraction.")
		}
	} else {
		warnings = append(warnings, compactErr("sampleQuery", sampleErr))
	}
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table fields --table %s", table),
		fmt.Sprintf("deutschlandatlas table sample --table %s --fields name,* --where \"1=1\" --limit 10", table),
		fmt.Sprintf("deutschlandatlas explain-field --table %s --field %s", table, firstLikelyIndicator(asSlice(payload["fields"]))),
	}
	emit(payload)
	return nil
}

func runQueryBuilder(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	layer, layerSource, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	where := firstNonEmpty(parsed.flags["where"], "1=1")
	if parsed.flags["region"] != "" && where == "1=1" {
		where = fmt.Sprintf("name LIKE '%%%s%%'", strings.ReplaceAll(parsed.flags["region"], "'", "''"))
	}
	params := url.Values{}
	params.Set("f", "json")
	params.Set("where", where)
	params.Set("outFields", firstNonEmpty(parsed.flags["fields"], "*"))
	params.Set("returnGeometry", boolString(flagBool(parsed, "geometry")))
	params.Set("resultRecordCount", strconv.Itoa(limit))
	for key, values := range parsed.params {
		for _, value := range values {
			params.Set(key, value)
		}
	}
	builtURL := queryURL(table, layer, params)
	payload := envelope("query-builder", builtURL, map[string]any{"table": table, "layer": layer, "params": params})
	payload["summary"] = map[string]any{
		"table":          table,
		"layer":          layer,
		"layerSource":    layerSource,
		"requestUrl":     builtURL,
		"doesNotFetch":   true,
		"limitApplied":   limit,
		"returnGeometry": flagBool(parsed, "geometry"),
	}
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = defaultWarnings()
	if parsed.flags["year"] != "" {
		payload["warnings"] = append(payload["warnings"].([]string), "Generic Deutschlandatlas table services do not expose one standard year parameter; use table search to choose the year-specific table.")
	}
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table query --table %s --layer %d --param where=%q --param outFields=%q --limit %d", table, layer, where, params.Get("outFields"), limit),
	}
	emit(payload)
	return nil
}

func runExplainField(argv []string) error {
	parsed := parseArgs(argv)
	table, err := requiredTable(parsed)
	if err != nil {
		return err
	}
	fieldName := firstNonEmpty(parsed.flags["field"], parsed.flags["name"], firstPosition(parsed))
	if fieldName == "" {
		return cliError{2, "missing_field", "explain-field requires --field"}
	}
	layer, _, err := resolveLayer(table, parsed)
	if err != nil {
		return err
	}
	layerData, err := fetchJSON(layerURL(table, layer))
	if err != nil {
		return err
	}
	fields := compactFields(layerData)
	var matchField any
	for _, item := range fields {
		m := asMap(item)
		if strings.EqualFold(stringAt(m, "name"), fieldName) {
			matchField = item
			break
		}
	}
	if matchField == nil {
		return cliError{2, "field_not_found", "field not found in layer metadata"}
	}
	payload := envelope("explain-field", layerURL(table, layer), map[string]any{"table": table, "field": fieldName, "layer": layer})
	payload["summary"] = map[string]any{
		"table":              table,
		"layer":              layer,
		"field":              matchField,
		"interpretationHint": "Use the alias, table title/snippet from tables search, and official downloads/method notes for statistical meaning and units.",
	}
	payload["sources"] = sourcesForTable(table, layer)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("deutschlandatlas table sample --table %s --fields name,%s --limit 10", table, fieldName),
	}
	emit(payload)
	return nil
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

func requiredTable(parsed parsedArgs) (string, error) {
	table := firstNonEmpty(parsed.flags["table"], parsed.flags["name"], firstPosition(parsed))
	if table == "" {
		return "", cliError{2, "missing_table", "command requires --table"}
	}
	return table, nil
}

func resolveLayer(table string, parsed parsedArgs) (int, string, error) {
	if flagBool(parsed, "raw-layer-zero") {
		return 0, "raw_layer_zero", nil
	}
	layerFlag := firstNonEmpty(parsed.flags["layer"], "auto")
	if layerFlag != "" && layerFlag != "auto" {
		layer, err := strconv.Atoi(layerFlag)
		if err != nil {
			return 0, "", cliError{2, "invalid_layer", "--layer must be auto or an integer"}
		}
		return layer, "explicit_flag", nil
	}
	data, err := fetchJSON(serviceURL(table))
	if err != nil {
		return 0, "", err
	}
	layer, ok := firstFeatureLayerID(data)
	if !ok {
		return 0, "", cliError{1, "no_feature_layer", "service metadata did not expose a feature layer"}
	}
	return layer, "service_metadata", nil
}

func limitFlag(parsed parsedArgs, fallback int, max int) int {
	raw := firstNonEmpty(parsed.flags["limit"], parsed.flags["resultrecordcount"], parsed.params.Get("resultRecordCount"), parsed.params.Get("resultrecordcount"))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	if n > max && !flagBool(parsed, "allow-large-output") {
		fail(2, "limit_exceeds_safe_max", fmt.Sprintf("limit %d exceeds safe max %d; pass --allow-large-output to override", n, max))
	}
	return n
}

func intFlag(parsed parsedArgs, key string, fallback int) int {
	raw := parsed.flags[key]
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func fetchJSON(requestURL string) (map[string]any, error) {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "germany-skills/deutschlandatlas")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, requestURL, truncate(string(body), 300))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode JSON from %s: %w", requestURL, err)
	}
	if upstreamErr, ok := data["error"].(map[string]any); ok {
		return nil, fmt.Errorf("upstream error %v: %v", upstreamErr["code"], upstreamErr["message"])
	}
	return data, nil
}

func portalSearchURL(term string, limit int, start int) string {
	query := "deutschlandatlas"
	if strings.TrimSpace(term) != "" {
		query += " " + strings.TrimSpace(term)
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("f", "json")
	params.Set("num", strconv.Itoa(limit))
	params.Set("start", strconv.Itoa(start))
	return portalSearchBase + "?" + params.Encode()
}

func serviceURL(table string) string {
	return hostingBase + "/" + url.PathEscape(table) + "/MapServer?f=json"
}

func layerURL(table string, layer int) string {
	return hostingBase + "/" + url.PathEscape(table) + "/MapServer/" + strconv.Itoa(layer) + "?f=json"
}

func queryURL(table string, layer int, params url.Values) string {
	return hostingBase + "/" + url.PathEscape(table) + "/MapServer/" + strconv.Itoa(layer) + "/query?" + params.Encode()
}

func firstFeatureLayerID(data map[string]any) (int, bool) {
	layers := asSlice(data["layers"])
	fallback := -1
	for _, layerAny := range layers {
		layer := asMap(layerAny)
		id := intAt(layer, "id")
		if fallback < 0 {
			fallback = id
		}
		if strings.Contains(strings.ToLower(stringAt(layer, "type")), "feature") {
			return id, true
		}
	}
	return fallback, fallback >= 0
}

func compactPortalResults(data map[string]any, limit int) []any {
	results := asSlice(data["results"])
	items := []any{}
	for i, itemAny := range results {
		if i >= limit {
			break
		}
		item := asMap(itemAny)
		service := stringAt(item, "url")
		table := firstNonEmpty(stringAt(item, "title"), tableFromURL(service))
		items = append(items, map[string]any{
			"table":       table,
			"title":       stringAt(item, "title"),
			"snippet":     stringAt(item, "snippet"),
			"type":        stringAt(item, "type"),
			"serviceUrl":  service,
			"access":      stringAt(item, "access"),
			"tags":        item["tags"],
			"modifiedUtc": millisToUTC(item["modified"]),
			"nextActions": []string{
				fmt.Sprintf("deutschlandatlas table fields --table %s", table),
				fmt.Sprintf("deutschlandatlas indicator dossier --table %s", table),
			},
		})
	}
	return items
}

func compactFields(layerData map[string]any) []any {
	fields := []any{}
	for _, fieldAny := range asSlice(layerData["fields"]) {
		field := asMap(fieldAny)
		fields = append(fields, map[string]any{
			"name":   stringAt(field, "name"),
			"alias":  stringAt(field, "alias"),
			"type":   stringAt(field, "type"),
			"length": intAt(field, "length"),
			"domain": field["domain"],
		})
	}
	return fields
}

func compactFeatures(data map[string]any, includeGeometry bool) []any {
	items := []any{}
	for _, featureAny := range asSlice(data["features"]) {
		feature := asMap(featureAny)
		item := map[string]any{"attributes": feature["attributes"]}
		if includeGeometry {
			item["geometry"] = feature["geometry"]
		}
		items = append(items, item)
	}
	return items
}

func likelyIndicatorFields(fields []any) []string {
	names := []string{}
	for _, fieldAny := range fields {
		field := asMap(fieldAny)
		name := stringAt(field, "name")
		lower := strings.ToLower(name)
		if name == "" || strings.HasPrefix(lower, "shape") {
			continue
		}
		switch lower {
		case "objectid", "gf", "gen", "bez", "gebietskennziffer", "name":
			continue
		}
		names = append(names, name)
	}
	return names
}

func firstLikelyIndicator(fields []any) string {
	likely := likelyIndicatorFields(fields)
	if len(likely) > 0 {
		return likely[0]
	}
	return "*"
}

func serviceSummary(data map[string]any) map[string]any {
	layerSummaries := []any{}
	for _, layerAny := range asSlice(data["layers"]) {
		layer := asMap(layerAny)
		layerSummaries = append(layerSummaries, map[string]any{
			"id":   intAt(layer, "id"),
			"name": stringAt(layer, "name"),
			"type": stringAt(layer, "type"),
		})
	}
	return map[string]any{
		"serviceDescription":    stringAt(data, "serviceDescription"),
		"mapName":               stringAt(data, "mapName"),
		"supportedQueryFormats": stringAt(data, "supportedQueryFormats"),
		"maxRecordCount":        intAt(data, "maxRecordCount"),
		"layers":                layerSummaries,
	}
}

func sourcesForTable(table string, layer int) []any {
	return []any{
		map[string]any{"title": "Deutschlandatlas start page", "url": officialHomeURL, "kind": "official_context"},
		map[string]any{"title": "Deutschlandatlas data downloads and method notes", "url": officialDownloadsURL, "kind": "official_downloads"},
		map[string]any{"title": "bundesAPI Deutschlandatlas OpenAPI wrapper", "url": githubSpecURL, "kind": "openapi_reference"},
		map[string]any{"title": "ArcGIS service metadata", "url": serviceURL(table), "kind": "api_service"},
		map[string]any{"title": "ArcGIS layer metadata", "url": layerURL(table, layer), "kind": "api_layer"},
		map[string]any{"title": "ArcGIS portal search", "url": portalSearchURL(table, 10, 1), "kind": "api_discovery"},
	}
}

func defaultSources() []any {
	return []any{
		map[string]any{"title": "Deutschlandatlas start page", "url": officialHomeURL, "kind": "official_context"},
		map[string]any{"title": "Deutschlandatlas data downloads and method notes", "url": officialDownloadsURL, "kind": "official_downloads"},
		map[string]any{"title": "bundesAPI Deutschlandatlas OpenAPI wrapper", "url": githubSpecURL, "kind": "openapi_reference"},
		map[string]any{"title": "ArcGIS portal Deutschlandatlas search", "url": portalSearchURL("", 100, 1), "kind": "api_discovery"},
	}
}

func defaultWarnings() []string {
	return []string{
		"No exact published API rate limit was found in the reviewed materials; keep requests small and cache stable metadata.",
		"Official download notes state that missing values in tabular downloads are represented as -9999; check field notes before statistical interpretation.",
		"ArcGIS services can enforce maxRecordCount/transfer limits; use filters, fields, and pagination rather than broad full-table pulls.",
	}
}

func nextActionsForTables(items []any) []string {
	actions := []string{}
	if len(items) == 0 {
		return []string{`deutschlandatlas tables search --term "Apotheken" --limit 5`}
	}
	for i, itemAny := range items {
		if i >= 3 {
			break
		}
		table := stringAt(asMap(itemAny), "table")
		if table != "" {
			actions = append(actions, fmt.Sprintf("deutschlandatlas indicator dossier --table %s", table))
		}
	}
	return actions
}

func envelope(command string, requestURL string, request any) map[string]any {
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request": map[string]any{
			"method": "GET",
			"url":    requestURL,
			"params": request,
		},
		"summary":     map[string]any{},
		"items":       []any{},
		"sources":     []any{},
		"warnings":    []string{},
		"nextActions": []string{},
	}
}

func emit(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func fail(exitCode int, code string, message string) {
	emit(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
	os.Exit(exitCode)
}

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

func isHelp(value string) bool {
	return value == "--help" || value == "-h" || value == "help"
}

func flagBool(parsed parsedArgs, key string) bool {
	value := strings.ToLower(strings.TrimSpace(parsed.flags[key]))
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPosition(parsed parsedArgs) string {
	if len(parsed.positionals) == 0 {
		return ""
	}
	return parsed.positionals[0]
}

func asMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asSlice(value any) []any {
	if s, ok := value.([]any); ok {
		return s
	}
	return []any{}
}

func stringAt(data map[string]any, key string) string {
	if value, ok := data[key]; ok && value != nil {
		return fmt.Sprint(value)
	}
	return ""
}

func intAt(data map[string]any, key string) int {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		n, _ := strconv.Atoi(typed)
		return n
	default:
		return 0
	}
}

func boolAt(data map[string]any, key string) bool {
	value, ok := data[key]
	if !ok || value == nil {
		return false
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return strings.EqualFold(fmt.Sprint(value), "true")
}

func tableFromURL(raw string) string {
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	for i, part := range parts {
		if part == "services" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func millisToUTC(value any) string {
	switch typed := value.(type) {
	case float64:
		return time.UnixMilli(int64(typed)).UTC().Format(time.RFC3339)
	case int64:
		return time.UnixMilli(typed).UTC().Format(time.RFC3339)
	default:
		return ""
	}
}

func compactErr(label string, err error) string {
	if err == nil {
		return ""
	}
	return label + ": " + err.Error()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
