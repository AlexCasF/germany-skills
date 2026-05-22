package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "bundeshaushaltctl",
		Description: "CLI for Bundeshaushalt Digital budget data.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"budget-data"},
				Summary:     "Fetch federal budget hierarchy data",
				Description: "Use --param key=value for year, account, and other published query parameters.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://bundeshaushalt.de/internalapi/budgetData"),
			},
		},
	}
	simplecli.Run(app)
}
