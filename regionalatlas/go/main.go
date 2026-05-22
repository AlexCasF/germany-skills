package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
	appName            = "regionalatlas"
	mapServerURL       = "https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer"
	queryEndpoint      = mapServerURL + "/dynamicLayer/query"
	catalogURL         = "https://regionalatlas.statistikportal.de/taskrunner/services.json"
	thesaurusURL       = "https://regionalatlas.statistikportal.de/app/csv/thesaurus.csv"
	appURL             = "https://regionalatlas.statistikportal.de/"
	statistikportalURL = "https://www.statistikportal.de/de/karten/regionalatlas-deutschland"
	destatisURL        = "https://www.destatis.de/DE/Service/Statistik-Visualisiert/RegionalatlasAktuell.html"
	openDataURL        = "https://www.statistikportal.de/de/open-data"
	mapsGeodataURL     = "https://www.destatis.de/DE/Service/OpenData/karten-geodaten.html"
	openAPIRepoURL     = "https://github.com/bundesAPI/regionalatlas-api"
	defaultTimeout     = 45 * time.Second
	defaultLimit       = 10
	safeLimit          = 100
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

type catalogNode struct {
	Title      string                 `json:"title"`
	Code       string                 `json:"code"`
	Timestamp  string                 `json:"timestamp"`
	TitleShort string                 `json:"title_short"`
	TitleLong  string                 `json:"title_long"`
	Years      map[string][]yearInfo  `json:"years"`
	Attributes []indicatorAttribute   `json:"attributes"`
	Children   []catalogNode          `json:"children"`
	Extra      map[string]interface{} `json:"-"`
}

type yearInfo struct {
	Precision        any   `json:"precision"`
	GeomLevel        int   `json:"geom_level"`
	GeomLevels       []int `json:"geom_levels"`
	YearUnitsCounter []int `json:"year_units_counter"`
}

type indicatorAttribute struct {
	Code       string `json:"code"`
	TitleShort string `json:"title_short"`
	TitleLong  string `json:"title_long"`
	Unit       string `json:"unit"`
	Meta       string `json:"meta"`
}

