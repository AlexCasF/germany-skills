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
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName = "dip-bundestag"
	baseURL = "https://search.dip.bundestag.de/api/v1"
)

var entitySummaries = map[string]string{
	"vorgang":              "Proceedings and legislative process metadata",
	"vorgangsposition":     "Proceeding positions and parliamentary process steps",
	"drucksache":           "Printed paper metadata",
	"drucksache-text":      "Printed paper metadata plus full text where available",
	"plenarprotokoll":      "Plenary protocol metadata",
	"plenarprotokoll-text": "Plenary protocol metadata plus full text where available",
	"person":               "Person master data",
	"aktivitaet":           "Parliamentary activities",
}

var rawEntities = map[string]bool{
	"vorgang":         true,
	"drucksache":      true,
	"plenarprotokoll": true,
	"person":          true,
	"aktivitaet":      true,
}

type parsedArgs struct {
	flags  map[string]string
	params url.Values
}

type apiResponse struct {
	statusCode  int
	contentType string
	body        []byte
	requestURL  string
}

type envelope map[string]any

func main() {
	args := os.Args[1:]
	if len(args) == 0 || isHelp(args[0]) {
		printRootHelp()
		return
	}

	if isHelp(args[len(args)-1]) {
		printHelpFor(args[:len(args)-1])
		return
	}

	switch {
	case args[0] == "doctor":
		runDoctor(args[1:])
	case len(args) >= 2 && args[0] == "person" && args[1] == "search":
		runPersonSearch(args[2:])
	case len(args) >= 2 && args[0] == "person" && args[1] == "dossier":
		runPersonDossier(args[2:])
	case len(args) >= 2 && args[0] == "vorgang" && args[1] == "dossier":
		runVorgangDossier(args[2:])
	case len(args) >= 2 && args[0] == "source":
		runSource(args[1:])
	case len(args) >= 2 && (args[0] == "plenarprotokoll" || args[0] == "drucksache") && args[1] == "text":
		runDocumentText(args[0], args[2:])
	case len(args) >= 3 && args[0] == "plenary" && args[1] == "speech" && args[2] == "search":
		runPlenarySpeechSearch(args[3:])
	case len(args) >= 2 && isEntity(args[0]) && args[1] == "list":
		runRawList(args[0], args[2:])
	case len(args) >= 2 && isEntity(args[0]) && args[1] == "get":
		runRawGet(args[0], args[2:])
	default:
		fail(2, "unknown_command", fmt.Sprintf("unknown command path: %s", strings.Join(args, " ")))
	}
}

func printRootHelp() {
	fmt.Printf(`dip-bundestag -- official Bundestag DIP research CLI

Purpose
  Search and cite official parliamentary material from the Bundestag DIP API.

Use this when
  - you need official proceedings, printed papers, protocols, people, or activities
  - you need to distinguish official plenary records from media or campaign quotes
  - you need API-backed source URLs and citation metadata

Do not use this when
  - you need general news context
  - you need lobbying register financial data
  - you need live Bundestag presentation feeds

Fast paths
  Check auth and endpoint health:
    dip-bundestag doctor

  Find a person:
    dip-bundestag person search --name "Mustername" --limit 3

  Build an evidence bundle:
    dip-bundestag person dossier --name "Mustername"

  Get official plenary text snippets:
    dip-bundestag plenarprotokoll text --document-number "20/139" --grep "Suchbegriff"

Raw endpoint commands
  dip-bundestag vorgang list|get
  dip-bundestag drucksache list|get
  dip-bundestag plenarprotokoll list|get
  dip-bundestag person list|get
  dip-bundestag aktivitaet list|get

Additional endpoint commands
  dip-bundestag vorgangsposition list|get
  dip-bundestag drucksache-text list|get
  dip-bundestag plenarprotokoll-text list|get

Research commands
  doctor                         Auth, endpoint, docs, and fair-use health check
  person search                  Compact person discovery
  person dossier                 Person record plus related activities and sources
  vorgang dossier                Proceeding plus related positions and sources
  source                         Source URL and citation metadata for one record
  plenarprotokoll text           Full text or grep snippets for a plenary protocol
  drucksache text                Full text or grep snippets for a printed paper
  plenary speech search          Bounded helper for official plenary text/activity search

Auth
  Prefer DIP_API_KEY from the environment.
  --apikey still works for compatibility and is redacted from normalized output.

Output
  Raw endpoint commands return upstream JSON by default.
  Research commands return a normalized JSON envelope with status, request,
  retrievedAt, sources, warnings, and nextActions.
`)
}

