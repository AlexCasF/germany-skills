package main

import (
	"bytes"
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
	appName        = "dashboardctl"
	baseURL        = "https://www.dashboard-deutschland.de"
	dashboardsURL  = baseURL + "/api/dashboard/get"
	indicatorsURL  = baseURL + "/api/tile/indicators"
	geoURL         = baseURL + "/geojson/de-all.geo.json"
	destatisURL    = "https://www.destatis.de/DE/Ueber-uns/Aufgaben/dashboards.html"
	bmweURL        = "https://www.bundeswirtschaftsministerium.de/Redaktion/DE/Dossier/WirtschaftlicheEntwicklung/dashboard-deutschland.html"
	openDataURL    = "https://www.statistikportal.de/de/open-data"
	pypiURL        = "https://pypi.org/project/de-dashboarddeutschland/"
	openAPIRepoURL = "https://github.com/AndreasFischer1985/dashboard-deutschland-api"
	defaultTimeout = 45 * time.Second
	defaultLimit   = 10
	safeLimit      = 100
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

func (e cliError) Error() string {
	return e.message
}

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
	case match(args, "dashboard", "get"):
		err = runDashboardGet(args[2:])
	case match(args, "dashboards", "list"):
		err = runDashboardsList(args[2:])
	case match(args, "dashboard", "dossier"):
		err = runDashboardDossier(args[2:])
	case args[0] == "indicators":
		err = runIndicatorsRaw(args[1:])
	case match(args, "indicator", "search"):
		err = runIndicatorSearch(args[2:])
	case match(args, "indicator", "get"):
		err = runIndicatorGet(args[2:])
	case match(args, "indicator", "data"):
		err = runIndicatorData(args[2:])
	case match(args, "indicator", "source"):
		err = runIndicatorSource(args[2:])
	case args[0] == "source":
		err = runIndicatorSource(args[1:])
	case args[0] == "geo":
		err = runGeo(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run dashboardctl --help"}
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
	fmt.Println(`dashboardctl -- Dashboard Deutschland research CLI

Purpose
  Discover and normalize curated Dashboard Deutschland indicators from the
  public dashboard API run for Destatis' dashboard offering.

Use this when
  - you need high-level dashboard indicators across economics, labor, energy,
    mobility, housing, finance, health, and related topics
  - you need chart-ready series, source links, update dates, or compact widgets
  - a curated dashboard tile is sufficient evidence

Do not use this when
  - you need deep configurable statistical tables; use destatisctl
  - you need regional atlas tables; use deutschlandatlasctl or regionalatlasctl

Fast paths
  dashboardctl doctor
  dashboardctl dashboards list --limit 5
  dashboardctl indicator search --term "Arbeitslosigkeit" --limit 5
  dashboardctl indicator get --id tile_1666958835081
  dashboardctl indicator data --id tile_1666958835081 --limit 5
  dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3

Legacy-compatible commands
  dashboard get [--param key=value]
  indicators --param ids=tile_1666958835081
  geo

Research commands
  doctor
  dashboards list
  dashboard dossier
  indicator search
  indicator get
  indicator data
  indicator source
  source

Output guarantees
  Research commands emit JSON envelopes with status, request, summary/items,
  sources, warnings, and nextActions. Legacy commands return upstream JSON.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "indicator data":
		fmt.Println(`dashboardctl indicator data

Extract chart-ready series from a Dashboard Deutschland indicator tile.

Examples
  dashboardctl indicator data --id tile_1666958835081 --limit 5
  dashboardctl indicator data --id tile_1666958835081 --series "Arbeitslose" --limit 10

Flags
  --id <indicator-id>       Required indicator ID
  --limit <n>              Points per series, defaults to 10, max 100
  --series <term>          Filter series by display name or ID
  --from-start             Return earliest points instead of latest points
  --include-raw            Include parsed tile config`)
	case "dashboard dossier":
		fmt.Println(`dashboardctl dashboard dossier

Bundle dashboard metadata and a small set of normalized indicator summaries.

Examples
  dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3
  dashboardctl dashboard dossier --name Arbeitsmarkt --indicator-limit 3`)
	case "geo":
		fmt.Println(`dashboardctl geo

Legacy GeoJSON endpoint wrapper. The documented endpoint currently returned
403 AccessDenied in live tests; doctor reports this as degraded.`)
	default:
		printRootHelp()
	}
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, 2, 10)
	payload := envelope("doctor", dashboardsURL, nil)
	warnings := defaultWarnings()
	summary := map[string]any{
		"authRequired":       false,
		"publishedRateLimit": "No exact public rate limit was found in reviewed materials. Use small batches and avoid repeated all-indicator pulls.",
		"fairUseHints": []string{
			"Use dashboards list or indicator search before fetching indicator data.",
			"Fetch indicator data by explicit ID.",
			"Use small --limit values for chart points.",
			"Back off on 429, 5xx, or CloudFront/S3 errors.",
		},
	}

	dashboards, err := fetchDashboards()
	if err != nil {
		summary["dashboardEndpoint"] = map[string]any{"ok": false, "error": err.Error()}
		payload["status"] = "degraded"
	} else {
		ids := uniqueIndicatorIDs(dashboards)
		summary["dashboardEndpoint"] = map[string]any{"ok": true, "dashboards": len(dashboards), "uniqueIndicatorIds": len(ids), "sampleDashboards": compactDashboards(dashboards, limit)}
		if len(ids) > 0 {
			indicators, indErr := fetchIndicators(ids[:minInt(1, len(ids))])
			if indErr != nil {
				summary["indicatorEndpoint"] = map[string]any{"ok": false, "error": indErr.Error()}
				payload["status"] = "degraded"
			} else {
				summary["indicatorEndpoint"] = map[string]any{"ok": true, "sample": compactIndicators(indicators, 1)}
			}
		}
	}

	status, _, body, geoErr := fetchRaw(geoURL)
	geoSummary := map[string]any{"url": geoURL, "statusCode": status}
	if geoErr != nil {
		geoSummary["ok"] = false
		geoSummary["error"] = geoErr.Error()
		geoSummary["bodyPreview"] = truncate(stripSpace(string(body)), 180)
		warnings = append(warnings, "The documented GeoJSON endpoint currently returns 403 AccessDenied; use geo as a legacy diagnostic command.")
	} else {
		geoSummary["ok"] = status >= 200 && status < 300
	}
	summary["geoEndpoint"] = geoSummary

	if payload["status"] == nil {
		payload["status"] = "ok"
	}
	if geoErr != nil {
		payload["status"] = "degraded"
	}
	payload["summary"] = summary
	payload["sources"] = defaultSources()
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		`dashboardctl indicator search --term "Arbeitslosigkeit" --limit 5`,
		"dashboardctl dashboards list --limit 5",
	}
	emit(payload)
	return nil
}

func runDashboardGet(argv []string) error {
	params := parseArgs(argv).params
	data, err := fetchJSON(withParams(dashboardsURL, params))
	if err != nil {
		return err
	}
	emit(data)
	return nil
}

func runIndicatorsRaw(argv []string) error {
	parsed := parseArgs(argv)
	params := parsed.params
	if parsed.flags["id"] != "" {
		params.Set("ids", parsed.flags["id"])
	}
	if parsed.flags["ids"] != "" {
		params.Set("ids", parsed.flags["ids"])
	}
	data, err := fetchJSON(withParams(indicatorsURL, params))
	if err != nil {
		return err
	}
	emit(data)
	return nil
}

func runGeo(argv []string) error {
	status, contentType, body, err := fetchRaw(geoURL)
	if err != nil {
		return cliError{1, "geo_endpoint_failed", fmt.Sprintf("geo endpoint status %d content-type %s body: %s", status, contentType, truncate(stripSpace(string(body)), 220))}
	}
	var data any
	if json.Unmarshal(body, &data) == nil {
		emit(data)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func runDashboardsList(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, defaultLimit, 50)
	term := strings.ToLower(firstNonEmpty(parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " ")))
	dashboards, err := fetchDashboards()
	if err != nil {
		return err
	}
	filtered := dashboards
	if term != "" {
		filtered = nil
		for _, item := range dashboards {
			if strings.Contains(strings.ToLower(dashboardSearchText(item)), term) {
				filtered = append(filtered, item)
			}
		}
	}
	payload := envelope("dashboards list", dashboardsURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"available": len(filtered), "returned": minInt(limit, len(filtered)), "totalDashboards": len(dashboards), "uniqueIndicatorIds": len(uniqueIndicatorIDs(dashboards))}
	payload["items"] = compactDashboards(filtered, limit)
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"dashboardctl dashboard dossier --id arbeitsmarkt --indicator-limit 3"}
	emit(payload)
	return nil
}

func runIndicatorSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "indicator search requires --term"}
	}
	limit := limitFlag(parsed, 5, 50)
	dashboards, err := fetchDashboards()
	if err != nil {
		return err
	}
	ids := uniqueIndicatorIDs(dashboards)
	indicators, err := fetchIndicators(ids)
	if err != nil {
		return err
	}
	needle := strings.ToLower(term)
	var matches []map[string]any
	for _, indicator := range indicators {
		if strings.Contains(strings.ToLower(indicatorSearchText(indicator)), needle) {
			matches = append(matches, indicator)
		}
	}
	payload := envelope("indicator search", indicatorsURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"term": term, "matches": len(matches), "searchedIndicatorIds": len(ids), "returned": minInt(limit, len(matches))}
	payload["items"] = compactIndicators(matches, limit)
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForIndicators(matches)
	emit(payload)
	return nil
}

func runIndicatorGet(argv []string) error {
	parsed := parseArgs(argv)
	id, err := requiredID(parsed)
	if err != nil {
		return err
	}
	indicator, err := fetchOneIndicator(id)
	if err != nil {
		return err
	}
	config, parseErr := parseTileConfig(indicator)
	warnings := defaultWarnings()
	if parseErr != nil {
		warnings = append(warnings, "Could not parse embedded tile json: "+parseErr.Error())
	}
	payload := envelope("indicator get", indicatorsURL+"?ids="+url.QueryEscape(id), map[string]any{"id": id})
	payload["summary"] = indicatorSummary(indicator, config)
	payload["items"] = []any{indicatorMetadata(indicator, config)}
	payload["sources"] = sourcesForIndicator(indicator, config)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{
		fmt.Sprintf("dashboardctl indicator data --id %s --limit 10", id),
		fmt.Sprintf("dashboardctl indicator source --id %s", id),
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = map[string]any{"indicator": indicator, "config": config}
	}
	emit(payload)
	return nil
}

func runIndicatorData(argv []string) error {
	parsed := parseArgs(argv)
	id, err := requiredID(parsed)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	seriesTerm := strings.ToLower(firstNonEmpty(parsed.flags["series"], parsed.flags["grep"]))
	indicator, err := fetchOneIndicator(id)
	if err != nil {
		return err
	}
	config, err := parseTileConfig(indicator)
	if err != nil {
		return err
	}
	series := extractSeries(config, limit, flagBool(parsed, "from-start"), seriesTerm)
	payload := envelope("indicator data", indicatorsURL+"?ids="+url.QueryEscape(id), map[string]any{"id": id, "limit": limit, "series": seriesTerm})
	payload["summary"] = map[string]any{
		"id":              id,
		"title":           firstNonEmpty(stringAt(config, "title"), stringAt(indicator, "title")),
		"seriesReturned":  len(series),
		"pointsPerSeries": limit,
		"dataVersionDate": stringAt(config, "dataVersionDate"),
		"lastUpdated":     millisSummary(config["lastUpdated"]),
	}
	payload["items"] = series
	payload["sources"] = sourcesForIndicator(indicator, config)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{fmt.Sprintf("dashboardctl indicator source --id %s", id)}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = config
	}
	emit(payload)
	return nil
}

func runIndicatorSource(argv []string) error {
	parsed := parseArgs(argv)
	id, err := requiredID(parsed)
	if err != nil {
		return err
	}
	indicator, err := fetchOneIndicator(id)
	if err != nil {
		return err
	}
	config, parseErr := parseTileConfig(indicator)
	warnings := defaultWarnings()
	if parseErr != nil {
		warnings = append(warnings, "Could not parse embedded tile json: "+parseErr.Error())
	}
	payload := envelope("indicator source", indicatorsURL+"?ids="+url.QueryEscape(id), map[string]any{"id": id})
	payload["summary"] = map[string]any{"id": id, "title": firstNonEmpty(stringAt(config, "title"), stringAt(indicator, "title")), "sourceCount": len(sourcesForIndicator(indicator, config))}
	payload["sources"] = sourcesForIndicator(indicator, config)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{fmt.Sprintf("dashboardctl indicator data --id %s --limit 10", id)}
	emit(payload)
	return nil
}

func runDashboardDossier(argv []string) error {
	parsed := parseArgs(argv)
	indicatorLimit := limitFlagName(parsed, "indicator-limit", 3, 10)
	dashboards, err := fetchDashboards()
	if err != nil {
		return err
	}
	dashboard, err := findDashboard(dashboards, parsed)
	if err != nil {
		return err
	}
	ids := dashboardIndicatorIDs(dashboard)
	if indicatorLimit < len(ids) {
		ids = ids[:indicatorLimit]
	}
	indicators, indErr := fetchIndicators(ids)
	warnings := defaultWarnings()
	if indErr != nil {
		warnings = append(warnings, "indicator fetch failed: "+indErr.Error())
	}
	payload := envelope("dashboard dossier", dashboardsURL, map[string]any{"id": stringAt(dashboard, "id"), "indicatorLimit": indicatorLimit})
	payload["summary"] = compactDashboard(dashboard)
	payload["items"] = compactIndicators(indicators, len(indicators))
	payload["sources"] = sourcesForDashboard(dashboard)
	payload["warnings"] = warnings
	payload["nextActions"] = []string{}
	for _, id := range ids[:minInt(3, len(ids))] {
		payload["nextActions"] = append(payload["nextActions"].([]string), fmt.Sprintf("dashboardctl indicator data --id %s --limit 10", id))
	}
	emit(payload)
	return nil
}

func fetchDashboards() ([]map[string]any, error) {
	data, err := fetchJSON(dashboardsURL)
	if err != nil {
		return nil, err
	}
	return asObjectSlice(data), nil
}

func fetchOneIndicator(id string) (map[string]any, error) {
	items, err := fetchIndicators([]string{id})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, cliError{2, "indicator_not_found", "indicator not found: " + id}
	}
	return items[0], nil
}

func fetchIndicators(ids []string) ([]map[string]any, error) {
	if len(ids) == 0 {
		return nil, cliError{2, "missing_ids", "indicator IDs required"}
	}
	var all []map[string]any
	for start := 0; start < len(ids); start += 20 {
		end := minInt(start+20, len(ids))
		params := url.Values{"ids": []string{strings.Join(ids[start:end], ";")}}
		data, err := fetchJSON(withParams(indicatorsURL, params))
		if err != nil {
			return nil, err
		}
		all = append(all, asObjectSlice(data)...)
	}
	return all, nil
}

func fetchJSON(requestURL string) (any, error) {
	status, _, body, err := fetchRaw(requestURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, httpError{status, string(body), requestURL}
	}
	var data any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func fetchRaw(requestURL string) (int, string, []byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("User-Agent", "democracy-researcher/dashboardctl-2.0")
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

func parseTileConfig(indicator map[string]any) (map[string]any, error) {
	raw := stringAt(indicator, "json")
	if raw == "" {
		return map[string]any{}, cliError{2, "missing_embedded_json", "indicator has no embedded json field"}
	}
	var config map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return map[string]any{}, err
	}
	return config, nil
}

func compactDashboards(dashboards []map[string]any, limit int) []map[string]any {
	out := []map[string]any{}
	for _, dashboard := range dashboards[:minInt(limit, len(dashboards))] {
		out = append(out, compactDashboard(dashboard))
	}
	return out
}

func compactDashboard(dashboard map[string]any) map[string]any {
	ids := dashboardIndicatorIDs(dashboard)
	return map[string]any{
		"id":             stringAt(dashboard, "id"),
		"name":           stringAt(dashboard, "name"),
		"nameEn":         stringAt(dashboard, "nameEn"),
		"description":    truncate(stripHTML(stringAt(dashboard, "description")), 420),
		"category":       compactCategory(objectAt(dashboard, "category")),
		"tags":           stringSliceAt(dashboard, "tags"),
		"indicatorCount": len(ids),
		"indicatorIds":   ids[:minInt(12, len(ids))],
		"nextActions": []string{
			fmt.Sprintf("dashboardctl dashboard dossier --id %s --indicator-limit 3", stringAt(dashboard, "id")),
		},
	}
}

func compactIndicators(indicators []map[string]any, limit int) []map[string]any {
	out := []map[string]any{}
	for _, indicator := range indicators[:minInt(limit, len(indicators))] {
		config, _ := parseTileConfig(indicator)
		summary := indicatorSummary(indicator, config)
		summary["nextActions"] = []string{
			fmt.Sprintf("dashboardctl indicator data --id %s --limit 10", stringAt(indicator, "id")),
			fmt.Sprintf("dashboardctl indicator source --id %s", stringAt(indicator, "id")),
		}
		out = append(out, summary)
	}
	return out
}

func indicatorSummary(indicator, config map[string]any) map[string]any {
	sources := sourceEntries(config)
	return map[string]any{
		"id":              stringAt(indicator, "id"),
		"title":           firstNonEmpty(stringAt(config, "title"), stringAt(indicator, "title")),
		"apiTitle":        stringAt(indicator, "title"),
		"category":        stringAt(config, "category"),
		"tags":            stringSliceAt(config, "tags"),
		"sourceCount":     len(sources),
		"sources":         sources,
		"componentCount":  len(asObjectSlice(config["components"])),
		"seriesCount":     countSeries(config),
		"widgetCount":     countWidgets(config),
		"dataVersionDate": stringAt(config, "dataVersionDate"),
		"dateUpload":      stringAt(config, "dateUpload"),
		"lastUpdated":     millisSummary(config["lastUpdated"]),
	}
}

func indicatorMetadata(indicator, config map[string]any) map[string]any {
	return map[string]any{
		"summary":      indicatorSummary(indicator, config),
		"textSnippets": textSnippets(config, "", 5),
		"widgets":      widgets(config),
		"chartSeries":  seriesSummaries(config),
	}
}

func extractSeries(config map[string]any, limit int, fromStart bool, seriesTerm string) []map[string]any {
	var out []map[string]any
	for _, component := range asObjectSlice(config["components"]) {
		chart := objectAt(component, "chart")
		for _, series := range asObjectSlice(chart["series"]) {
			name := firstNonEmpty(stringAt(objectAt(series, "custom"), "name"), stringAt(series, "name"))
			id := stringAt(series, "id")
			if seriesTerm != "" && !strings.Contains(strings.ToLower(name+" "+id), seriesTerm) {
				continue
			}
			points := asArray(series["data"])
			selected := selectPoints(points, limit, fromStart)
			out = append(out, map[string]any{
				"id":         id,
				"name":       name,
				"color":      stringAt(series, "color"),
				"pointCount": len(points),
				"points":     selected,
				"firstPoint": firstPoint(points),
				"lastPoint":  lastPoint(points),
			})
		}
	}
	return out
}

func seriesSummaries(config map[string]any) []map[string]any {
	var out []map[string]any
	for _, component := range asObjectSlice(config["components"]) {
		chart := objectAt(component, "chart")
		for _, series := range asObjectSlice(chart["series"]) {
			points := asArray(series["data"])
			out = append(out, map[string]any{
				"id":         stringAt(series, "id"),
				"name":       firstNonEmpty(stringAt(objectAt(series, "custom"), "name"), stringAt(series, "name")),
				"pointCount": len(points),
				"firstPoint": firstPoint(points),
				"lastPoint":  lastPoint(points),
			})
		}
	}
	return out
}

func selectPoints(points []any, limit int, fromStart bool) []any {
	if len(points) <= limit {
		return points
	}
	if fromStart {
		return points[:limit]
	}
	return points[len(points)-limit:]
}

func firstPoint(points []any) any {
	if len(points) == 0 {
		return nil
	}
	return points[0]
}

func lastPoint(points []any) any {
	if len(points) == 0 {
		return nil
	}
	return points[len(points)-1]
}

func countSeries(config map[string]any) int {
	return len(seriesSummaries(config))
}

func countWidgets(config map[string]any) int {
	count := 0
	for _, component := range asObjectSlice(config["components"]) {
		count += len(asArray(component["widgets"]))
	}
	return count
}

func widgets(config map[string]any) []map[string]any {
	var out []map[string]any
	for _, component := range asObjectSlice(config["components"]) {
		for _, widget := range asObjectSlice(component["widgets"]) {
			out = append(out, map[string]any{
				"num":  stringAt(widget, "num"),
				"desc": stripHTML(stringAt(widget, "desc")),
				"icon": stringAt(widget, "icon"),
			})
		}
	}
	return out
}

func textSnippets(config map[string]any, grep string, limit int) []map[string]any {
	needle := strings.ToLower(grep)
	var snippets []map[string]any
	for _, component := range asObjectSlice(config["components"]) {
		text := stripHTML(firstNonEmpty(stringAt(component, "text"), stringAt(component, "infoButtonText"), stringAt(component, "description")))
		if len(text) < 20 {
			continue
		}
		if needle == "" || strings.Contains(strings.ToLower(text), needle) {
			snippets = append(snippets, map[string]any{"text": truncate(text, 700), "type": stringAt(component, "type")})
		}
		if len(snippets) >= limit {
			break
		}
	}
	return snippets
}

func sourceEntries(config map[string]any) []map[string]any {
	var out []map[string]any
	for _, source := range asObjectSlice(config["sources"]) {
		out = append(out, map[string]any{
			"title":   firstNonEmpty(stringAt(source, "name"), "Dashboard Deutschland source"),
			"url":     stringAt(source, "link"),
			"kind":    "indicator_source",
			"quality": source["quality"],
		})
	}
	if stringAt(config, "source") != "" && len(out) == 0 {
		out = append(out, map[string]any{"title": "Dashboard source field", "url": "", "kind": "source_text", "text": stripHTML(stringAt(config, "source"))})
	}
	return out
}

func sourcesForIndicator(indicator, config map[string]any) []map[string]any {
	sources := []map[string]any{
		{"title": "Dashboard Deutschland indicator API", "url": indicatorsURL + "?ids=" + url.QueryEscape(stringAt(indicator, "id")), "kind": "api_endpoint"},
		{"title": "Dashboard Deutschland", "url": baseURL, "kind": "official_dashboard"},
	}
	sources = append(sources, sourceEntries(config)...)
	return sources
}

func sourcesForDashboard(dashboard map[string]any) []map[string]any {
	return []map[string]any{
		{"title": "Dashboard Deutschland dashboard API", "url": dashboardsURL, "kind": "api_endpoint"},
		{"title": "Dashboard Deutschland", "url": baseURL, "kind": "official_dashboard"},
		{"title": "Destatis dashboard page", "url": destatisURL, "kind": "official_context"},
	}
}

func defaultSources() []map[string]any {
	return []map[string]any{
		{"title": "Dashboard Deutschland", "url": baseURL, "kind": "official_dashboard"},
		{"title": "Dashboard Deutschland dashboard API", "url": dashboardsURL, "kind": "api_endpoint"},
		{"title": "Dashboard Deutschland indicator API", "url": indicatorsURL, "kind": "api_endpoint"},
		{"title": "Dashboard Deutschland GeoJSON endpoint", "url": geoURL, "kind": "api_endpoint"},
		{"title": "Destatis dashboards page", "url": destatisURL, "kind": "official_context"},
		{"title": "BMWE Dashboard Deutschland page", "url": bmweURL, "kind": "official_context"},
		{"title": "PyPI generated DashboardDeutschland package", "url": pypiURL, "kind": "openapi_reference"},
		{"title": "Dashboard Deutschland OpenAPI wrapper", "url": openAPIRepoURL, "kind": "openapi_reference"},
	}
}

func defaultWarnings() []string {
	return []string{
		"No exact published API rate limit was found in reviewed materials; use small batches and avoid repeated all-indicator pulls.",
		"Indicator tiles contain an embedded JSON string; parse it before interpreting chart data, sources, widgets, or update dates.",
		"The documented GeoJSON endpoint returned 403 AccessDenied in live tests.",
		"Dashboard Deutschland is curated and mixed-source; for deep statistical table work use Destatis/GENESIS where appropriate.",
	}
}

func findDashboard(dashboards []map[string]any, parsed parsedArgs) (map[string]any, error) {
	wanted := strings.ToLower(firstNonEmpty(parsed.flags["id"], parsed.flags["name"], strings.Join(parsed.positionals, " ")))
	if wanted == "" {
		return nil, cliError{2, "missing_dashboard", "dashboard dossier requires --id or --name"}
	}
	for _, dashboard := range dashboards {
		if strings.ToLower(stringAt(dashboard, "id")) == wanted || strings.Contains(strings.ToLower(stringAt(dashboard, "name")), wanted) {
			return dashboard, nil
		}
	}
	return nil, cliError{2, "dashboard_not_found", "dashboard not found: " + wanted}
}

func uniqueIndicatorIDs(dashboards []map[string]any) []string {
	seen := map[string]bool{}
	var ids []string
	for _, dashboard := range dashboards {
		for _, id := range dashboardIndicatorIDs(dashboard) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func dashboardIndicatorIDs(dashboard map[string]any) []string {
	var ids []string
	for _, tile := range asObjectSlice(dashboard["layoutTiles"]) {
		id := firstNonEmpty(stringAt(tile, "indicatorid"), stringAt(tile, "indicatorId"))
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func dashboardSearchText(dashboard map[string]any) string {
	parts := []string{stringAt(dashboard, "id"), stringAt(dashboard, "name"), stringAt(dashboard, "nameEn"), stringAt(dashboard, "description"), stringAt(objectAt(dashboard, "category"), "name")}
	parts = append(parts, stringSliceAt(dashboard, "tags")...)
	parts = append(parts, dashboardIndicatorIDs(dashboard)...)
	return strings.Join(parts, " ")
}

func indicatorSearchText(indicator map[string]any) string {
	config, _ := parseTileConfig(indicator)
	parts := []string{stringAt(indicator, "id"), stringAt(indicator, "title"), stringAt(config, "title"), stringAt(config, "category"), stringAt(config, "source"), stringAt(config, "dataVersionDate"), stringAt(config, "dateUpload")}
	parts = append(parts, stringSliceAt(config, "tags")...)
	for _, source := range sourceEntries(config) {
		parts = append(parts, stringAt(source, "title"), stringAt(source, "url"))
	}
	for _, snippet := range textSnippets(config, "", 8) {
		parts = append(parts, stringAt(snippet, "text"))
	}
	return strings.Join(parts, " ")
}

func nextActionsForIndicators(items []map[string]any) []string {
	var actions []string
	for _, item := range items[:minInt(3, len(items))] {
		id := stringAt(item, "id")
		actions = append(actions, fmt.Sprintf("dashboardctl indicator get --id %s", id))
		actions = append(actions, fmt.Sprintf("dashboardctl indicator data --id %s --limit 10", id))
	}
	if len(actions) == 0 {
		return []string{`dashboardctl indicator search --term "Arbeitsmarkt" --limit 5`}
	}
	return actions
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

func requiredID(parsed parsedArgs) (string, error) {
	id := firstNonEmpty(parsed.flags["id"], parsed.flags["ids"], firstPosition(parsed))
	if id == "" {
		return "", cliError{2, "missing_id", "command requires --id"}
	}
	return id, nil
}

func envelope(command, requestURL string, request any) map[string]any {
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

func fail(exitCode int, code, message string) {
	emit(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error":       map[string]any{"code": code, "message": message},
	})
	os.Exit(exitCode)
}

func withParams(base string, params url.Values) string {
	if len(params) == 0 {
		return base
	}
	return base + "?" + params.Encode()
}

func asObjectSlice(value any) []map[string]any {
	var out []map[string]any
	for _, item := range asArray(value) {
		if obj, ok := item.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}

func asArray(value any) []any {
	if arr, ok := value.([]any); ok {
		return arr
	}
	return nil
}

func objectAt(obj map[string]any, key string) map[string]any {
	if child, ok := obj[key].(map[string]any); ok {
		return child
	}
	return map[string]any{}
}

func stringAt(obj map[string]any, key string) string {
	if value, ok := obj[key]; ok && value != nil {
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		case json.Number:
			return typed.String()
		default:
			return strings.TrimSpace(fmt.Sprint(typed))
		}
	}
	return ""
}

func stringSliceAt(obj map[string]any, key string) []string {
	var out []string
	for _, item := range asArray(obj[key]) {
		if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func compactCategory(category map[string]any) map[string]any {
	return map[string]any{
		"id":          stringAt(category, "id"),
		"name":        stringAt(category, "name"),
		"nameEn":      stringAt(category, "nameEn"),
		"description": truncate(stripHTML(stringAt(category, "description")), 300),
	}
}

func millisSummary(value any) map[string]any {
	ms := int64FromAny(value)
	if ms <= 0 {
		return map[string]any{}
	}
	return map[string]any{"epochMs": ms, "iso": time.UnixMilli(ms).UTC().Format(time.RFC3339)}
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case json.Number:
		n, _ := typed.Int64()
		return n
	case float64:
		return int64(typed)
	case string:
		n, _ := strconv.ParseInt(typed, 10, 64)
		return n
	default:
		return 0
	}
}

func limitFlag(parsed parsedArgs, fallback, maxValue int) int {
	return limitFlagName(parsed, "limit", fallback, maxValue)
}

func limitFlagName(parsed parsedArgs, name string, fallback, maxValue int) int {
	value := fallback
	if raw := firstNonEmpty(parsed.flags[name]); raw != "" {
		parsedValue, err := strconv.Atoi(raw)
		if err == nil && parsedValue > 0 {
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

func isHelp(value string) bool {
	return value == "--help" || value == "-h" || value == "help"
}

func match(args []string, expected ...string) bool {
	if len(args) < len(expected) {
		return false
	}
	for i, value := range expected {
		if args[i] != value {
			return false
		}
	}
	return true
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var spacePattern = regexp.MustCompile(`\s+`)

func stripHTML(value string) string {
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = htmlTagPattern.ReplaceAllString(value, " ")
	return stripSpace(value)
}

func stripSpace(value string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func truncate(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
