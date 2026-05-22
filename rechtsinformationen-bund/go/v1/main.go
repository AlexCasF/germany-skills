package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

const baseURL = "https://testphase.rechtsinformationen.bund.de/v1"

func main() {
	app := simplecli.App{
		Name:        "rechtsinformationenctl",
		Description: "CLI for the Rechtsinformationen des Bundes preview API.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"statistics"},
				Summary:     "Dataset counts by document family",
				Description: "Returns the preview service statistics payload.",
				OutputHint:  "json",
				BuildURL:    simplecli.StaticURL(baseURL + "/statistics"),
			},
			listCommand([]string{"documents", "list"}, "List documents across all indexed families", "Use --param key=value for size, pageIndex, sort, and other published filters.", baseURL+"/document"),
			listCommand([]string{"documents", "search"}, "Full-text search across all indexed families", "Use --param searchTerm=value plus size, pageIndex, sort, or other published filters.", baseURL+"/document/lucene-search"),
			listCommand([]string{"documents", "search-administrative-directive"}, "Full-text search limited to administrative directives", "Use --param searchTerm=value plus size, pageIndex, sort, or other published filters.", baseURL+"/document/lucene-search/administrative-directive"),
			listCommand([]string{"documents", "search-case-law"}, "Full-text search limited to case law", "Use --param searchTerm=value plus size, pageIndex, sort, or other published filters.", baseURL+"/document/lucene-search/case-law"),
			listCommand([]string{"documents", "search-legislation"}, "Full-text search limited to legislation", "Use --param searchTerm=value plus size, pageIndex, sort, or other published filters.", baseURL+"/document/lucene-search/legislation"),
			listCommand([]string{"documents", "search-literature"}, "Full-text search limited to literature", "Use --param searchTerm=value plus size, pageIndex, sort, or other published filters.", baseURL+"/document/lucene-search/literature"),
			listCommand([]string{"administrative-directive", "list"}, "List administrative directives", "Use --param key=value for published filters such as searchTerm, dateFrom, dateTo, size, pageIndex, and sort.", baseURL+"/administrative-directive"),
			documentNumberCommand([]string{"administrative-directive", "get"}, "Administrative-directive detail metadata", "Requires --document-number.", "json", baseURL+"/administrative-directive/{documentNumber}"),
			documentNumberCommand([]string{"administrative-directive", "html"}, "Administrative-directive HTML rendition", "Requires --document-number.", "html", baseURL+"/administrative-directive/{documentNumber}.html"),
			documentNumberCommand([]string{"administrative-directive", "xml"}, "Administrative-directive XML rendition", "Requires --document-number.", "xml", baseURL+"/administrative-directive/{documentNumber}.xml"),
			listCommand([]string{"case-law", "list"}, "List and search case law", "Use --param key=value for searchTerm, dateFrom, dateTo, courtType, size, pageIndex, and other published filters.", baseURL+"/case-law"),
			{
				Path:        []string{"case-law", "courts"},
				Summary:     "Court inventory counts",
				Description: "Returns court labels and document counts for case-law materials.",
				OutputHint:  "json",
				BuildURL:    simplecli.StaticURL(baseURL + "/case-law/courts"),
			},
			documentNumberCommand([]string{"case-law", "get"}, "Case-law detail metadata and text", "Requires --document-number.", "json", baseURL+"/case-law/{documentNumber}"),
			documentNumberCommand([]string{"case-law", "html"}, "Case-law HTML rendition", "Requires --document-number.", "html", baseURL+"/case-law/{documentNumber}.html"),
			documentNumberCommand([]string{"case-law", "xml"}, "Case-law XML rendition", "Requires --document-number.", "xml", baseURL+"/case-law/{documentNumber}.xml"),
			listCommand([]string{"legislation", "list"}, "List and search legislation expressions", "Use --param key=value for searchTerm, eli, temporalCoverageFrom, temporalCoverageTo, mostRelevantOn, dateFrom, dateTo, size, pageIndex, and sort.", baseURL+"/legislation"),
			legislationExpressionCommand([]string{"legislation", "get"}, "Legislation expression detail metadata", "Requires the ELI path components for a single legislation expression.", "json", baseURL+"/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}"),
			legislationManifestationCommand([]string{"legislation", "html"}, "Legislation HTML rendition", "Requires the ELI path components plus manifestation date and subtype.", "html", baseURL+"/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.html"),
			legislationManifestationCommand([]string{"legislation", "xml"}, "Legislation XML rendition", "Requires the ELI path components plus manifestation date and subtype.", "xml", baseURL+"/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}.xml"),
			legislationArticleHTMLCommand(),
			listCommand([]string{"literature", "list"}, "List and search legal literature metadata", "Use --param key=value for documentNumber, yearOfPublication, author, collaborator, searchTerm, dateFrom, dateTo, size, pageIndex, and sort.", baseURL+"/literature"),
			documentNumberCommand([]string{"literature", "get"}, "Literature detail metadata", "Requires --document-number.", "json", baseURL+"/literature/{documentNumber}"),
			documentNumberCommand([]string{"literature", "html"}, "Literature HTML rendition", "Requires --document-number.", "html", baseURL+"/literature/{documentNumber}.html"),
			documentNumberCommand([]string{"literature", "xml"}, "Literature XML rendition", "Requires --document-number.", "xml", baseURL+"/literature/{documentNumber}.xml"),
		},
	}
	simplecli.Run(app)
}

