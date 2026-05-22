package main

import (
	"bytes"
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
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName = "rechtsinformationen-bund"
	baseURL = "https://testphase.rechtsinformationen.bund.de/v1"
	rootURL = "https://testphase.rechtsinformationen.bund.de"
)

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

type legacyCommand struct {
	path      string
	kind      string
	rawFormat string
}

var legacyCommands = map[string]legacyCommand{
	"statistics":                                      {path: "/statistics", kind: "singleton"},
	"documents list":                                  {path: "/document", kind: "list"},
	"documents search":                                {path: "/document/lucene-search", kind: "list"},
	"documents search-administrative-directive":       {path: "/document/lucene-search/administrative-directive", kind: "list"},
	"documents search-case-law":                       {path: "/document/lucene-search/case-law", kind: "list"},
	"documents search-legislation":                    {path: "/document/lucene-search/legislation", kind: "list"},
	"documents search-literature":                     {path: "/document/lucene-search/literature", kind: "list"},
	"administrative-directive list":                   {path: "/administrative-directive", kind: "list"},
	"administrative-directive get":                    {path: "/administrative-directive/{documentNumber}", kind: "document"},
	"administrative-directive html":                   {path: "/administrative-directive/{documentNumber}.html", kind: "document", rawFormat: "html"},
	"administrative-directive xml":                    {path: "/administrative-directive/{documentNumber}.xml", kind: "document", rawFormat: "xml"},
	"case-law list":                                  {path: "/case-law", kind: "list"},
	"case-law courts":                                {path: "/case-law/courts", kind: "singleton"},
	"case-law get":                                   {path: "/case-law/{documentNumber}", kind: "document"},
	"case-law html":                                  {path: "/case-law/{documentNumber}.html", kind: "document", rawFormat: "html"},
	"case-law xml":                                   {path: "/case-law/{documentNumber}.xml", kind: "document", rawFormat: "xml"},
	"legislation list":                               {path: "/legislation", kind: "list"},
	"legislation get":                                {path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}", kind: "legislation"},
	"legislation html":                               {path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.html", kind: "manifestation", rawFormat: "html"},
	"legislation xml":                                {path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.xml", kind: "manifestation", rawFormat: "xml"},
	"legislation article-html":                       {path: "/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}/{articleEid}.html", kind: "article", rawFormat: "html"},
	"literature list":                                {path: "/literature", kind: "list"},
	"literature get":                                 {path: "/literature/{documentNumber}", kind: "document"},
	"literature html":                                {path: "/literature/{documentNumber}.html", kind: "document", rawFormat: "html"},
	"literature xml":                                 {path: "/literature/{documentNumber}.xml", kind: "document", rawFormat: "xml"},
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

	switch {
	case args[0] == "doctor":
		runDoctor(args[1:])
	case args[0] == "source":
		runSource(args[1:])
	case args[0] == "cite":
		runCite(args[1:])
	case len(args) >= 2 && args[0] == "documents" && args[1] == "source":
		runSource(args[2:])
	case len(args) >= 2 && args[0] == "documents" && args[1] == "text":
		runText(args[2:])
	case len(args) >= 2 && args[0] == "documents" && args[1] == "dossier":
		runDossier("documents", args[2:])
	case len(args) >= 2 && args[0] == "case-law" && args[1] == "dossier":
		runDossier("case-law", args[2:])
	case len(args) >= 2 && args[0] == "legislation" && args[1] == "dossier":
		runDossier("legislation", args[2:])
	default:
		if runLegacy(args) {
			return
		}
		fail(2, "unknown_command", "unknown command path: "+strings.Join(args, " "))
	}
}

func printRootHelp() {
	fmt.Print(`rechtsinformationen-bund -- official German federal legal information preview API

Purpose
  Search and cite legal information from the Rechtsinformationen des Bundes
  trial service: federal legislation, federal case law, legal literature, and
  administrative directives.

Use this when
  - you need official German federal legal text or court decisions
  - you need HTML/XML source renditions and stable ELI/ECLI/document identifiers
  - you need a source-backed legal evidence bundle

Do not use this when
  - you need legal advice rather than sourced legal text
  - you need state or municipal law not present in the federal portal
  - you need production certainty without checking current official sources

Fast paths
  Check service status:
    rechtsinformationen-bund doctor

  Search all indexed documents:
    rechtsinformationen-bund documents search --search-term "Bürgergeld" --limit 3

  Build an evidence bundle:
    rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision

Legacy endpoint commands
  statistics
  documents list|search|search-case-law|search-legislation
  administrative-directive list|get|html|xml
  case-law list|courts|get|html|xml
  legislation list|get|html|xml|article-html
  literature list|get|html|xml

Research commands
  doctor
  source / documents source
  documents text
  documents dossier
  case-law dossier
  legislation dossier
  cite

Output
  Legacy commands return upstream output unless new helper flags such as
  --search-term or --limit request a compact research envelope.
`)
}

func printHelp(path []string) {
	joined := strings.Join(path, " ")
	switch joined {
	case "doctor":
		fmt.Println("rechtsinformationen-bund doctor\n\nChecks preview status, statistics, auth status, and rate-limit guidance.")
	case "documents dossier":
		fmt.Println(`rechtsinformationen-bund documents dossier

What it does
  Builds a compact evidence bundle with metadata, source URLs, optional text
  snippets, warnings, and next actions.

Inputs
  --type              case-law, legislation, literature, or administrative-directive
  --document-number   Document number such as KORE600422026
  --eli               Legislation identifier such as eli/bund/bgbl-1/...
  --url               API, HTML, or XML URL emitted by a previous command
  --search-term       Search first, then choose the first result
  --grep              Optional source-text snippet term

Examples
  rechtsinformationen-bund documents dossier --type case-law --document-number KORE600422026 --grep Revision
  rechtsinformationen-bund documents dossier --search-term "Bürgergeld" --grep Bürgergeld`)
	case "documents text":
		fmt.Println("rechtsinformationen-bund documents text\n\nFetches HTML/XML source text and optionally returns --grep snippets.")
	case "source", "documents source":
		fmt.Println("rechtsinformationen-bund source\n\nPrints API, HTML, XML, ZIP, and citation metadata for one legal document.")
	case "cite":
		fmt.Println("rechtsinformationen-bund cite\n\nReturns a concise citation line for one legal record.")
	default:
		printRootHelp()
	}
}

func runDoctor(args []string) {
	_ = mustParse(args)
	resp, err := apiGet("/statistics", url.Values{})
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	stats := decodeJSON(resp.body)
	writeJSON(map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     "doctor",
		"retrievedAt": now(),
		"request":     requestMeta(resp.requestURL),
		"summary": map[string]any{
			"baseUrl":           baseURL,
			"authRequired":      false,
			"trialService":      true,
			"rateLimit":         "600 requests per minute per client IP",
			"rateLimitExceeded": "may return HTTP 503",
			"statistics":        stats,
		},
		"sources": docSources(),
		"warnings": []string{
			"This is a trial service and may change.",
			"The dataset is not yet complete.",
			"Use existing official sources for production-grade legal research.",
			"Use small size/pageIndex values during discovery.",
		},
		"nextActions": []string{
			"rechtsinformationen-bund documents search --search-term \"Bürgergeld\" --limit 3",
			"rechtsinformationen-bund case-law courts",
		},
	})
}

func runLegacy(args []string) bool {
	for width := minInt(3, len(args)); width >= 1; width-- {
		key := strings.Join(args[:width], " ")
		cmd, ok := legacyCommands[key]
		if !ok {
			continue
		}
		remaining := args[width:]
		pa := mustParse(remaining)
		if cmd.kind == "list" && pa.flags["search-term"] != "" {
			runCompactList(key, cmd.path, pa)
			return true
		}
		path, err := resolveLegacyPath(cmd, pa.flags)
		if err != nil {
			fail(2, "invalid_arguments", err.Error())
		}
		params := normalizeParams(pa)
		resp, err := apiGet(path, params)
		if err != nil {
			fail(1, "request_failed", err.Error())
		}
		body := resp.body
		if pa.flags["limit"] != "" && strings.Contains(resp.contentType, "json") {
			limit := optionalLimit(pa.flags["limit"], 10)
			limited, err := limitHydraMembers(body, limit)
			if err == nil {
				body = limited
			}
		}
		writeBody(body)
		return true
	}
	return false
}

func runCompactList(commandName string, path string, pa parsedArgs) {
	limit := optionalLimit(pa.flags["limit"], 10)
	params := normalizeParams(pa)
	if pa.flags["search-term"] != "" {
		params.Set("searchTerm", pa.flags["search-term"])
	}
	if params.Get("size") == "" {
		params.Set("size", strconv.Itoa(limit))
	}
	resp, err := apiGet(path, params)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	data := decodeJSON(resp.body)
	members := compactMembers(data, limit)
	writeJSON(map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     commandName,
		"retrievedAt": now(),
		"request":     requestMeta(resp.requestURL),
		"summary": map[string]any{
			"searchTerm":  pa.flags["search-term"],
			"totalItems":  data["totalItems"],
			"returned":    len(members),
			"clientLimit": limit,
			"nextPage":    nestedString(data, "view", "next"),
		},
		"items":    members,
		"sources":  []map[string]string{{"title": "Rechtsinformationen API", "url": rootURL + path, "kind": "api"}},
		"warnings": []string{"This is a trial service; preserve retrieval dates and source URLs."},
		"nextActions": []string{
			"rechtsinformationen-bund documents dossier --type <type> --document-number <documentNumber>",
			"rechtsinformationen-bund documents text --type <type> --document-number <documentNumber> --grep <term>",
		},
	})
}