type flatIndicator struct {
	Topic string
	Node  catalogNode
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
	case match(args, "indicators", "list"):
		err = runIndicatorsList(args[2:])
	case match(args, "indicators", "search"):
		err = runIndicatorsSearch(args[2:])
	case match(args, "indicator", "get"):
		err = runIndicatorGet(args[2:])
	case args[0] == "fields":
		err = runFields(args[1:])
	case args[0] == "sample":
		err = runSample(args[1:])
	case args[0] == "source":
		err = runSource(args[1:])
	case args[0] == "dossier":
		err = runDossier(args[1:])
	case args[0] == "query-builder":
		err = runQueryBuilder(args[1:])
	case args[0] == "explain-field":
		err = runExplainField(args[1:])
	case args[0] == "query":
		err = runRawQuery(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run regionalatlas --help"}
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
	fmt.Println(`regionalatlas -- Regionalatlas Deutschland research CLI

Purpose
  Discover and query official Regionalatlas indicators from the statistical
  offices of the German federation and states. The atlas covers regional
  indicators for Laender, Regierungsbezirke/statistical regions, districts,
  cities, and municipalities.

Use this when
  - you need official regional statistical indicators by administrative level
  - you need field meanings, units, years, and source metadata before querying
  - you need small, cited samples from the ArcGIS dynamic-layer endpoint

Do not use this when
  - you need broad national GENESIS tables without map/regional context; use Destatis
  - you need Deutschlandatlas indicator tables; use deutschlandatlas

Fast paths
  Check health and fair-use hints:
    regionalatlas doctor

  Discover indicators:
    regionalatlas indicators search --term "Arbeitslosenquote" --limit 5

  Inspect metadata before data:
    regionalatlas fields --indicator AI008-1-5
    regionalatlas explain-field --indicator AI008-1-5 --field AI0801

  Fetch a tiny safe sample:
    regionalatlas sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 5

Commands
  doctor
  indicators list
  indicators search
  indicator get
  fields
  sample
  source
  dossier
  query-builder
  explain-field
  query              Raw legacy dynamic-layer query

Output guarantees
  Research commands emit JSON envelopes with status, request, retrievedAt,
  summary/items, sources, warnings, and nextActions. Raw query returns upstream
  ArcGIS JSON but still adds a safe default resultRecordCount when omitted.

Safety defaults
  - resultRecordCount defaults small and is capped at 100 without --allow-large-output
  - returnGeometry=false unless --geometry true is passed
  - region-level defaults to 1 (Laender), not municipalities
  - no authentication is required for the public endpoints tested here`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "sample":
		fmt.Println(`regionalatlas sample

Fetch a small bounded sample from a Regionalatlas dynamic-layer query.

Examples
  regionalatlas sample --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1 --limit 5
  regionalatlas sample --indicator AI002-1-5 --field AI0201 --year 2020 --region-level 1 --limit 5
  regionalatlas sample --indicator AI008-1-5 --field AI0801 --year 2024 --ags 11

Region levels
  1 = Laender
  2 = Regierungsbezirke/statistical regions
  3 = Kreise and kreisfreie Staedte
  5 = Gemeinden/Gemeindeverbaende

Safety
  Geometry is off by default. Limits above 100 require --allow-large-output.`)
	case "dossier":
		fmt.Println(`regionalatlas dossier

Build a compact evidence bundle: catalog metadata, fields, source URLs, query
URL, a tiny sample, warnings, and next actions.

Example
  regionalatlas dossier --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1`)
	case "query-builder":
		fmt.Println(`regionalatlas query-builder

Build the encoded ArcGIS dynamic-layer query URL without fetching data.

Example
  regionalatlas query-builder --indicator AI008-1-5 --field AI0801 --year 2024 --region-level 1`)
	case "query":
		fmt.Println(`regionalatlas query

Run a raw ArcGIS dynamic-layer query. This is the low-level compatibility
escape hatch; prefer sample/query-builder when possible.

Examples
  regionalatlas query --layer-file layer.json --param outFields=ags,gen,ai0801
  regionalatlas query --layer <json> --param resultRecordCount=5

Safety
  resultRecordCount defaults small and is capped at 100 without --allow-large-output.
  On Windows shells, prefer --layer-file because raw JSON quoting is fragile.`)
	default:
		printRootHelp()
	}
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, 1, 10)
	serviceData, serviceErr := fetchJSON(mapServerURL + "?f=json")
	catalog, catalogErr := fetchCatalog()
	payload := envelope("doctor", mapServerURL+"?f=json", nil)
	summary := map[string]any{
		"authRequired":       false,
		"catalogUrl":         catalogURL,
		"mapServerUrl":       mapServerURL,
		"publishedRateLimit": "No exact public API rate limit found in reviewed Regionalatlas/API materials. Use small limits, cache catalog metadata, and avoid parallel broad ArcGIS queries.",
		"fairUseHints": []string{
			"Use indicators search/list and fields before sample/query.",
			"Do not request geometry unless map shapes are required.",
			"Avoid municipality-level full pulls unless explicitly exporting with a plan.",
			"Back off on 429, 5xx, or slow responses.",
		},
	}
	warnings := defaultWarnings()
	if serviceErr == nil {
		summary["mapServerReachable"] = true
		summary["mapServer"] = mapServerSummary(serviceData)
	} else {
		summary["mapServerReachable"] = false
		warnings = append(warnings, "mapServer: "+serviceErr.Error())
	}
	if catalogErr == nil {
		flat := flattenCatalog(catalog)
		summary["catalogReachable"] = true
		summary["topics"] = len(catalog)
		summary["indicators"] = len(flat)
		summary["sampleIndicators"] = compactIndicators(flat, limit)
	} else {
		summary["catalogReachable"] = false
		warnings = append(warnings, "catalog: "+catalogErr.Error())
	}
	if serviceErr != nil || catalogErr != nil {
		payload["status"] = "degraded"
	}
	payload["summary"] = summary
	payload["sources"] = defaultSources()
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		`regionalatlas indicators search --term "Arbeitslosenquote" --limit 5`,
		"regionalatlas fields --indicator AI008-1-5",
	}
	emit(payload)
	return nil
}

func runIndicatorsList(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, defaultLimit, 50)
	catalog, err := fetchCatalog()
	if err != nil {
		return err
	}
	flat := flattenCatalog(catalog)
	topic := strings.ToLower(firstNonEmpty(parsed.flags["topic"], parsed.flags["thema"]))
	if topic != "" {
		filtered := []flatIndicator{}
		for _, item := range flat {
			if strings.Contains(strings.ToLower(item.Topic), topic) {
				filtered = append(filtered, item)
			}
		}
		flat = filtered
	}
	payload := envelope("indicators list", catalogURL, map[string]any{"limit": limit, "topic": topic})
	payload["summary"] = map[string]any{"returned": min(limit, len(flat)), "available": len(flat), "topicFilter": topic}
	payload["items"] = compactIndicators(flat, limit)
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		`regionalatlas indicators search --term "Arbeitslosenquote" --limit 5`,
	}
	emit(payload)
	return nil
}

func runIndicatorsSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "indicators search requires --term"}
	}
	limit := limitFlag(parsed, 5, 50)
	catalog, err := fetchCatalog()
	if err != nil {
		return err
	}
	matches := searchCatalog(flattenCatalog(catalog), term)
	payload := envelope("indicators search", catalogURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"term": term, "matches": len(matches), "returned": min(limit, len(matches))}
	payload["items"] = compactIndicators(matches, limit)
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForIndicators(matches)
	emit(payload)
	return nil
}