func printHelpFor(path []string) {
	switch {
	case len(path) == 1 && path[0] == "doctor":
		fmt.Println(`dip-bundestag doctor

What it does
  Checks whether DIP_API_KEY or --apikey is configured and performs one small
  authenticated request when possible.

Examples
  dip-bundestag doctor
  dip-bundestag doctor --apikey <key>

Safety
  The key is never printed.`)
	case len(path) >= 2 && path[0] == "person" && path[1] == "search":
		fmt.Println(`dip-bundestag person search

What it does
  Searches official DIP person master data and returns compact result rows.

Inputs
  --name    Required person name or name fragment
  --limit   Optional client-side result limit, default 10

Examples
  dip-bundestag person search --name "Mustername"
  dip-bundestag person search --name "Mustername" --limit 3`)
	case len(path) >= 2 && path[0] == "person" && path[1] == "dossier":
		fmt.Println(`dip-bundestag person dossier

What it does
  Builds a compact official-source evidence bundle for one person.

Inputs
  --id      Stable DIP person ID
  --name    Name to search when ID is not known
  --limit   Related activity limit, default 10

Examples
  dip-bundestag person dossier --id 760
  dip-bundestag person dossier --name "Mustername"`)
	case len(path) >= 2 && path[0] == "vorgang" && path[1] == "dossier":
		fmt.Println(`dip-bundestag vorgang dossier

What it does
  Fetches one proceeding and related proceeding positions.

Inputs
  --id      Required DIP proceeding ID
  --limit   Related position limit, default 10

Examples
  dip-bundestag vorgang dossier --id 123456`)
	case len(path) >= 1 && path[0] == "source":
		fmt.Println(`dip-bundestag source

What it does
  Prints API, PDF, XML, and citation metadata for one DIP record.

Inputs
  --type              Entity type, e.g. plenarprotokoll, drucksache, vorgang
  --id                Stable DIP ID
  --document-number   Document number, e.g. 20/139

Examples
  dip-bundestag source --type plenarprotokoll --document-number "20/139"
  dip-bundestag source --type drucksache --id 264030`)
	case len(path) >= 2 && (path[0] == "plenarprotokoll" || path[0] == "drucksache") && path[1] == "text":
		fmt.Printf(`dip-bundestag %s text

What it does
  Fetches official full text where available and optionally returns grep snippets.

Inputs
  --id                Stable DIP text record ID
  --document-number   Document number, e.g. 20/139
  --grep              Optional case-insensitive snippet search
  --context           Optional snippet context chars, default 220

Examples
  dip-bundestag %s text --document-number "20/139" --grep "Suchbegriff"
`, path[0], path[0])
	case len(path) >= 3 && path[0] == "plenary" && path[1] == "speech" && path[2] == "search":
		fmt.Println(`dip-bundestag plenary speech search

What it does
  Bounded helper for official plenary research. With --document-number it greps
  official plenary full text. With --person-id or --person it searches official
  activity records.

Inputs
  --document-number   Optional plenary protocol number
  --person-id         Optional DIP person ID for activity search
  --person            Optional person name for activity search
  --term              Required search term
  --limit             Optional activity result limit, default 10

Examples
  dip-bundestag plenary speech search --document-number "20/139" --term "Suchbegriff"
  dip-bundestag plenary speech search --person "Mustername" --term "Indikator"`)
	case len(path) >= 2 && isEntity(path[0]) && (path[1] == "list" || path[1] == "get"):
		fmt.Printf(`dip-bundestag %s %s

Raw endpoint command.

Auth
  Uses DIP_API_KEY unless --apikey is passed.

Flags
  --apikey       Optional explicit API key
  --id           Required for get
  --param k=v    Repeatable upstream query parameter
  --limit n      Optional client-side document limit for JSON list responses

Examples
  dip-bundestag %s list --param "f.person=Mustername"
  dip-bundestag %s get --id 760
`, path[0], path[1], path[0], path[0])
	default:
		printRootHelp()
	}
}