func runSource(args []string) {
	pa := mustParse(args)
	record, request, err := resolveRecord(pa.flags)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	sources := extractSources(record)
	writeJSON(map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     "source",
		"retrievedAt": now(),
		"request":     request,
		"summary": map[string]any{
			"record":         compactRecord(record),
			"sourceCount":    len(sources),
			"citationSource": "Rechtsinformationen des Bundes",
		},
		"sources": sources,
		"warnings": []string{
			"Trial service; verify current legal status for production use.",
			"Prefer ELI/ECLI/documentNumber in citations.",
		},
		"nextActions": []string{
			"rechtsinformationen-bund documents text --type <type> --document-number <documentNumber> --grep <term>",
			"rechtsinformationen-bund cite --type <type> --document-number <documentNumber>",
		},
	})
}

func runText(args []string) {
	pa := mustParse(args)
	record, request, err := resolveRecord(pa.flags)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	text, textSource, err := sourceText(record, pa.flags)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	grep := pa.flags["grep"]
	context := optionalLimit(pa.flags["context"], 220)
	out := map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     "documents text",
		"retrievedAt": now(),
		"request": map[string]any{
			"metadata": request,
			"text":     requestMeta(textSource),
		},
		"summary": map[string]any{
			"record":       compactRecord(record),
			"textLength":   len([]rune(text)),
			"grep":         grep,
			"snippetCount": 0,
		},
		"sources":  extractSources(record),
		"warnings": []string{"Text is extracted from the official HTML/XML source rendition."},
		"nextActions": []string{
			"rechtsinformationen-bund source --type <type> --document-number <documentNumber>",
			"rechtsinformationen-bund cite --type <type> --document-number <documentNumber>",
		},
	}
	if grep != "" {
		snips := snippets(text, grep, context)
		out["snippets"] = snips
		out["summary"].(map[string]any)["snippetCount"] = len(snips)
	} else {
		out["textPreview"] = preview(text, 1800)
	}
	writeJSON(out)
}