func listCommand(path []string, summary string, description string, rawURL string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: description,
		OutputHint:  "json",
		AllowParams: true,
		BuildURL:    simplecli.StaticURL(rawURL),
	}
}

func documentNumberCommand(path []string, summary string, description string, outputHint string, rawTemplate string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: description,
		OutputHint:  outputHint,
		Flags: []simplecli.Flag{
			{Name: "document-number", Description: "The published document number", Required: true},
		},
		BuildURL: simplecli.TemplatedURL(rawTemplate, map[string]string{
			"documentNumber": "document-number",
		}),
	}
}

func legislationExpressionCommand(path []string, summary string, description string, outputHint string, rawTemplate string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: description,
		OutputHint:  outputHint,
		Flags:       legislationExpressionFlags(),
		BuildURL: simplecli.TemplatedURL(rawTemplate, map[string]string{
			"jurisdiction":      "jurisdiction",
			"agent":             "agent",
			"year":              "year",
			"naturalIdentifier": "natural-identifier",
			"pointInTime":       "point-in-time",
			"version":           "version",
			"language":          "language",
		}),
	}
}

func legislationManifestationCommand(path []string, summary string, description string, outputHint string, rawTemplate string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: description,
		OutputHint:  outputHint,
		Flags:       legislationManifestationFlags(),
		BuildURL: simplecli.TemplatedURL(rawTemplate, map[string]string{
			"jurisdiction":             "jurisdiction",
			"agent":                    "agent",
			"year":                     "year",
			"naturalIdentifier":        "natural-identifier",
			"pointInTime":              "point-in-time",
			"version":                  "version",
			"language":                 "language",
			"pointInTimeManifestation": "point-in-time-manifestation",
			"subtype":                  "subtype",
		}),
	}
}

func legislationArticleHTMLCommand() simplecli.Command {
	return simplecli.Command{
		Path:        []string{"legislation", "article-html"},
		Summary:     "Legislation article-level HTML rendition",
		Description: "Requires the ELI path components, manifestation date, subtype, and article EID.",
		OutputHint:  "html",
		Flags: append(
			legislationManifestationFlags(),
			simplecli.Flag{Name: "article-eid", Description: "Article identifier such as art-z6", Required: true},
		),
		BuildURL: simplecli.TemplatedURL(
			baseURL+"/legislation/eli/{jurisdiction}/{agent}/{year}/{naturalIdentifier}/{pointInTime}/{version}/{language}/{pointInTimeManifestation}/{subtype}/{articleEid}.html",
			map[string]string{
				"jurisdiction":             "jurisdiction",
				"agent":                    "agent",
				"year":                     "year",
				"naturalIdentifier":        "natural-identifier",
				"pointInTime":              "point-in-time",
				"version":                  "version",
				"language":                 "language",
				"pointInTimeManifestation": "point-in-time-manifestation",
				"subtype":                  "subtype",
				"articleEid":               "article-eid",
			},
		),
	}
}

func legislationExpressionFlags() []simplecli.Flag {
	return []simplecli.Flag{
		{Name: "jurisdiction", Description: "ELI jurisdiction segment such as bund", Required: true},
		{Name: "agent", Description: "ELI agent segment such as bgbl-1", Required: true},
		{Name: "year", Description: "ELI year segment", Required: true},
		{Name: "natural-identifier", Description: "ELI natural identifier segment such as s2704", Required: true},
		{Name: "point-in-time", Description: "Expression point in time such as 2025-01-01", Required: true},
		{Name: "version", Description: "Expression version segment such as 1", Required: true},
		{Name: "language", Description: "Language segment such as deu", Required: true},
	}
}

func legislationManifestationFlags() []simplecli.Flag {
	return append(
		legislationExpressionFlags(),
		simplecli.Flag{Name: "point-in-time-manifestation", Description: "Manifestation date such as 2026-02-17", Required: true},
		simplecli.Flag{Name: "subtype", Description: "Manifestation subtype such as regelungstext-verkuendung-1", Required: true},
	)
}