func runIndicatorGet(argv []string) error {
	parsed := parseArgs(argv)
	code := requiredIndicator(parsed)
	item, err := findIndicator(code)
	if err != nil {
		return err
	}
	payload := envelope("indicator get", catalogURL, map[string]any{"indicator": code})
	payload["summary"] = indicatorSummary(item)
	payload["items"] = compactAttributes(item.Node.Attributes, 50, "")
	payload["sources"] = sourcesForIndicator(item.Node, firstAttributeCode(item.Node), latestYear(item.Node))
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas fields --indicator %s", item.Node.Code),
		fmt.Sprintf("regionalatlas sample --indicator %s --field %s --year %d --region-level 1 --limit 5", item.Node.Code, firstAttributeCode(item.Node), latestYear(item.Node)),
	}
	emit(payload)
	return nil
}

func runFields(argv []string) error {
	parsed := parseArgs(argv)
	code := requiredIndicator(parsed)
	item, err := findIndicator(code)
	if err != nil {
		return err
	}
	payload := envelope("fields", catalogURL, map[string]any{"indicator": code})
	payload["summary"] = map[string]any{
		"indicator":       item.Node.Code,
		"title":           item.Node.TitleShort,
		"topic":           item.Topic,
		"availableYears":  availableYears(item.Node),
		"latestYear":      latestYear(item.Node),
		"regionLevels":    regionLevelAvailability(item.Node),
		"attributeCount":  len(item.Node.Attributes),
		"regionalDbTable": item.Node.Code,
	}
	payload["items"] = compactAttributes(item.Node.Attributes, 100, "")
	payload["sources"] = sourcesForIndicator(item.Node, firstAttributeCode(item.Node), latestYear(item.Node))
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas explain-field --indicator %s --field %s", item.Node.Code, firstAttributeCode(item.Node)),
		fmt.Sprintf("regionalatlas sample --indicator %s --field %s --year %d --region-level 1 --limit 5", item.Node.Code, firstAttributeCode(item.Node), latestYear(item.Node)),
	}
	emit(payload)
	return nil
}

func runSample(argv []string) error {
	parsed := parseArgs(argv)
	item, field, year, regionLevel, err := resolveQueryInputs(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	params := buildQueryParams(item.Node, field, year, regionLevel, limit, parsed)
	requestURL := queryEndpoint + "?" + params.Encode()
	data, err := fetchJSON(requestURL)
	if err != nil {
		return err
	}
	items := compactFeatures(data, flagBool(parsed, "geometry"))
	warnings := defaultWarnings()
	if boolAt(data, "exceededTransferLimit") {
		warnings = append(warnings, "ArcGIS reported exceededTransferLimit=true; the returned sample is not a complete extract.")
	}
	if flagBool(parsed, "geometry") {
		warnings = append(warnings, "Geometry was requested intentionally; municipality-level geometry can be very large.")
	}
	payload := envelope("sample", requestURL, map[string]any{"indicator": item.Node.Code, "field": strings.ToUpper(field), "year": year, "regionLevel": regionLevel, "limit": limit})
	payload["summary"] = map[string]any{
		"indicator":             item.Node.Code,
		"field":                 strings.ToUpper(field),
		"fieldTitle":            attributeTitle(item.Node, field),
		"unit":                  attributeUnit(item.Node, field),
		"year":                  year,
		"regionLevel":           regionLevel,
		"regionLevelLabel":      regionLevelLabel(regionLevel),
		"returned":              len(items),
		"limitApplied":          limit,
		"returnGeometry":        flagBool(parsed, "geometry"),
		"exceededTransferLimit": boolAt(data, "exceededTransferLimit"),
	}
	payload["items"] = items
	payload["sources"] = sourcesForIndicator(item.Node, field, year)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas query-builder --indicator %s --field %s --year %d --region-level %d --limit %d", item.Node.Code, strings.ToUpper(field), year, regionLevel, limit),
		fmt.Sprintf("regionalatlas explain-field --indicator %s --field %s", item.Node.Code, strings.ToUpper(field)),
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = data
	}
	emit(payload)
	return nil
}

func runSource(argv []string) error {
	parsed := parseArgs(argv)
	code := requiredIndicator(parsed)
	item, err := findIndicator(code)
	if err != nil {
		return err
	}
	field := firstNonEmpty(parsed.flags["field"], firstAttributeCode(item.Node))
	year := intFlag(parsed, "year", latestYear(item.Node))
	payload := envelope("source", catalogURL, map[string]any{"indicator": code, "field": field, "year": year})
	payload["summary"] = map[string]any{
		"indicator":       item.Node.Code,
		"title":           item.Node.TitleShort,
		"field":           strings.ToUpper(field),
		"year":            year,
		"regionalDbTable": item.Node.Code,
		"authRequired":    false,
	}
	payload["sources"] = sourcesForIndicator(item.Node, field, year)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas dossier --indicator %s --field %s --year %d", item.Node.Code, strings.ToUpper(field), year),
	}
	emit(payload)
	return nil
}

