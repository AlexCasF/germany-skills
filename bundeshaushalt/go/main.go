package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName            = "bundeshaushalt"
	baseURL            = "https://bundeshaushalt.de"
	wwwBaseURL         = "https://www.bundeshaushalt.de"
	budgetDataURL      = "https://bundeshaushalt.de/internalapi/budgetData"
	digitalURL         = "https://www.bundeshaushalt.de/DE/Bundeshaushalt-digital/bundeshaushalt-digital.html"
	downloadPortalURL  = "https://www.bundeshaushalt.de/DE/Download-Portal/download-portal.html"
	userNotesURL       = "https://www.bundeshaushalt.de/DE/Service/Benutzerhinweise/benutzerhinweise.html"
	imprintURL         = "https://www.bundeshaushalt.de/DE/Service/Impressum/impressum.html"
	privacyURL         = "https://www.bundeshaushalt.de/DE/Service/Datenschutz/datenschutz.html"
	robotsURL          = "https://www.bundeshaushalt.de/robots.txt"
	bmfBudgetURL       = "https://www.bundesfinanzministerium.de/Web/DE/Themen/Oeffentliche_Finanzen/Bundeshaushalt/bundeshaushalt.html"
	bmfDataUseURL      = "https://www.bundesfinanzministerium.de/Datenportal/Nutzungshinweise/nutzungshinweise.html"
	openAPIWrapperURL  = "https://github.com/bundesAPI/bundeshaushalt-api"
	defaultLimit       = 10
	safeLimit          = 100
	defaultTimeout     = 45 * time.Second
	defaultUserAgent   = "germany-skills/bundeshaushalt"
	latestTargetYear   = 2026
	latestActualYear   = 2024
	earliestKnownYear  = 2012
	defaultSearchDepth = 3
)

var knownYears = []int{2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025, 2026}

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

type budgetResponse struct {
	Meta     budgetMeta                  `json:"meta"`
	Detail   budgetElement               `json:"detail"`
	Children []budgetElement             `json:"children"`
	Parents  [][]labeledElement          `json:"parents"`
	Related  map[string][]labeledElement `json:"related"`
}

type budgetMeta struct {
	Year       int    `json:"year"`
	Unit       string `json:"unit"`
	Quota      string `json:"quota"`
	Account    string `json:"account"`
	Timestamp  int64  `json:"timestamp"`
	ModifyDate string `json:"modifyDate"`
	Entity     string `json:"entity"`
	LevelCur   int    `json:"levelCur"`
	LevelMax   int    `json:"levelMax"`
}

type budgetElement struct {
	ID                    string  `json:"id"`
	BudgetNumber          string  `json:"budgetNumber"`
	Label                 string  `json:"label"`
	Value                 float64 `json:"value"`
	RelativeToParentValue float64 `json:"relativeToParentValue"`
	RelativeValue         float64 `json:"relativeValue"`
	TableLabel            string  `json:"tableLabel"`
	SelectionLabel        string  `json:"selectionLabel"`
}

type labeledElement struct {
	ID    *string `json:"id"`
	Label string  `json:"label"`
}