func runDossier(defaultType string, args []string) {
	pa := mustParse(args)
	if defaultType != "documents" && pa.flags["type"] == "" {
		pa.flags["type"] = defaultType
	}
	record, request, err := resolveRecord(pa.flags)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	text := ""
	textSource := ""
	if pa.flags["grep"] != "" || pa.flags["include-text"] == "true" {
		text, textSource, _ = sourceText(record, pa.flags)
	}
	sources := extractSources(record)
	out := map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     defaultType + " dossier",
		"retrievedAt": now(),
		"request":     request,
		"summary": map[string]any{
			"record":      compactRecord(record),
			"sourceCount": len(sources),
			"citation":    citation(record),
		},
		"record":   record,
		"sources":  sources,
		"warnings": dossierWarnings(record),
		"nextActions": []string{
			"rechtsinformationen-bund documents text --type " + inferType(record, pa.flags) + " --document-number " + bestIdentifier(record),
			"rechtsinformationen-bund cite --type " + inferType(record, pa.flags) + " --document-number " + bestIdentifier(record),
		},
	}
	if text != "" && pa.flags["grep"] != "" {
		out["textRequest"] = requestMeta(textSource)
		out["snippets"] = snippets(text, pa.flags["grep"], optionalLimit(pa.flags["context"], 220))
	}
	writeJSON(out)
}

func runCite(args []string) {
	pa := mustParse(args)
	record, request, err := resolveRecord(pa.flags)
	if err != nil {
		fail(1, "request_failed", err.Error())
	}
	writeJSON(map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     "cite",
		"retrievedAt": now(),
		"request":     request,
		"summary": map[string]any{
			"record":   compactRecord(record),
			"citation": citation(record),
		},
		"sources":  extractSources(record),
		"warnings": []string{"Preserve retrieval date for trial-service citations."},
	})
}