func runDoctor(args []string) {
	pa := mustParse(args)
	key, source := resolveAPIKey(pa.flags)
	result := envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "doctor",
		"retrievedAt": now(),
		"summary": map[string]any{
			"baseUrl":               baseURL,
			"authRequired":          true,
			"apiKeyConfigured":      key != "",
			"apiKeySource":          source,
			"maxConcurrentRequests": 25,
			"normalListMaxItems":    100,
			"fullTextListMaxItems":  "usually 10",
		},
		"sources": []map[string]string{
			{"title": "DIP API help", "url": "https://dip.bundestag.de/%C3%BCber-dip/hilfe/api", "kind": "documentation"},
			{"title": "DIP short documentation PDF", "url": "https://dip.bundestag.de/documents/informationsblatt_zur_dip_api.pdf", "kind": "documentation"},
			{"title": "DIP terms PDF", "url": "https://dip.bundestag.de/documents/nutzungsbedingungen_dip.pdf", "kind": "terms"},
			{"title": "DIP OpenAPI YAML", "url": "https://search.dip.bundestag.de/api/v1/openapi.yaml", "kind": "openapi"},
		},
		"warnings": []string{
			"Do not exceed 25 concurrent API requests.",
			"Detailed rate-limit internals beyond official notes are not published.",
			"Use source attribution: Deutscher Bundestag/Bundesrat - DIP.",
		},
		"nextActions": []string{
			"dip-bundestag person search --name \"Mustername\"",
			"dip-bundestag plenarprotokoll text --document-number \"20/139\" --grep \"Suchbegriff\"",
		},
	}
	if key == "" {
		result["status"] = "error"
		result["error"] = map[string]string{
			"code":    "missing_api_key",
			"message": "Set DIP_API_KEY or pass --apikey.",
		}
		writeJSON(result)
		os.Exit(2)
	}

	params := url.Values{}
	params.Set("f.person", "Steinmeier")
	params.Set("format", "json")
	resp, err := apiGet("person", params, key)
	if err != nil {
		result["status"] = "error"
		result["error"] = map[string]string{"code": "request_failed", "message": err.Error()}
		writeJSON(result)
		os.Exit(1)
	}
	result["request"] = requestMeta(resp.requestURL)
	result["summary"].(map[string]any)["healthStatusCode"] = resp.statusCode
	writeJSON(result)
}