func runDossier(argv []string) error {
	parsed := parseArgs(argv)
	item, field, year, regionLevel, err := resolveQueryInputs(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, 5, 25)
	params := buildQueryParams(item.Node, field, year, regionLevel, limit, parsed)
	requestURL := queryEndpoint + "?" + params.Encode()
	sample, sampleErr := fetchJSON(requestURL)
	warnings := defaultWarnings()
	if sampleErr != nil {
		warnings = append(warnings, "sampleQuery: "+sampleErr.Error())
	}
	payload := envelope("dossier", requestURL, map[string]any{"indicator": item.Node.Code, "field": strings.ToUpper(field), "year": year, "regionLevel": regionLevel, "limit": limit})
	payload["summary"] = map[string]any{
		"indicator":        item.Node.Code,
		"title":            item.Node.TitleShort,
		"topic":            item.Topic,
		"field":            strings.ToUpper(field),
		"fieldTitle":       attributeTitle(item.Node, field),
		"unit":             attributeUnit(item.Node, field),
		"year":             year,
		"regionLevel":      regionLevel,
		"regionLevelLabel": regionLevelLabel(regionLevel),
		"availableYears":   availableYears(item.Node),
		"regionLevels":     regionLevelAvailability(item.Node),
	}
	payload["fields"] = compactAttributes(item.Node.Attributes, 100, "")
	payload["metadata"] = map[string]any{
		"indicatorTitleLong": item.Node.TitleLong,
		"fieldMetaSnippets":  metaSnippets(attributeMeta(item.Node, field), "", 6),
	}
	if sampleErr == nil {
		payload["sample"] = map[string]any{
			"items":                 compactFeatures(sample, flagBool(parsed, "geometry")),
			"exceededTransferLimit": boolAt(sample, "exceededTransferLimit"),
		}
		if boolAt(sample, "exceededTransferLimit") {
			warnings = append(warnings, "Sample query reports exceededTransferLimit=true; use pagination/filtering for complete extraction.")
		}
	}
	payload["sources"] = sourcesForIndicator(item.Node, field, year)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas explain-field --indicator %s --field %s --grep Quelle", item.Node.Code, strings.ToUpper(field)),
		fmt.Sprintf("regionalatlas query-builder --indicator %s --field %s --year %d --region-level %d", item.Node.Code, strings.ToUpper(field), year, regionLevel),
	}
	emit(payload)
	return nil
}

func runQueryBuilder(argv []string) error {
	parsed := parseArgs(argv)
	item, field, year, regionLevel, err := resolveQueryInputs(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	params := buildQueryParams(item.Node, field, year, regionLevel, limit, parsed)
	requestURL := queryEndpoint + "?" + params.Encode()
	layerDecoded, _ := url.QueryUnescape(params.Get("layer"))
	payload := envelope("query-builder", requestURL, map[string]any{"indicator": item.Node.Code, "field": strings.ToUpper(field), "year": year, "regionLevel": regionLevel, "limit": limit})
	payload["summary"] = map[string]any{
		"indicator":        item.Node.Code,
		"field":            strings.ToUpper(field),
		"year":             year,
		"regionLevel":      regionLevel,
		"regionLevelLabel": regionLevelLabel(regionLevel),
		"requestUrl":       requestURL,
		"layerJson":        layerDecoded,
		"doesNotFetch":     true,
	}
	payload["sources"] = sourcesForIndicator(item.Node, field, year)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas sample --indicator %s --field %s --year %d --region-level %d --limit %d", item.Node.Code, strings.ToUpper(field), year, regionLevel, limit),
	}
	emit(payload)
	return nil
}

func runExplainField(argv []string) error {
	parsed := parseArgs(argv)
	code := requiredIndicator(parsed)
	item, err := findIndicator(code)
	if err != nil {
		return err
	}
	field := firstNonEmpty(parsed.flags["field"], parsed.flags["name"], firstPosition(parsed), firstAttributeCode(item.Node))
	attr, ok := findAttribute(item.Node, field)
	if !ok {
		return cliError{2, "field_not_found", "field not found in indicator attributes"}
	}
	grep := firstNonEmpty(parsed.flags["grep"])
	payload := envelope("explain-field", catalogURL, map[string]any{"indicator": item.Node.Code, "field": strings.ToUpper(field), "grep": grep})
	payload["summary"] = map[string]any{
		"indicator": item.Node.Code,
		"field":     strings.ToUpper(attr.Code),
		"title":     attr.TitleShort,
		"titleLong": attr.TitleLong,
		"unit":      attr.Unit,
	}
	payload["items"] = metaSnippets(attr.Meta, grep, 10)
	payload["sources"] = sourcesForIndicator(item.Node, attr.Code, latestYear(item.Node))
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("regionalatlas sample --indicator %s --field %s --year %d --region-level 1 --limit 5", item.Node.Code, strings.ToUpper(attr.Code), latestYear(item.Node)),
	}
	emit(payload)
	return nil
}