func resolveRecord(flags map[string]string) (map[string]any, map[string]any, error) {
	if flags["url"] != "" {
		resp, err := apiGetAbsolute(flags["url"])
		if err != nil {
			return nil, nil, err
		}
		return decodeJSON(resp.body), requestMeta(resp.requestURL), nil
	}
	if flags["search-term"] != "" {
		params := url.Values{}
		params.Set("searchTerm", flags["search-term"])
		params.Set("size", "1")
		resp, err := apiGet("/document/lucene-search", params)
		if err != nil {
			return nil, nil, err
		}
		data := decodeJSON(resp.body)
		members := members(data)
		if len(members) == 0 {
			return nil, requestMeta(resp.requestURL), errors.New("no search result found")
		}
		item := mapValue(members[0], "item")
		return item, requestMeta(resp.requestURL), nil
	}
	docType := flags["type"]
	if docType == "" {
		docType = inferTypeFromFlags(flags)
	}
	docNumber := flags["document-number"]
	if flags["eli"] != "" {
		docType = "legislation"
		docNumber = flags["eli"]
	}
	if docType == "" || docNumber == "" {
		return nil, nil, errors.New("pass --type and --document-number, --eli, --url, or --search-term")
	}
	path, err := recordPath(docType, docNumber)
	if err != nil {
		return nil, nil, err
	}
	resp, err := apiGet(path, url.Values{})
	if err != nil {
		return nil, nil, err
	}
	return decodeJSON(resp.body), requestMeta(resp.requestURL), nil
}

func sourceText(record map[string]any, flags map[string]string) (string, string, error) {
	sourceURL := flags["source-url"]
	if sourceURL == "" {
		for _, src := range extractSources(record) {
			if src["kind"] == "html" {
				sourceURL = src["url"]
				break
			}
		}
	}
	if sourceURL == "" {
		for _, src := range extractSources(record) {
			if src["kind"] == "xml" {
				sourceURL = src["url"]
				break
			}
		}
	}
	if sourceURL == "" {
		return "", "", errors.New("no HTML/XML source URL found")
	}
	resp, err := apiGetAbsolute(sourceURL)
	if err != nil {
		return "", sourceURL, err
	}
	if strings.Contains(resp.contentType, "html") || strings.HasSuffix(sourceURL, ".html") {
		return stripHTML(string(resp.body)), resp.requestURL, nil
	}
	return stripXML(string(resp.body)), resp.requestURL, nil
}

func recordPath(docType string, id string) (string, error) {
	switch docType {
	case "case-law", "literature", "administrative-directive":
		return "/" + docType + "/" + url.PathEscape(id), nil
	case "legislation":
		eli := strings.TrimPrefix(id, "/v1/legislation/")
		eli = strings.TrimPrefix(eli, "legislation/")
		if !strings.HasPrefix(eli, "eli/") {
			return "", errors.New("legislation records require --eli or --document-number starting with eli/")
		}
		return "/legislation/" + eli, nil
	default:
		return "", errors.New("unsupported type: " + docType)
	}
}

func resolveLegacyPath(cmd legacyCommand, flags map[string]string) (string, error) {
	path := cmd.path
	if strings.Contains(path, "{documentNumber}") {
		value := flags["document-number"]
		if value == "" {
			return "", errors.New("missing required flag --document-number")
		}
		path = strings.ReplaceAll(path, "{documentNumber}", url.PathEscape(value))
	}
	repls := map[string]string{
		"jurisdiction":             flags["jurisdiction"],
		"agent":                    flags["agent"],
		"year":                     flags["year"],
		"naturalIdentifier":        flags["natural-identifier"],
		"pointInTime":              flags["point-in-time"],
		"version":                  flags["version"],
		"language":                 flags["language"],
		"pointInTimeManifestation": flags["point-in-time-manifestation"],
		"subtype":                  flags["subtype"],
		"articleEid":               flags["article-eid"],
	}
	for key, value := range repls {
		token := "{" + key + "}"
		if strings.Contains(path, token) {
			if value == "" {
				return "", fmt.Errorf("missing required flag --%s", camelToFlag(key))
			}
			path = strings.ReplaceAll(path, token, url.PathEscape(value))
		}
	}
	return path, nil
}

