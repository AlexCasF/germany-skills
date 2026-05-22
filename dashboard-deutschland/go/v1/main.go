package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "dashboardctl",
		Description: "CLI for Dashboard Deutschland indicator and geo data.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"dashboard", "get"},
				Summary:     "Get dashboard entries",
				Description: "Use --param ids=value to request published dashboard IDs.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.dashboard-deutschland.de/api/dashboard/get"),
			},
			{
				Path:        []string{"indicators"},
				Summary:     "Get indicator tiles",
				Description: "Use --param ids=value to request one or more indicator IDs.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.dashboard-deutschland.de/api/tile/indicators"),
			},
			{
				Path:       []string{"geo"},
				Summary:    "Get Germany and Länder GeoJSON",
				OutputHint: "json",
				BuildURL:   simplecli.StaticURL("https://www.dashboard-deutschland.de/geojson/de-all.geo.json"),
			},
		},
	}
	simplecli.Run(app)
}