func runRawQuery(argv []string) error {
	parsed := parseArgs(argv)
	if parsed.params.Get("layer") == "" && parsed.flags["layer"] == "" && parsed.flags["layer-file"] == "" {
		return cliError{2, "missing_layer", "raw query requires --param layer=<json>, --layer-file <path>, or use query-builder/sample"}
	}
	params := url.Values{}
	for key, values := range parsed.params {
		for _, value := range values {
			params.Set(key, value)
		}
	}
	if parsed.flags["layer-file"] != "" {
		layerBytes, err := os.ReadFile(parsed.flags["layer-file"])
		if err != nil {
			return cliError{2, "layer_file_read_failed", err.Error()}
		}
		params.Set("layer", strings.TrimSpace(strings.TrimPrefix(string(layerBytes), "\ufeff")))
	}
	if parsed.flags["layer"] != "" {
		params.Set("layer", parsed.flags["layer"])
	}
	if params.Get("f") == "" {
		params.Set("f", "json")
	}
	if params.Get("returnGeometry") == "" && params.Get("returngeometry") == "" {
		params.Set("returnGeometry", "false")
	}
	if params.Get("where") == "" {
		params.Set("where", "1=1")
	}
	if params.Get("spatialRel") == "" {
		params.Set("spatialRel", "esriSpatialRelIntersects")
	}
	if params.Get("resultRecordCount") == "" && params.Get("resultrecordcount") == "" {
		params.Set("resultRecordCount", strconv.Itoa(limitFlag(parsed, defaultLimit, safeLimit)))
	}
	if !flagBool(parsed, "allow-large-output") {
		count := intFromString(firstNonEmpty(params.Get("resultRecordCount"), params.Get("resultrecordcount")), 0)
		if count > safeLimit {
			return cliError{2, "limit_exceeds_safe_max", "resultRecordCount exceeds safe max 100; pass --allow-large-output to override"}
		}
	}
	data, err := fetchJSON(queryEndpoint + "?" + params.Encode())
	if err != nil {
		return err
	}
	emit(data)
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

func fetchCatalog() ([]catalogNode, error) {
	body, err := fetchBytes(catalogURL)
	if err != nil {
		return nil, err
	}
	var catalog []catalogNode
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

func fetchJSON(requestURL string) (map[string]any, error) {
	body, err := fetchBytes(requestURL)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode JSON from %s: %w", requestURL, err)
	}
	if upstream, ok := data["error"].(map[string]any); ok {
		return nil, fmt.Errorf("upstream error %v: %v", upstream["code"], upstream["message"])
	}
	return data, nil
}

func fetchBytes(requestURL string) ([]byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "germany-skills/regionalatlas-2.0")
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
	return body, nil
}

func requiredIndicator(parsed parsedArgs) string {
	code := firstNonEmpty(parsed.flags["indicator"], parsed.flags["code"], parsed.flags["table"], firstPosition(parsed))
	if code == "" {
		fail(2, "missing_indicator", "command requires --indicator")
	}
	return strings.ToUpper(code)
}

func findIndicator(code string) (flatIndicator, error) {
	catalog, err := fetchCatalog()
	if err != nil {
		return flatIndicator{}, err
	}
	code = normalizeCode(code)
	for _, item := range flattenCatalog(catalog) {
		if normalizeCode(item.Node.Code) == code {
			return item, nil
		}
	}
	return flatIndicator{}, cliError{2, "indicator_not_found", "indicator not found in Regionalatlas catalog"}
}

func resolveQueryInputs(parsed parsedArgs) (flatIndicator, string, int, int, error) {
	code := requiredIndicator(parsed)
	item, err := findIndicator(code)
	if err != nil {
		return item, "", 0, 0, err
	}
	field := strings.ToLower(firstNonEmpty(parsed.flags["field"], parsed.flags["icode"], firstAttributeCode(item.Node)))
	if _, ok := findAttribute(item.Node, field); !ok {
		return item, "", 0, 0, cliError{2, "field_not_found", "field not found for indicator"}
	}
	year := intFlag(parsed, "year", latestYear(item.Node))
	if year == 0 {
		return item, "", 0, 0, cliError{2, "missing_year", "year could not be inferred; pass --year"}
	}
	regionLevel := intFlag(parsed, "region-level", intFlag(parsed, "typ", 1))
	if !validRegionLevel(regionLevel) {
		return item, "", 0, 0, cliError{2, "invalid_region_level", "region-level must be one of 1, 2, 3, or 5"}
	}
	return item, field, year, regionLevel, nil
}

