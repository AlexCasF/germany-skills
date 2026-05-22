package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "deutschlandatlasctl",
		Description: "CLI for Deutschlandatlas table queries.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"table", "query"},
				Summary:     "Query a Deutschlandatlas table",
				Description: "Requires --table. Use --param key=value for where, outFields, returnGeometry, f, and related ArcGIS query parameters.",
				OutputHint:  "json",
				AllowParams: true,
				DefaultParams: map[string]string{
					"f":              "json",
					"returnGeometry": "false",
				},
				Flags: []simplecli.Flag{
					{Name: "table", Description: "Deutschlandatlas table name", Required: true},
				},
				BuildURL: simplecli.TemplatedURL("https://www.karto365.de/hosting/rest/services/{table}/MapServer/0/query", map[string]string{"table": "table"}),
			},
		},
	}
	simplecli.Run(app)
}