func runRawList(entity string, args []string) {
	pa := mustParse(args)
	key := mustAPIKey(pa.flags)
	params := cloneValues(pa.params)
	params.Set("format", defaultString(params.Get("format"), "json"))
	resp, err := apiGet(entity, params, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	body := resp.body
	if limitRaw := pa.flags["limit"]; limitRaw != "" && strings.Contains(resp.contentType, "json") {
		limit, err := parsePositiveInt(limitRaw, "limit")
		if err != nil {
			fail(2, "invalid_arguments", err.Error())
		}
		body, err = limitJSONDocuments(resp.body, limit)
		if err != nil {
			fail(1, "output_failed", err.Error())
		}
	}
	writeBody(body)
}

func runRawGet(entity string, args []string) {
	pa := mustParse(args)
	id := pa.flags["id"]
	if id == "" {
		fail(2, "invalid_arguments", "missing required flag --id")
	}
	key := mustAPIKey(pa.flags)
	params := cloneValues(pa.params)
	params.Set("format", defaultString(params.Get("format"), "json"))
	resp, err := apiGet(entity+"/"+url.PathEscape(id), params, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	writeBody(resp.body)
}

func runPersonSearch(args []string) {
	pa := mustParse(args)
	name := pa.flags["name"]
	if name == "" {
		fail(2, "invalid_arguments", "missing required flag --name")
	}
	limit := optionalLimit(pa.flags["limit"], 10)
	key := mustAPIKey(pa.flags)
	params := url.Values{}
	params.Set("f.person", name)
	params.Set("format", "json")
	resp, err := apiGet("person", params, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	data := mustDecodeJSON(resp.body)
	docs := takeDocuments(data, limit)
	items := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		items = append(items, compactItem(doc))
	}
	writeJSON(envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "person search",
		"retrievedAt": now(),
		"request":     requestMeta(resp.requestURL),
		"summary": map[string]any{
			"query":       name,
			"numFound":    getNumber(data, "numFound"),
			"returned":    len(items),
			"clientLimit": limit,
		},
		"items":    items,
		"sources":  []map[string]string{{"title": "DIP API person endpoint", "url": baseURL + "/person", "kind": "api"}},
		"warnings": []string{},
		"nextActions": []string{
			"dip-bundestag person dossier --id <id>",
			"dip-bundestag aktivitaet list --param \"f.person_id=<id>\"",
		},
	})
}

func runPersonDossier(args []string) {
	pa := mustParse(args)
	limit := optionalLimit(pa.flags["limit"], 10)
	key := mustAPIKey(pa.flags)
	id := pa.flags["id"]
	var searchSummary any
	if id == "" {
		name := pa.flags["name"]
		if name == "" {
			fail(2, "invalid_arguments", "pass --id or --name")
		}
		params := url.Values{}
		params.Set("f.person", name)
		params.Set("format", "json")
		resp, err := apiGet("person", params, key)
		if err != nil {
			fail(1, "request_failed", err.Error())
		}
		data := mustDecodeJSON(resp.body)
		docs := takeDocuments(data, 1)
		if len(docs) == 0 {
			fail(1, "not_found", "no person found for --name")
		}
		id = firstString(docs[0], "id")
		searchSummary = map[string]any{"request": requestMeta(resp.requestURL), "selected": compactItem(docs[0])}
	}

	personResp, err := apiGet("person/"+url.PathEscape(id), url.Values{"format": []string{"json"}}, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	person := mustDecodeJSON(personResp.body)

	actParams := url.Values{}
	actParams.Set("f.person_id", id)
	actParams.Set("format", "json")
	actResp, err := apiGet("aktivitaet", actParams, key)
	activities := []map[string]any{}
	activityWarning := ""
	if err == nil {
		actData := mustDecodeJSON(actResp.body)
		for _, doc := range takeDocuments(actData, limit) {
			activities = append(activities, compactItem(doc))
		}
	} else {
		activityWarning = err.Error()
	}

	sources := extractSources(person)
	sources = append(sources, map[string]string{"title": "DIP API person detail", "url": baseURL + "/person/" + url.PathEscape(id), "kind": "api"})

	warnings := []string{
		"Dossier uses official DIP person and activity records only.",
		"Outside quotes, campaign statements, and news context are not included.",
	}
	if activityWarning != "" {
		warnings = append(warnings, "Related activities could not be loaded: "+activityWarning)
	}

	writeJSON(envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "person dossier",
		"retrievedAt": now(),
		"request": map[string]any{
			"person": requestMeta(personResp.requestURL),
		},
		"summary": map[string]any{
			"person":                 compactItem(person),
			"relatedActivitiesShown": len(activities),
			"search":                 searchSummary,
		},
		"record": person,
		"related": map[string]any{
			"activities": activities,
		},
		"sources":  dedupeSources(sources),
		"warnings": warnings,
		"nextActions": []string{
			"dip-bundestag aktivitaet list --param \"f.person_id=" + id + "\"",
			"dip-bundestag plenary speech search --person-id " + id + " --term <term>",
		},
	})
}

func runVorgangDossier(args []string) {
	pa := mustParse(args)
	id := pa.flags["id"]
	if id == "" {
		fail(2, "invalid_arguments", "missing required flag --id")
	}
	limit := optionalLimit(pa.flags["limit"], 10)
	key := mustAPIKey(pa.flags)

	vorgangResp, err := apiGet("vorgang/"+url.PathEscape(id), url.Values{"format": []string{"json"}}, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	vorgang := mustDecodeJSON(vorgangResp.body)

	posParams := url.Values{}
	posParams.Set("f.vorgang", id)
	posParams.Set("format", "json")
	posResp, err := apiGet("vorgangsposition", posParams, key)
	positions := []map[string]any{}
	warnings := []string{"Dossier uses official DIP proceeding and proceeding-position records."}
	if err == nil {
		posData := mustDecodeJSON(posResp.body)
		for _, doc := range takeDocuments(posData, limit) {
			positions = append(positions, compactItem(doc))
		}
	} else {
		warnings = append(warnings, "Related positions could not be loaded: "+err.Error())
	}

	sources := extractSources(vorgang)
	sources = append(sources, map[string]string{"title": "DIP API proceeding detail", "url": baseURL + "/vorgang/" + url.PathEscape(id), "kind": "api"})

	writeJSON(envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "vorgang dossier",
		"retrievedAt": now(),
		"request":     requestMeta(vorgangResp.requestURL),
		"summary": map[string]any{
			"vorgang":               compactItem(vorgang),
			"relatedPositionsShown": len(positions),
		},
		"record": vorgang,
		"related": map[string]any{
			"positions": positions,
		},
		"sources":  dedupeSources(sources),
		"warnings": warnings,
		"nextActions": []string{
			"dip-bundestag vorgangsposition list --param \"f.vorgang=" + id + "\"",
			"dip-bundestag source --type vorgang --id " + id,
		},
	})
}

func runSource(args []string) {
	pa := mustParse(args)
	entity := pa.flags["type"]
	if entity == "" {
		entity = pa.flags["entity"]
	}
	if entity == "" {
		fail(2, "invalid_arguments", "missing required flag --type")
	}
	if !isEntity(entity) {
		fail(2, "invalid_arguments", "unknown --type: "+entity)
	}
	key := mustAPIKey(pa.flags)
	record, request, err := resolveRecord(entity, pa.flags["id"], pa.flags["document-number"], key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	sources := extractSources(record)
	if id := firstString(record, "id"); id != "" {
		sources = append(sources, map[string]string{"title": "DIP API " + entity + " detail", "url": baseURL + "/" + entity + "/" + url.PathEscape(id), "kind": "api"})
	}
	writeJSON(envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "source",
		"retrievedAt": now(),
		"request":     request,
		"summary": map[string]any{
			"entity":         entity,
			"record":         compactItem(record),
			"sourceCount":    len(dedupeSources(sources)),
			"citationSource": "Deutscher Bundestag/Bundesrat - DIP",
		},
		"sources": dedupeSources(sources),
		"warnings": []string{
			"Cite DIP as source. For BT plenary protocols use BT-PlPr. plus document number.",
		},
		"nextActions": []string{
			"dip-bundestag " + entity + " get --id <id>",
		},
	})
}

func runDocumentText(kind string, args []string) {
	pa := mustParse(args)
	key := mustAPIKey(pa.flags)
	textEntity := kind + "-text"
	record, request, err := resolveRecord(textEntity, pa.flags["id"], pa.flags["document-number"], key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	text := firstString(record, "text")
	grep := pa.flags["grep"]
	context := optionalLimit(pa.flags["context"], 220)
	snips := []map[string]any{}
	if grep != "" {
		snips = snippets(text, grep, context)
	}
	sources := extractSources(record)
	if id := firstString(record, "id"); id != "" {
		sources = append(sources, map[string]string{"title": "DIP API " + textEntity + " detail", "url": baseURL + "/" + textEntity + "/" + url.PathEscape(id), "kind": "api"})
	}
	summary := map[string]any{
		"record":      compactItem(record),
		"textLength":   len([]rune(text)),
		"grep":         grep,
		"snippetCount": len(snips),
	}
	out := envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     kind + " text",
		"retrievedAt": now(),
		"request":     request,
		"summary":     summary,
		"sources":     dedupeSources(sources),
		"warnings": []string{
			"Full text is official DIP text where available.",
			"Use source attribution: Deutscher Bundestag/Bundesrat - DIP.",
		},
		"nextActions": []string{
			"dip-bundestag source --type " + kind + " --id " + firstString(record, "id"),
		},
	}
	if grep != "" {
		out["snippets"] = snips
	} else {
		out["textPreview"] = preview(text, 1800)
	}
	writeJSON(out)
}

func runPlenarySpeechSearch(args []string) {
	pa := mustParse(args)
	term := pa.flags["term"]
	if term == "" {
		fail(2, "invalid_arguments", "missing required flag --term")
	}
	if pa.flags["document-number"] != "" || pa.flags["id"] != "" {
		runDocumentText("plenarprotokoll", append(args, "--grep", term))
		return
	}
	key := mustAPIKey(pa.flags)
	limit := optionalLimit(pa.flags["limit"], 10)
	params := url.Values{}
	params.Set("format", "json")
	if pa.flags["person-id"] != "" {
		params.Set("f.person_id", pa.flags["person-id"])
	} else if pa.flags["person"] != "" {
		params.Set("f.person", pa.flags["person"])
	} else {
		fail(2, "invalid_arguments", "pass --document-number, --person-id, or --person")
	}
	resp, err := apiGet("aktivitaet", params, key)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	data := mustDecodeJSON(resp.body)
	matches := []map[string]any{}
	for _, doc := range takeDocuments(data, 100) {
		if strings.Contains(strings.ToLower(string(mustMarshal(doc))), strings.ToLower(term)) {
			matches = append(matches, compactItem(doc))
			if len(matches) >= limit {
				break
			}
		}
	}
	writeJSON(envelope{
		"status":      "ok",
		"tool":        appName,
		"command":     "plenary speech search",
		"retrievedAt": now(),
		"request":     requestMeta(resp.requestURL),
		"summary": map[string]any{
			"mode":        "aktivitaet-search",
			"term":        term,
			"returned":    len(matches),
			"clientLimit": limit,
		},
		"items":   matches,
		"sources": []map[string]string{{"title": "DIP API activity endpoint", "url": baseURL + "/aktivitaet", "kind": "api"}},
		"warnings": []string{
			"Activity search is official DIP metadata, not a full transcript search.",
			"For transcript snippets, pass --document-number so the command searches plenarprotokoll-text.",
		},
		"nextActions": []string{
			"dip-bundestag plenarprotokoll text --document-number <number> --grep \"" + term + "\"",
		},
	})
}

func resolveRecord(entity string, id string, documentNumber string, key string) (map[string]any, map[string]any, error) {
	if id != "" {
		resp, err := apiGet(entity+"/"+url.PathEscape(id), url.Values{"format": []string{"json"}}, key)
		if err != nil {
			return nil, nil, err
		}
		return mustDecodeJSON(resp.body), requestMeta(resp.requestURL), nil
	}
	if documentNumber == "" {
		return nil, nil, errors.New("pass --id or --document-number")
	}
	params := url.Values{}
	params.Set("f.dokumentnummer", documentNumber)
	params.Set("format", "json")
	resp, err := apiGet(entity, params, key)
	if err != nil {
		return nil, nil, err
	}
	data := mustDecodeJSON(resp.body)
	docs := takeDocuments(data, 1)
	if len(docs) == 0 {
		return nil, requestMeta(resp.requestURL), errors.New("no record found for document number")
	}
	return docs[0], requestMeta(resp.requestURL), nil
}

func apiGet(path string, params url.Values, apiKey string) (apiResponse, error) {
	u, err := url.Parse(baseURL + "/" + strings.TrimPrefix(path, "/"))
	if err != nil {
		return apiResponse{}, err
	}
	q := u.Query()
	for k, values := range params {
		for _, v := range values {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "ApiKey "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, err
	}
	out := apiResponse{
		statusCode:  resp.StatusCode,
		contentType: resp.Header.Get("Content-Type"),
		body:        body,
		requestURL:  u.String(),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("DIP API returned HTTP %d: %s", resp.StatusCode, preview(string(body), 500))
	}
	return out, nil
}

func mustParse(args []string) parsedArgs {
	flags := map[string]string{}
	params := url.Values{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			fail(2, "invalid_arguments", "unexpected positional argument: "+arg)
		}
		nameValue := strings.TrimPrefix(arg, "--")
		name := nameValue
		value := ""
		if strings.Contains(nameValue, "=") {
			parts := strings.SplitN(nameValue, "=", 2)
			name = parts[0]
			value = parts[1]
		} else {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				value = "true"
			} else {
				i++
				value = args[i]
			}
		}
		if name == "param" {
			key, val, ok := strings.Cut(value, "=")
			if !ok || key == "" {
				fail(2, "invalid_arguments", "--param must be key=value")
			}
			params.Add(key, val)
			continue
		}
		flags[name] = value
	}
	return parsedArgs{flags: flags, params: params}
}