func buildQueryParams(node catalogNode, field string, year int, regionLevel int, limit int, parsed parsedArgs) url.Values {
	table := tableName(node.Code)
	geoYear := intFlag(parsed, "geo-year", year)
	query := fmt.Sprintf("SELECT * FROM verwaltungsgrenzen_gesamt LEFT OUTER JOIN %s ON ags = ags2 and jahr = jahr2 WHERE typ = %d AND jahr = %d AND (jahr2 = %d OR jahr2 IS NULL)", table, regionLevel, geoYear, year)
	layer := map[string]any{
		"source": map[string]any{
			"dataSource": map[string]any{
				"geometryType":     "esriGeometryPolygon",
				"workspaceId":      "gdb",
				"query":            query,
				"oidFields":        "id",
				"spatialReference": map[string]any{"wkid": 25832},
				"type":             "queryTable",
			},
			"type": "dataLayer",
		},
	}
	layerJSON, _ := json.Marshal(layer)
	outFields := firstNonEmpty(parsed.flags["fields"], parsed.params.Get("outFields"), fmt.Sprintf("ags,gen,typ,jahr,jahr2,ags2,gen2,%s", strings.ToLower(field)))
	where := firstNonEmpty(parsed.flags["where"], parsed.params.Get("where"), "1=1")
	if parsed.flags["ags"] != "" && where == "1=1" {
		where = fmt.Sprintf("ags = '%s'", strings.ReplaceAll(parsed.flags["ags"], "'", "''"))
	}
	params := url.Values{}
	params.Set("layer", string(layerJSON))
	params.Set("f", "json")
	params.Set("outFields", outFields)
	params.Set("returnGeometry", boolString(flagBool(parsed, "geometry")))
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("where", where)
	params.Set("resultRecordCount", strconv.Itoa(limit))
	for key, values := range parsed.params {
		for _, value := range values {
			if key != "layer" {
				params.Set(key, value)
			}
		}
	}
	return params
}

func flattenCatalog(catalog []catalogNode) []flatIndicator {
	flat := []flatIndicator{}
	var walk func(nodes []catalogNode, topic string)
	walk = func(nodes []catalogNode, topic string) {
		for _, node := range nodes {
			nextTopic := topic
			if node.Title != "" && node.Code == "" {
				nextTopic = node.Title
			}
			if node.Code != "" && len(node.Attributes) > 0 {
				flat = append(flat, flatIndicator{Topic: nextTopic, Node: node})
			}
			if len(node.Children) > 0 {
				walk(node.Children, nextTopic)
			}
		}
	}
	walk(catalog, "")
	sort.Slice(flat, func(i, j int) bool { return flat[i].Node.Code < flat[j].Node.Code })
	return flat
}

func searchCatalog(flat []flatIndicator, term string) []flatIndicator {
	needle := strings.ToLower(term)
	matches := []flatIndicator{}
	for _, item := range flat {
		hay := strings.ToLower(strings.Join([]string{item.Topic, item.Node.Code, item.Node.TitleShort, item.Node.TitleLong}, " "))
		for _, attr := range item.Node.Attributes {
			hay += " " + strings.ToLower(strings.Join([]string{attr.Code, attr.TitleShort, attr.TitleLong, attr.Unit, stripWiki(attr.Meta)}, " "))
		}
		if strings.Contains(hay, needle) {
			matches = append(matches, item)
		}
	}
	return matches
}

func compactIndicators(items []flatIndicator, limit int) []any {
	out := []any{}
	for i, item := range items {
		if i >= limit {
			break
		}
		firstField := firstAttributeCode(item.Node)
		out = append(out, map[string]any{
			"code":           item.Node.Code,
			"table":          tableName(item.Node.Code),
			"topic":          item.Topic,
			"title":          item.Node.TitleShort,
			"titleLong":      item.Node.TitleLong,
			"latestYear":     latestYear(item.Node),
			"availableYears": availableYears(item.Node),
			"attributes":     compactAttributes(item.Node.Attributes, 8, ""),
			"nextActions": []string{
				fmt.Sprintf("regionalatlas fields --indicator %s", item.Node.Code),
				fmt.Sprintf("regionalatlas sample --indicator %s --field %s --year %d --region-level 1 --limit 5", item.Node.Code, firstField, latestYear(item.Node)),
			},
		})
	}
	return out
}

func compactAttributes(attrs []indicatorAttribute, limit int, grep string) []any {
	out := []any{}
	for i, attr := range attrs {
		if i >= limit {
			break
		}
		item := map[string]any{
			"code":        strings.ToUpper(attr.Code),
			"field":       strings.ToLower(attr.Code),
			"title":       attr.TitleShort,
			"titleLong":   attr.TitleLong,
			"unit":        attr.Unit,
			"metaPreview": truncate(stripWiki(attr.Meta), 500),
		}
		if grep != "" {
			item["snippets"] = metaSnippets(attr.Meta, grep, 5)
		}
		out = append(out, item)
	}
	return out
}

func compactFeatures(data map[string]any, includeGeometry bool) []any {
	items := []any{}
	for _, featureAny := range asSlice(data["features"]) {
		feature := asMap(featureAny)
		item := map[string]any{"attributes": normalizeAttributes(asMap(feature["attributes"]))}
		if includeGeometry {
			item["geometry"] = feature["geometry"]
		}
		items = append(items, item)
	}
	return items
}