func normalizeParams(pa parsedArgs) url.Values {
	params := cloneValues(pa.params)
	if pa.flags["search-term"] != "" {
		params.Set("searchTerm", pa.flags["search-term"])
	}
	if pa.flags["size"] != "" {
		params.Set("size", pa.flags["size"])
	}
	if pa.flags["page-index"] != "" {
		params.Set("pageIndex", pa.flags["page-index"])
	}
	return params
}

func apiGet(path string, params url.Values) (apiResponse, error) {
	return apiGetAbsolute(baseURL + pathWithQuery(path, params))
}

func apiGetAbsolute(raw string) (apiResponse, error) {
	if strings.HasPrefix(raw, "/v1/") {
		raw = rootURL + raw
	}
	if strings.HasPrefix(raw, "/") {
		raw = rootURL + raw
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Accept", "*/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, err
	}
	out := apiResponse{statusCode: resp.StatusCode, contentType: resp.Header.Get("Content-Type"), body: body, requestURL: raw}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, preview(string(body), 500))
	}
	return out, nil
}

func pathWithQuery(path string, params url.Values) string {
	u := &url.URL{Path: path}
	u.RawQuery = params.Encode()
	return u.String()
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
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			i++
			value = args[i]
		} else {
			value = "true"
		}
		if name == "param" {
			k, v, ok := strings.Cut(value, "=")
			if !ok || k == "" {
				fail(2, "invalid_arguments", "--param must be key=value")
			}
			params.Add(k, v)
		} else {
			flags[name] = value
		}
	}
	return parsedArgs{flags: flags, params: params}
}

func decodeJSON(body []byte) map[string]any {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		fail(1, "decode_failed", err.Error())
	}
	return data
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
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
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": now(),
		"error": map[string]string{"code": code, "message": message},
	})
	os.Exit(exitCode)
}