func mustAPIKey(flags map[string]string) string {
	key, _ := resolveAPIKey(flags)
	if key == "" {
		fail(2, "missing_api_key", "set DIP_API_KEY or pass --apikey")
	}
	return key
}

func resolveAPIKey(flags map[string]string) (string, string) {
	if flags["apikey"] != "" {
		return flags["apikey"], "flag"
	}
	if os.Getenv("DIP_API_KEY") != "" {
		return os.Getenv("DIP_API_KEY"), "env:DIP_API_KEY"
	}
	return "", "missing"
}

func requestMeta(rawURL string) map[string]any {
	redacted := redactURL(rawURL)
	return map[string]any{
		"method":     "GET",
		"url":        redacted,
		"redactions": []string{"Authorization", "apikey"},
	}
}

func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if q.Has("apikey") {
		q.Set("apikey", "REDACTED")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func mustDecodeJSON(body []byte) map[string]any {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		fail(1, "decode_failed", err.Error())
	}
	return data
}

func mustMarshal(v any) []byte {
	body, _ := json.Marshal(v)
	return body
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fail(1, "output_failed", err.Error())
	}
}

func writeBody(body []byte) {
	os.Stdout.Write(body)
	if len(body) == 0 || body[len(body)-1] != '\n' {
		fmt.Println()
	}
}