func normalizeAttributes(attrs map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range attrs {
		if strings.HasSuffix(strings.ToLower(key), ".shape") {
			continue
		}
		if str, ok := value.(string); ok {
			out[key] = strings.TrimSpace(str)
		} else {
			out[key] = value
		}
	}
	return out
}

func indicatorSummary(item flatIndicator) map[string]any {
	return map[string]any{
		"code":           item.Node.Code,
		"table":          tableName(item.Node.Code),
		"topic":          item.Topic,
		"title":          item.Node.TitleShort,
		"titleLong":      item.Node.TitleLong,
		"timestamp":      item.Node.Timestamp,
		"latestYear":     latestYear(item.Node),
		"availableYears": availableYears(item.Node),
		"regionLevels":   regionLevelAvailability(item.Node),
		"attributeCount": len(item.Node.Attributes),
	}
}

func availableYears(node catalogNode) []int {
	years := []int{}
	for key := range node.Years {
		if y, err := strconv.Atoi(key); err == nil {
			years = append(years, y)
		}
	}
	sort.Ints(years)
	return years
}

func latestYear(node catalogNode) int {
	years := availableYears(node)
	if len(years) == 0 {
		return 0
	}
	return years[len(years)-1]
}

func regionLevelAvailability(node catalogNode) map[string]any {
	out := map[string]any{}
	latest := latestYear(node)
	if latest == 0 {
		return out
	}
	infos := node.Years[strconv.Itoa(latest)]
	if len(infos) == 0 {
		return out
	}
	for _, level := range []int{1, 2, 3, 5} {
		available := false
		for _, info := range infos {
			for _, count := range info.GeomLevels {
				if count > 0 && level == geomIndexToRegionLevel(indexOf(info.GeomLevels, count)) {
					available = true
				}
			}
			if info.GeomLevel == level {
				available = true
			}
		}
		out[strconv.Itoa(level)] = map[string]any{"label": regionLevelLabel(level), "appearsAvailableLatestYear": available}
	}
	return out
}

func indexOf(values []int, value int) int {
	for i, candidate := range values {
		if candidate == value {
			return i
		}
	}
	return -1
}

func geomIndexToRegionLevel(index int) int {
	switch index {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 3
	case 3:
		return 5
	default:
		return 0
	}
}

func findAttribute(node catalogNode, code string) (indicatorAttribute, bool) {
	code = strings.ToLower(code)
	for _, attr := range node.Attributes {
		if strings.ToLower(attr.Code) == code {
			return attr, true
		}
	}
	return indicatorAttribute{}, false
}

func firstAttributeCode(node catalogNode) string {
	for _, attr := range node.Attributes {
		if attr.Code != "" && !strings.HasSuffix(strings.ToLower(attr.Code), "v") {
			return strings.ToUpper(attr.Code)
		}
	}
	if len(node.Attributes) > 0 {
		return strings.ToUpper(node.Attributes[0].Code)
	}
	return ""
}

func attributeTitle(node catalogNode, code string) string {
	if attr, ok := findAttribute(node, code); ok {
		return attr.TitleShort
	}
	return ""
}

func attributeUnit(node catalogNode, code string) string {
	if attr, ok := findAttribute(node, code); ok {
		return attr.Unit
	}
	return ""
}

func attributeMeta(node catalogNode, code string) string {
	if attr, ok := findAttribute(node, code); ok {
		return attr.Meta
	}
	return ""
}