func compactMembers(data map[string]any, limit int) []map[string]any {
	out := []map[string]any{}
	for _, member := range members(data) {
		item := mapValue(member, "item")
		if len(item) == 0 {
			item = member
		}
		compact := compactRecord(item)
		if matches, ok := member["textMatches"].([]any); ok {
			compact["textMatchCount"] = len(matches)
			if len(matches) > 0 {
				compact["firstTextMatch"] = matches[0]
			}
		}
		out = append(out, compact)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactRecord(record map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"@type", "@id", "documentNumber", "ecli", "headline", "name", "alternateName", "abbreviation", "legislationIdentifier", "decisionDate", "datePublished", "courtType", "courtName", "documentType", "inLanguage"} {
		if value, ok := record[key]; ok {
			out[key] = value
		}
	}
	sources := extractSources(record)
	if len(sources) > 0 {
		out["sources"] = sources
	}
	return out
}

func members(data map[string]any) []map[string]any {
	raw, ok := data["member"].([]any)
	if !ok {
		return nil
	}
	out := []map[string]any{}
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func mapValue(data map[string]any, key string) map[string]any {
	if m, ok := data[key].(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func extractSources(record map[string]any) []map[string]string {
	out := []map[string]string{}
	if id, ok := record["@id"].(string); ok && id != "" {
		out = append(out, map[string]string{"title": "@id", "url": absoluteURL(id), "kind": "api"})
	}
	if encodings, ok := record["encoding"].([]any); ok {
		for _, raw := range encodings {
			enc, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			contentURL, _ := enc["contentUrl"].(string)
			format, _ := enc["encodingFormat"].(string)
			if contentURL == "" {
				continue
			}
			kind := "url"
			switch {
			case strings.Contains(format, "html") || strings.HasSuffix(contentURL, ".html"):
				kind = "html"
			case strings.Contains(format, "xml") || strings.HasSuffix(contentURL, ".xml"):
				kind = "xml"
			case strings.Contains(format, "zip") || strings.HasSuffix(contentURL, ".zip"):
				kind = "zip"
			}
			out = append(out, map[string]string{"title": format, "url": absoluteURL(contentURL), "kind": kind})
		}
	}
	return dedupeSources(out)
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

func absoluteURL(raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return rootURL + raw
	}
	return rootURL + "/" + raw
}

func stripHTML(raw string) string {
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	raw = reScript.ReplaceAllString(raw, " ")
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	raw = reTags.ReplaceAllString(raw, " ")
	return clean(html.UnescapeString(raw))
}

func stripXML(raw string) string {
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	raw = reTags.ReplaceAllString(raw, " ")
	return clean(html.UnescapeString(raw))
}

func snippets(text string, term string, contextChars int) []map[string]any {
	lower := strings.ToLower(text)
	needle := strings.ToLower(term)
	out := []map[string]any{}
	from := 0
	for len(out) < 10 {
		idx := strings.Index(lower[from:], needle)
		if idx < 0 {
			break
		}
		start := from + idx
		end := start + len(term)
		s := maxInt(0, start-contextChars)
		e := minInt(len(text), end+contextChars)
		out = append(out, map[string]any{"start": start, "end": end, "snippet": clean(text[s:e])})
		from = end
	}
	return out
}

func citation(record map[string]any) string {
	if ecli, ok := record["ecli"].(string); ok && ecli != "" {
		headline := stringValue(record, "headline")
		if headline != "" {
			return headline + ", " + ecli
		}
		return ecli
	}
	if name := stringValue(record, "name"); name != "" {
		if eli := stringValue(record, "legislationIdentifier"); eli != "" {
			return name + ", " + eli
		}
		return name
	}
	if doc := stringValue(record, "documentNumber"); doc != "" {
		return doc
	}
	return "Rechtsinformationen des Bundes record"
}

func dossierWarnings(record map[string]any) []string {
	warnings := []string{
		"This is a trial service and the API/data surface may change.",
		"Do not treat this CLI output as legal advice.",
		"Verify current legal status when legal accuracy is high stakes.",
	}
	if force := stringValue(record, "legislationLegalForce"); force != "" && force != "InForce" {
		warnings = append(warnings, "Legislation legal force is "+force+".")
	}
	return warnings
}

func inferType(record map[string]any, flags map[string]string) string {
	if flags["type"] != "" {
		return flags["type"]
	}
	t := stringValue(record, "@type")
	switch t {
	case "Decision":
		return "case-law"
	case "Legislation":
		return "legislation"
	}
	if id := stringValue(record, "@id"); strings.Contains(id, "/case-law/") {
		return "case-law"
	} else if strings.Contains(id, "/legislation/") {
		return "legislation"
	}
	return "case-law"
}

func inferTypeFromFlags(flags map[string]string) string {
	doc := flags["document-number"]
	if strings.HasPrefix(doc, "K") {
		return "case-law"
	}
	if strings.HasPrefix(doc, "eli/") || strings.Contains(doc, "/eli/") {
		return "legislation"
	}
	return flags["type"]
}

func bestIdentifier(record map[string]any) string {
	for _, key := range []string{"documentNumber", "legislationIdentifier", "@id"} {
		if value := stringValue(record, key); value != "" {
			if key == "@id" {
				value = strings.TrimPrefix(value, "/v1/legislation/")
				value = strings.TrimPrefix(value, "/v1/case-law/")
			}
			return value
		}
	}
	return "<id>"
}

func requestMeta(rawURL string) map[string]any {
	return map[string]any{"method": "GET", "url": rawURL, "redactions": []string{}}
}

func docSources() []map[string]string {
	return []map[string]string{
		{"title": "Portal", "url": "https://testphase.rechtsinformationen.bund.de/", "kind": "portal"},
		{"title": "API documentation", "url": "https://docs.rechtsinformationen.bund.de/", "kind": "documentation"},
		{"title": "Getting started", "url": "https://docs.rechtsinformationen.bund.de/get-started", "kind": "documentation"},
		{"title": "Rate limiting", "url": "https://docs.rechtsinformationen.bund.de/guides/rate-limiting", "kind": "documentation"},
		{"title": "OpenAPI JSON", "url": "https://testphase.rechtsinformationen.bund.de/openapi.json", "kind": "openapi"},
	}
}

func limitHydraMembers(body []byte, limit int) ([]byte, error) {
	data := decodeJSON(body)
	if raw, ok := data["member"].([]any); ok && len(raw) > limit {
		data["member"] = raw[:limit]
		data["clientLimit"] = limit
		data["clientReturned"] = limit
		return json.MarshalIndent(data, "", "  ")
	}
	return body, nil
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

func optionalLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		fail(2, "invalid_arguments", "--limit must be a positive integer")
	}
	return value
}

func stringValue(data map[string]any, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func nestedString(data map[string]any, parent string, key string) string {
	if child, ok := data[parent].(map[string]any); ok {
		if value, ok := child[key].(string); ok {
			return absoluteURL(value)
		}
	}
	return ""
}

func clean(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func preview(text string, limit int) string {
	text = clean(text)
	if len([]rune(text)) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "..."
}

func camelToFlag(value string) string {
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	return strings.ToLower(re.ReplaceAllString(value, `${1}-${2}`))
}

func isHelp(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