func fail(exitCode int, code string, message string) {
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": now(),
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
	os.Exit(exitCode)
}

func limitJSONDocuments(body []byte, limit int) ([]byte, error) {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	docs, ok := data["documents"].([]any)
	if !ok {
		return body, nil
	}
	if len(docs) > limit {
		data["documents"] = docs[:limit]
	}
	data["clientLimit"] = limit
	data["clientReturned"] = len(data["documents"].([]any))
	return json.MarshalIndent(data, "", "  ")
}

func takeDocuments(data map[string]any, limit int) []map[string]any {
	raw, ok := data["documents"].([]any)
	if !ok {
		return nil
	}
	out := []map[string]any{}
	for _, item := range raw {
		doc, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, doc)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactItem(doc map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "typ", "dokumentart", "vorgangstyp", "titel", "dokumentnummer", "wahlperiode", "herausgeber", "datum", "aktualisiert", "person_id"} {
		if value, ok := doc[key]; ok {
			out[key] = value
		}
	}
	if title := firstString(doc, "titel"); title != "" {
		out["title"] = title
	}
	sources := extractSources(doc)
	if len(sources) > 0 {
		out["sources"] = sources
	}
	return out
}

func extractSources(value any) []map[string]string {
	out := []map[string]string{}
	walkSources(value, "", &out)
	return dedupeSources(out)
}