func metaSnippets(meta string, grep string, limit int) []any {
	clean := stripWiki(meta)
	if clean == "" {
		return []any{}
	}
	lines := splitMeaningfulLines(clean)
	out := []any{}
	needle := strings.ToLower(grep)
	for _, line := range lines {
		if needle == "" || strings.Contains(strings.ToLower(line), needle) {
			out = append(out, map[string]any{"text": truncate(line, 700)})
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func splitMeaningfulLines(value string) []string {
	raw := strings.Split(value, "\n")
	lines := []string{}
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if len(line) > 10 {
			lines = append(lines, line)
		}
	}
	return lines
}

func stripWiki(value string) string {
	value = strings.TrimPrefix(value, "wiki")
	replacements := []string{"==", "", "===", "", "'''", "", "''", "", "*", "", "[", "", "]", ""}
	replacer := strings.NewReplacer(replacements...)
	value = replacer.Replace(value)
	value = regexp.MustCompile(`\s+\|`).ReplaceAllString(value, " |")
	return strings.TrimSpace(value)
}

func sourcesForIndicator(node catalogNode, field string, year int) []any {
	appDeepLink := appURL + "?" + url.Values{"BL": {"DE"}, "TCode": {node.Code}, "ICode": {strings.ToUpper(field)}, "Jhr": {strconv.Itoa(year)}}.Encode()
	return []any{
		map[string]any{"title": "Regionalatlas app", "url": appDeepLink, "kind": "interactive_atlas"},
		map[string]any{"title": "Regionalatlas Statistikportal page", "url": statistikportalURL, "kind": "official_context"},
		map[string]any{"title": "Destatis Regionalatlas page", "url": destatisURL, "kind": "official_context"},
		map[string]any{"title": "Statistikportal Open Data", "url": openDataURL, "kind": "terms_and_downloads"},
		map[string]any{"title": "Destatis maps and geodata", "url": mapsGeodataURL, "kind": "terms_and_downloads"},
		map[string]any{"title": "Regionalatlas catalog JSON", "url": catalogURL, "kind": "catalog"},
		map[string]any{"title": "Regionaldatenbank table", "url": "https://www.regionalstatistik.de/genesis/online/data?operation=table&code=" + node.Code, "kind": "official_table"},
		map[string]any{"title": "ArcGIS dynamic-layer query endpoint", "url": queryEndpoint, "kind": "api_endpoint"},
		map[string]any{"title": "bundesAPI Regionalatlas OpenAPI wrapper", "url": openAPIRepoURL, "kind": "openapi_reference"},
	}
}

func defaultSources() []any {
	return []any{
		map[string]any{"title": "Regionalatlas Statistikportal page", "url": statistikportalURL, "kind": "official_context"},
		map[string]any{"title": "Destatis Regionalatlas page", "url": destatisURL, "kind": "official_context"},
		map[string]any{"title": "Statistikportal Open Data", "url": openDataURL, "kind": "terms_and_downloads"},
		map[string]any{"title": "Regionalatlas catalog JSON", "url": catalogURL, "kind": "catalog"},
		map[string]any{"title": "Regionalatlas thesaurus CSV", "url": thesaurusURL, "kind": "catalog"},
		map[string]any{"title": "ArcGIS MapServer metadata", "url": mapServerURL + "?f=json", "kind": "api_metadata"},
		map[string]any{"title": "bundesAPI Regionalatlas OpenAPI wrapper", "url": openAPIRepoURL, "kind": "openapi_reference"},
	}
}

func defaultWarnings() []string {
	return []string{
		"No exact published API rate limit was found in reviewed materials; keep requests small and cache catalog metadata.",
		"The ArcGIS service advertises a very high maxRecordCount; never run broad municipality-level pulls accidentally.",
		"Use field metadata for units, definitions, source statistics, and regional caveats before interpreting values.",
		"Statistikportal Open Data notes point to Datenlizenz Deutschland 2.0 for statistical data and atlas/imprint license hints for geodata.",
	}
}

func nextActionsForIndicators(items []flatIndicator) []string {
	actions := []string{}
	for i, item := range items {
		if i >= 3 {
			break
		}
		actions = append(actions, fmt.Sprintf("regionalatlas dossier --indicator %s --field %s --year %d --region-level 1", item.Node.Code, firstAttributeCode(item.Node), latestYear(item.Node)))
	}
	if len(actions) == 0 {
		return []string{`regionalatlas indicators search --term "Bevoelkerung" --limit 5`}
	}
	return actions
}

func mapServerSummary(data map[string]any) map[string]any {
	return map[string]any{
		"mapName":                stringAt(data, "mapName"),
		"supportsDynamicLayers":  boolAt(data, "supportsDynamicLayers"),
		"supportedQueryFormats":  stringAt(data, "supportedQueryFormats"),
		"maxRecordCount":         intAt(data, "maxRecordCount"),
		"capabilities":           stringAt(data, "capabilities"),
		"featureLayerCount":      len(asSlice(data["layers"])),
		"spatialReferenceLatest": intAt(asMap(data["spatialReference"]), "latestWkid"),
	}
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
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
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

func isHelp(value string) bool { return value == "--help" || value == "-h" || value == "help" }

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

func limitFlag(parsed parsedArgs, fallback int, max int) int {
	raw := firstNonEmpty(parsed.flags["limit"], parsed.flags["resultrecordcount"], parsed.params.Get("resultRecordCount"), parsed.params.Get("resultrecordcount"))
	if raw == "" {
		return fallback
	}
	value := intFromString(raw, fallback)
	if value < 1 {
		value = fallback
	}
	if value > max && !flagBool(parsed, "allow-large-output") {
		fail(2, "limit_exceeds_safe_max", fmt.Sprintf("limit %d exceeds safe max %d; pass --allow-large-output to override", value, max))
	}
	return value
}

func intFlag(parsed parsedArgs, key string, fallback int) int {
	return intFromString(parsed.flags[key], fallback)
}

func intFromString(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
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

func tableName(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "_", "-"))
}

func validRegionLevel(level int) bool { return level == 1 || level == 2 || level == 3 || level == 5 }

func regionLevelLabel(level int) string {
	switch level {
	case 1:
		return "Laender"
	case 2:
		return "Regierungsbezirke/statistical regions"
	case 3:
		return "Kreise and kreisfreie Staedte"
	case 5:
		return "Gemeinden/Gemeindeverbaende"
	default:
		return "unknown"
	}
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
		return intFromString(typed, 0)
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

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