type searchNode struct {
	id    string
	depth int
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
	case args[0] == "fields":
		err = runFields(args[1:])
	case args[0] == "source":
		err = runSource(args[1:])
	case match(args, "years", "list"):
		err = runYearsList(args[2:])
	case match(args, "budget", "tree"):
		err = runBudgetTree(args[2:])
	case match(args, "budget", "sample"):
		err = runSample(args[2:])
	case args[0] == "sample":
		err = runSample(args[1:])
	case args[0] == "search":
		err = runSearch(args[1:])
	case match(args, "title", "get"):
		err = runTitleGet(args[2:])
	case args[0] == "compare":
		err = runCompare(args[1:])
	case args[0] == "budget-data":
		err = runBudgetData(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run bundeshaushalt --help"}
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
	fmt.Println(`bundeshaushalt -- Bundeshaushalt Digital research CLI

Purpose
  Query and normalize Bundeshaushalt Digital budget hierarchy data from the
  public internal API used by bundeshaushalt.de.

Use this when
  - you need German federal budget revenue or expenditure hierarchy data
  - you need Soll/Ist values by year, Einzelplan, Funktion, or Gruppe
  - you need to drill from top-level budget areas to chapters and titles
  - you need budget values for evidence tables or chart-ready comparisons

Do not use this when
  - you need macroeconomic or labor-market statistics; use Destatis or another
    statistical source
  - you need full legal budget documents; use BMF download pages and cite them
  - you need Bundestag/Bundesrat proceedings; use DIP or the parliament CLIs

Fast paths
  bundeshaushalt doctor
  bundeshaushalt years list
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt search --year 2025 --account expenses --term "BÃ¼rgergeld" --limit 5
  bundeshaushalt title get --year 2025 --account expenses --id 110168112
  bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112
  bundeshaushalt budget-data --year 2025 --account expenses --quota target --unit single --raw

Endpoint-compatible command
  budget-data

Research commands
  doctor
  examples
  years list
  fields
  budget tree
  budget sample
  search
  title get
  compare
  source

Output guarantees
  Commands emit JSON envelopes with status, request, summary/items, sources,
  warnings, and nextActions. Pass --raw on budget-data/tree/title commands for
  the original upstream JSON.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "budget tree":
		fmt.Println(`bundeshaushalt budget tree

Fetch one budget hierarchy node and return compact children.

Examples
  bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8
  bundeshaushalt budget tree --year 2025 --account expenses --id 11 --limit 10

Flags
  --year <year>       Required budget year
  --account <value>   expenses|income, defaults to expenses
  --quota <value>     target|actual, defaults to target
  --unit <value>      single|function|group, defaults to single
  --id <id>           Optional hierarchy node ID
  --limit <n>         Child result cap, defaults to 10
  --grep <term>       Filter/snippet labels
  --raw               Print upstream JSON`)
	case "search":
		fmt.Println(`bundeshaushalt search

Search budget hierarchy labels by traversing from the selected root.

Examples
  bundeshaushalt search --year 2025 --account expenses --term "BÃ¼rgergeld" --limit 5
  bundeshaushalt search --year 2025 --account income --unit group --term "Steuern"

Flags
  --year <year>          Required budget year
  --term <text>          Required search term
  --account <value>      expenses|income, defaults to expenses
  --quota <value>        target|actual, defaults to target
  --unit <value>         single|function|group, defaults to single
  --depth <n>            Traversal depth, defaults to 3
  --max-requests <n>     Traversal request cap, defaults to 60
  --limit <n>            Result cap, defaults to 10`)
	case "title get":
		fmt.Println(`bundeshaushalt title get

Fetch one precise budget node by ID.

Examples
  bundeshaushalt title get --year 2025 --account expenses --id 110168112
  bundeshaushalt title get --year 2025 --account expenses --id 1101 --limit 8`)
	case "compare":
		fmt.Println(`bundeshaushalt compare

Compare one node across years.

Examples
  bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112
  bundeshaushalt compare --years 2024,2025,2026 --account expenses`)
	default:
		printRootHelp()
	}
}

func printExamples() {
	fmt.Println(`bundeshaushalt examples

1. Check health, auth, and fair-use hints:
   bundeshaushalt doctor

2. Inspect available parameter values:
   bundeshaushalt fields

3. List known budget years:
   bundeshaushalt years list

4. Fetch top-level 2026 planned expenses by Einzelplan:
   bundeshaushalt budget tree --year 2026 --account expenses --quota target --unit single --limit 8

5. Drill into the 2025 labour/social ministry:
   bundeshaushalt budget tree --year 2025 --account expenses --id 11 --limit 10

6. Search for BÃ¼rgergeld in the 2025 expense hierarchy:
   bundeshaushalt search --year 2025 --account expenses --term "BÃ¼rgergeld" --limit 5

7. Fetch the BÃ¼rgergeld title:
   bundeshaushalt title get --year 2025 --account expenses --id 110168112

8. Compare that title across two years:
   bundeshaushalt compare --years 2024,2025 --account expenses --id 110168112

9. Query by function instead of Einzelplan:
   bundeshaushalt budget tree --year 2025 --account expenses --unit function --limit 8

10. Use raw endpoint JSON if needed:
   bundeshaushalt budget-data --year 2025 --account expenses --quota target --unit single --raw`)
}

func runDoctor(argv []string) error {
	targetParams := url.Values{"year": {"2026"}, "account": {"expenses"}, "quota": {"target"}, "unit": {"single"}}
	actualParams := url.Values{"year": {"2024"}, "account": {"expenses"}, "quota": {"actual"}, "unit": {"single"}}
	checks := []map[string]any{}
	status := "ok"
	for _, check := range []struct {
		name   string
		params url.Values
	}{
		{"latestTargetExpenses", targetParams},
		{"latestActualExpenses", actualParams},
	} {
		raw, resp, requestURL, err := fetchBudget(check.params)
		item := map[string]any{"name": check.name, "url": requestURL}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
			status = "degraded"
		} else {
			item["ok"] = true
			item["meta"] = resp.Meta
			item["bodyBytes"] = len(raw)
		}
		checks = append(checks, item)
	}
	payload := envelope("doctor", budgetDataURL, map[string]any{})
	payload["status"] = status
	payload["summary"] = map[string]any{
		"authRequired":       false,
		"publishedRateLimit": "No exact public request quota was found for the Bundeshaushalt Digital internal API. robots.txt publishes Crawl-delay: 30 for crawling-style workflows; use small request caps and cache repeated hierarchy traversals.",
		"endpointBehavior":   "GET /internalapi/budgetData requires at least year and account; missing required params produce 400. Some actual values 404 until accounting data exists.",
		"checks":             checks,
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		"bundeshaushalt years list",
		"bundeshaushalt budget tree --year 2026 --account expenses --quota target --limit 8",
		`bundeshaushalt search --year 2025 --account expenses --term "BÃ¼rgergeld" --limit 5`,
	}
	emit(payload)
	return nil
}

func runFields(argv []string) error {
	payload := envelope("fields", budgetDataURL, map[string]any{})
	payload["summary"] = map[string]any{
		"accounts": []map[string]string{
			{"value": "expenses", "label": "Ausgaben"},
			{"value": "income", "label": "Einnahmen"},
		},
		"quotas": []map[string]string{
			{"value": "target", "label": "Sollwerte / planned budget"},
			{"value": "actual", "label": "Istwerte / actual accounting values"},
		},
		"units": []map[string]string{
			{"value": "single", "label": "Einzelplan: ministries and top federal bodies"},
			{"value": "function", "label": "Funktion: policy/task area"},
			{"value": "group", "label": "Gruppe: economic revenue/expenditure type"},
		},
		"knownYears": knownYears,
	}
	payload["items"] = []any{}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundeshaushalt budget tree --year 2026 --account expenses --limit 8"}
	emit(payload)
	return nil
}

func runSource(argv []string) error {
	payload := envelope("source", digitalURL, map[string]any{})
	payload["summary"] = map[string]any{
		"primaryDataSource": "Bundeshaushalt Digital, Bundesministerium der Finanzen",
		"apiEndpoint":       budgetDataURL,
		"citation":          "Bundesministerium der Finanzen, Bundeshaushalt Digital, " + digitalURL,
		"licenseNote":       "BMF Datenportal open data are under the applicable attribution license where marked as open data; Bundeshaushalt Digital also contains freely usable products and public web content. Preserve BMF attribution and dataset/page URLs.",
	}
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundeshaushalt fields", "bundeshaushalt years list"}
	emit(payload)
	return nil
}

func runYearsList(argv []string) error {
	payload := envelope("years list", budgetDataURL, map[string]any{})
	var items []map[string]any
	for _, year := range knownYears {
		items = append(items, map[string]any{
			"year":             year,
			"targetLikely":     year <= latestTargetYear,
			"actualLikely":     year <= latestActualYear,
			"exampleTargetCmd": fmt.Sprintf("bundeshaushalt budget tree --year %d --account expenses --quota target --limit 8", year),
		})
	}
	payload["summary"] = map[string]any{
		"earliestKnownYear": earliestKnownYear,
		"latestTargetYear":  latestTargetYear,
		"latestActualYear":  latestActualYear,
		"count":             len(items),
		"note":              "Known from live endpoint probes and Bundeshaushalt Digital current behavior; the bundled OpenAPI enum stops at 2021.",
	}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundeshaushalt budget tree --year 2026 --account expenses --quota target --limit 8"}
	emit(payload)
	return nil
}

func runBudgetData(argv []string) error {
	parsed := parseArgs(argv)
	params, err := budgetParams(parsed, true, true)
	if err != nil {
		return err
	}
	raw, resp, requestURL, err := fetchBudget(params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	emitBudgetEnvelope("budget-data", requestURL, params, raw, resp, parsed)
	return nil
}

func runBudgetTree(argv []string) error {
	parsed := parseArgs(argv)
	params, err := budgetParams(parsed, true, false)
	if err != nil {
		return err
	}
	raw, resp, requestURL, err := fetchBudget(params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	emitBudgetEnvelope("budget tree", requestURL, params, raw, resp, parsed)
	return nil
}

func runSample(argv []string) error {
	parsed := parseArgs(argv)
	if parsed.flags["year"] == "" {
		parsed.flags["year"] = strconv.Itoa(latestTargetYear)
	}
	if parsed.flags["limit"] == "" {
		parsed.flags["limit"] = "5"
	}
	params, err := budgetParams(parsed, true, false)
	if err != nil {
		return err
	}
	raw, resp, requestURL, err := fetchBudget(params)
	if err != nil {
		return err
	}
	emitBudgetEnvelope("budget sample", requestURL, params, raw, resp, parsed)
	return nil
}

func runTitleGet(argv []string) error {
	parsed := parseArgs(argv)
	if firstNonEmpty(parsed.flags["id"], parsed.params.Get("id")) == "" {
		return cliError{2, "missing_id", "title get requires --id"}
	}
	params, err := budgetParams(parsed, true, false)
	if err != nil {
		return err
	}
	raw, resp, requestURL, err := fetchBudget(params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	emitBudgetEnvelope("title get", requestURL, params, raw, resp, parsed)
	return nil
}

func runSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "search requires --term"}
	}
	params, err := budgetParams(parsed, true, false)
	if err != nil {
		return err
	}
	params.Del("id")
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	depth := intFlag(parsed, "depth", defaultSearchDepth, 6)
	maxRequests := intFlag(parsed, "max-requests", 60, 250)
	items, requests, err := searchHierarchy(params, term, depth, maxRequests, limit, flagBool(parsed, "include-raw"))
	if err != nil {
		return err
	}
	payload := envelope("search", budgetDataURL, map[string]any{
		"year": params.Get("year"), "account": params.Get("account"), "quota": params.Get("quota"), "unit": params.Get("unit"), "term": term, "depth": depth, "maxRequests": maxRequests, "limit": limit,
	})
	payload["summary"] = map[string]any{
		"term":           term,
		"returned":       len(items),
		"requestsUsed":   requests,
		"requestCap":     maxRequests,
		"traversalDepth": depth,
	}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsFromSearch(items, params)
	emit(payload)
	return nil
}

func runCompare(argv []string) error {
	parsed := parseArgs(argv)
	rawYears := firstNonEmpty(parsed.flags["years"], parsed.flags["year"], parsed.params.Get("years"))
	if rawYears == "" {
		return cliError{2, "missing_years", "compare requires --years 2024,2025"}
	}
	years, err := parseYears(rawYears)
	if err != nil {
		return err
	}
	baseParams, err := budgetParams(parsed, false, false)
	if err != nil {
		return err
	}
	var items []map[string]any
	status := "ok"
	for _, year := range years {
		params := cloneValues(baseParams)
		params.Set("year", strconv.Itoa(year))
		raw, resp, requestURL, err := fetchBudget(params)
		item := map[string]any{"year": year, "requestUrl": requestURL}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
			status = "partial"
		} else {
			item["ok"] = true
			item["meta"] = resp.Meta
			item["detail"] = compactElement(resp.Detail, resp.Meta, params)
			item["childCount"] = len(resp.Children)
			if flagBool(parsed, "include-raw") {
				item["raw"] = json.RawMessage(raw)
			}
		}
		items = append(items, item)
	}
	payload := envelope("compare", budgetDataURL, map[string]any{"years": years, "account": baseParams.Get("account"), "quota": baseParams.Get("quota"), "unit": baseParams.Get("unit"), "id": baseParams.Get("id")})
	payload["status"] = status
	payload["summary"] = map[string]any{"years": years, "id": baseParams.Get("id"), "account": baseParams.Get("account"), "quota": baseParams.Get("quota"), "unit": baseParams.Get("unit"), "returned": len(items)}
	payload["items"] = items
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundeshaushalt source"}
	emit(payload)
	return nil
}

func emitBudgetEnvelope(command string, requestURL string, params url.Values, raw []byte, resp *budgetResponse, parsed parsedArgs) {
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	grep := firstNonEmpty(parsed.flags["grep"], parsed.flags["term"], parsed.flags["q"])
	children := compactChildren(resp.Children, resp.Meta, params, limit, grep)
	payload := envelope(command, requestURL, map[string]any{
		"year": params.Get("year"), "account": params.Get("account"), "quota": params.Get("quota"), "unit": params.Get("unit"), "id": params.Get("id"), "limit": limit, "grep": grep,
	})
	payload["summary"] = map[string]any{
		"meta":          resp.Meta,
		"detail":        compactElement(resp.Detail, resp.Meta, params),
		"childrenTotal": len(resp.Children),
		"childrenShown": len(children),
		"parentsLevels": len(resp.Parents),
		"relatedKeys":   sortedRelatedKeys(resp.Related),
	}
	payload["items"] = children
	payload["sources"] = sourcesForBudget(requestURL)
	payload["warnings"] = defaultWarningsForResponse(resp)
	payload["nextActions"] = nextActionsForResponse(resp, params)
	if flagBool(parsed, "include-raw") {
		payload["raw"] = json.RawMessage(raw)
	}
	emit(payload)
}

func searchHierarchy(baseParams url.Values, term string, maxDepth int, maxRequests int, limit int, includeRaw bool) ([]map[string]any, int, error) {
	queue := []searchNode{{id: "", depth: 0}}
	seen := map[string]bool{}
	var items []map[string]any
	requests := 0
	needle := strings.ToLower(term)
	for len(queue) > 0 && requests < maxRequests && len(items) < limit {
		node := queue[0]
		queue = queue[1:]
		key := node.id + ":" + strconv.Itoa(node.depth)
		if seen[key] {
			continue
		}
		seen[key] = true
		params := cloneValues(baseParams)
		if node.id != "" {
			params.Set("id", node.id)
		} else {
			params.Del("id")
		}
		raw, resp, _, err := fetchBudget(params)
		requests++
		if err != nil {
			continue
		}
		if matchesElement(resp.Detail, needle) && resp.Detail.Label != "" {
			item := compactElement(resp.Detail, resp.Meta, params)
			item["matchType"] = "detail"
			if includeRaw {
				item["raw"] = json.RawMessage(raw)
			}
			items = append(items, item)
		}
		for _, child := range resp.Children {
			if matchesElement(child, needle) {
				item := compactElement(child, resp.Meta, params)
				item["matchType"] = "child"
				item["parentId"] = resp.Detail.ID
				item["parentLabel"] = resp.Detail.Label
				items = append(items, item)
				if len(items) >= limit {
					break
				}
			}
			if child.ID != "" && node.depth < maxDepth {
				queue = append(queue, searchNode{id: child.ID, depth: node.depth + 1})
			}
		}
	}
	return items, requests, nil
}

func matchesElement(element budgetElement, needle string) bool {
	return strings.Contains(strings.ToLower(strings.Join([]string{element.ID, element.BudgetNumber, element.Label}, " ")), needle)
}

func compactChildren(children []budgetElement, meta budgetMeta, params url.Values, limit int, grep string) []map[string]any {
	var out []map[string]any
	needle := strings.ToLower(grep)
	for _, child := range children {
		if grep != "" && !matchesElement(child, needle) {
			continue
		}
		out = append(out, compactElement(child, meta, params))
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactElement(element budgetElement, meta budgetMeta, params url.Values) map[string]any {
	id := element.ID
	item := map[string]any{
		"id":                    id,
		"budgetNumber":          element.BudgetNumber,
		"label":                 element.Label,
		"value":                 element.Value,
		"valueEur":              element.Value,
		"valueBillionEur":       element.Value / 1_000_000_000,
		"relativeToParentValue": element.RelativeToParentValue,
		"relativeValue":         element.RelativeValue,
		"tableLabel":            element.TableLabel,
		"selectionLabel":        element.SelectionLabel,
		"year":                  meta.Year,
		"account":               meta.Account,
		"quota":                 meta.Quota,
		"unit":                  meta.Unit,
		"entity":                meta.Entity,
		"levelCur":              meta.LevelCur,
		"levelMax":              meta.LevelMax,
		"nextActions":           nextActionsForElement(id, params),
	}
	return item
}

func nextActionsForElement(id string, params url.Values) []string {
	if id == "" {
		return nil
	}
	year := params.Get("year")
	account := params.Get("account")
	quota := params.Get("quota")
	unit := params.Get("unit")
	return []string{
		fmt.Sprintf("bundeshaushalt title get --year %s --account %s --quota %s --unit %s --id %s", year, account, quota, unit, id),
		fmt.Sprintf("bundeshaushalt budget tree --year %s --account %s --quota %s --unit %s --id %s --limit 10", year, account, quota, unit, id),
	}
}

func nextActionsForResponse(resp *budgetResponse, params url.Values) []string {
	actions := []string{"bundeshaushalt source"}
	for _, child := range resp.Children {
		if child.ID != "" {
			actions = append(actions, fmt.Sprintf("bundeshaushalt budget tree --year %s --account %s --quota %s --unit %s --id %s --limit 10", params.Get("year"), params.Get("account"), params.Get("quota"), params.Get("unit"), child.ID))
		}
		if len(actions) >= 4 {
			break
		}
	}
	return actions
}

func nextActionsFromSearch(items []map[string]any, params url.Values) []string {
	var actions []string
	for _, item := range items {
		if id := fmt.Sprint(item["id"]); id != "" {
			actions = append(actions, fmt.Sprintf("bundeshaushalt title get --year %s --account %s --quota %s --unit %s --id %s", params.Get("year"), params.Get("account"), params.Get("quota"), params.Get("unit"), id))
		}
		if len(actions) >= 4 {
			return actions
		}
	}
	return []string{"bundeshaushalt budget tree --year " + params.Get("year") + " --account " + params.Get("account") + " --limit 8"}
}

func budgetParams(parsed parsedArgs, requireYear bool, requireAccount bool) (url.Values, error) {
	params := cloneValues(parsed.params)
	for _, key := range []string{"year", "account", "quota", "unit", "id"} {
		if value := parsed.flags[key]; value != "" {
			params.Set(key, value)
		}
	}
	if params.Get("account") == "" && !requireAccount {
		params.Set("account", "expenses")
	}
	if params.Get("quota") == "" {
		params.Set("quota", "target")
	}
	if params.Get("unit") == "" {
		params.Set("unit", "single")
	}
	if requireYear && params.Get("year") == "" {
		return nil, cliError{2, "missing_year", "command requires --year"}
	}
	if requireAccount && params.Get("account") == "" {
		return nil, cliError{2, "missing_account", "budget-data requires --account expenses|income"}
	}
	if err := validateParams(params); err != nil {
		return nil, err
	}
	return params, nil
}

func validateParams(params url.Values) error {
	if year := params.Get("year"); year != "" {
		parsedYear, err := strconv.Atoi(year)
		if err != nil || parsedYear < 2000 || parsedYear > 2100 {
			return cliError{2, "invalid_year", "year must be a four-digit year"}
		}
	}
	if account := params.Get("account"); account != "" && account != "expenses" && account != "income" {
		return cliError{2, "invalid_account", "account must be expenses or income"}
	}
	if quota := params.Get("quota"); quota != "" && quota != "target" && quota != "actual" {
		return cliError{2, "invalid_quota", "quota must be target or actual"}
	}
	if unit := params.Get("unit"); unit != "" && unit != "single" && unit != "function" && unit != "group" {
		return cliError{2, "invalid_unit", "unit must be single, function, or group"}
	}
	return nil
}

func fetchBudget(params url.Values) ([]byte, *budgetResponse, string, error) {
	requestURL := withParams(budgetDataURL, params)
	status, _, raw, err := fetchRaw(requestURL)
	if err != nil {
		return nil, nil, requestURL, err
	}
	if status < 200 || status >= 300 {
		return nil, nil, requestURL, httpError{status, string(raw), requestURL}
	}
	var resp budgetResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, requestURL, err
	}
	return raw, &resp, requestURL, nil
}

func fetchRaw(requestURL string) (int, string, []byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	var lastStatus int
	var lastContentType string
	var lastBody []byte
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 750 * time.Millisecond)
		}
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return 0, "", nil, err
		}
		req.Header.Set("User-Agent", defaultUserAgent)
		req.Header.Set("Accept", "application/json")
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
			return resp.StatusCode, lastContentType, body, readErr
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusGatewayTimeout {
			return lastStatus, lastContentType, lastBody, nil
		}
	}
	return lastStatus, lastContentType, lastBody, lastErr
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

func parseYears(raw string) ([]int, error) {
	var years []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		year, err := strconv.Atoi(part)
		if err != nil {
			return nil, cliError{2, "invalid_years", "years must be comma-separated integers"}
		}
		years = append(years, year)
	}
	if len(years) == 0 {
		return nil, cliError{2, "invalid_years", "years must contain at least one year"}
	}
	return years, nil
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

func limitFlag(parsed parsedArgs, fallback int, maxValue int) int {
	return intFlag(parsed, "limit", fallback, maxValue)
}

func intFlag(parsed parsedArgs, name string, fallback int, maxValue int) int {
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

func cloneValues(values url.Values) url.Values {
	out := url.Values{}
	for key, vals := range values {
		for _, value := range vals {
			out.Add(key, value)
		}
	}
	return out
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

func defaultSources() []map[string]any {
	return []map[string]any{
		{"title": "Bundeshaushalt Digital", "url": digitalURL, "kind": "official_application"},
		{"title": "Bundeshaushalt internal API endpoint", "url": budgetDataURL, "kind": "api_endpoint"},
		{"title": "BMF Bundeshaushalt overview", "url": bmfBudgetURL, "kind": "official_context"},
		{"title": "BMF Datenportal usage notes", "url": bmfDataUseURL, "kind": "terms"},
		{"title": "Bundeshaushalt user notes", "url": userNotesURL, "kind": "terms"},
		{"title": "Bundeshaushalt robots.txt", "url": robotsURL, "kind": "fair_use"},
		{"title": "OpenAPI wrapper", "url": openAPIWrapperURL, "kind": "openapi_reference"},
	}
}

func sourcesForBudget(requestURL string) []map[string]any {
	out := defaultSources()
	out = append([]map[string]any{{"title": "Bundeshaushalt API request", "url": requestURL, "kind": "api_request"}}, out...)
	return out
}

func defaultWarnings() []string {
	return []string{
		"No exact public rate limit for the Bundeshaushalt Digital API was found; robots.txt publishes Crawl-delay: 30 for crawling-like workflows.",
		"Actual/Ist values are only available after accounting data exists; newer years can return 404 for quota=actual.",
		"The bundled OpenAPI enum stops at 2021; live endpoint checks show newer target years are available.",
		"Budget values are nominal euro amounts; use statistical APIs for inflation, population, or macroeconomic context.",
		"Use BMF attribution and preserve dataset/page URLs in final citations.",
	}
}

func defaultWarningsForResponse(resp *budgetResponse) []string {
	warnings := defaultWarnings()
	if resp.Meta.Quota == "actual" && resp.Meta.Year > latestActualYear {
		warnings = append(warnings, "This actual/Ist year is newer than the latest actual year observed during testing; verify availability carefully.")
	}
	if resp.Meta.Unit == "function" || resp.Meta.Unit == "group" {
		warnings = append(warnings, "Function and group views classify titles differently from Einzelplan ministry structure; do not mix categories without saying so.")
	}
	return warnings
}

func sortedRelatedKeys(related map[string][]labeledElement) []string {
	var keys []string
	for key := range related {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

func stripSpace(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func truncate(value string, maxLen int) string {
	value = stripSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