func walkSources(value any, key string, out *[]map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			walkSources(v, k, out)
		}
	case []any:
		for _, v := range typed {
			walkSources(v, key, out)
		}
	case string:
		if strings.HasPrefix(typed, "https://") || strings.HasPrefix(typed, "http://") {
			kind := "url"
			lowerKey := strings.ToLower(key)
			switch {
			case strings.Contains(lowerKey, "pdf"):
				kind = "pdf"
			case strings.Contains(lowerKey, "xml"):
				kind = "xml"
			case strings.Contains(lowerKey, "api"):
				kind = "api"
			}
			*out = append(*out, map[string]string{"title": key, "url": typed, "kind": kind})
		}
	}
}

func dedupeSources(in []map[string]string) []map[string]string {
	seen := map[string]bool{}
	out := []map[string]string{}
	for _, src := range in {
		u := src["url"]
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, src)
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["url"] < out[j]["url"] })
	return out
}

func snippets(text string, term string, contextChars int) []map[string]any {
	if text == "" || term == "" {
		return nil
	}
	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)
	out := []map[string]any{}
	searchFrom := 0
	for {
		idx := strings.Index(lowerText[searchFrom:], lowerTerm)
		if idx < 0 {
			break
		}
		start := searchFrom + idx
		end := start + len(term)
		snippetStart := maxInt(0, start-contextChars)
		snippetEnd := minInt(len(text), end+contextChars)
		out = append(out, map[string]any{
			"start":   start,
			"end":     end,
			"snippet": cleanWhitespace(text[snippetStart:snippetEnd]),
		})
		if len(out) >= 10 {
			break
		}
		searchFrom = end
	}
	return out
}

func cleanWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func preview(s string, max int) string {
	rs := []rune(cleanWhitespace(s))
	if len(rs) <= max {
		return string(rs)
	}
	return string(rs[:max]) + "..."
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := data[key]; ok {
			switch typed := v.(type) {
			case string:
				return typed
			case json.Number:
				return typed.String()
			case float64:
				return strconv.FormatInt(int64(typed), 10)
			}
		}
	}
	return ""
}

func getNumber(data map[string]any, key string) any {
	if value, ok := data[key]; ok {
		return value
	}
	return nil
}

func cloneValues(values url.Values) url.Values {
	out := url.Values{}
	for key, vals := range values {
		for _, val := range vals {
			out.Add(key, val)
		}
	}
	return out
}

func parsePositiveInt(raw string, name string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return value, nil
}

func optionalLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := parsePositiveInt(raw, "limit")
	if err != nil {
		fail(2, "invalid_arguments", err.Error())
	}
	return value
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func isEntity(entity string) bool {
	_, ok := entitySummaries[entity]
	return ok
}

func isHelp(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
