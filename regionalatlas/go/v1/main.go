package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "regionalatlasctl",
		Description: "CLI for Regionalatlas dynamic-layer queries.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"query"},
				Summary:     "Run a Regionalatlas dynamic-layer query",
				Description: "Use --param key=value for layer, where, outFields, returnGeometry, f, and other ArcGIS query arguments.",
				OutputHint:  "json",
				AllowParams: true,
				DefaultParams: map[string]string{
					"f":              "json",
					"returnGeometry": "false",
				},
				BuildURL: simplecli.StaticURL("https://www.gis-idmz.nrw.de/arcgis/rest/services/stba/regionalatlas/MapServer/dynamicLayer/query"),
			},
		},
	}
	simplecli.Run(app)
}
