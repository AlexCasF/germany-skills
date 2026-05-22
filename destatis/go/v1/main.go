package main

import (
	"net/url"

	"github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"
)

func main() {
	app := simplecli.App{
		Name:        "destatisctl",
		Description: "CLI for the Destatis and GENESIS-style statistical API.",
		Commands: []simplecli.Command{
			destatis("catalogue", "statistics", "Catalogue statistics", "/catalogue/statistics"),
			destatis("catalogue", "tables", "Catalogue tables", "/catalogue/tables"),
			destatis("catalogue", "variables", "Catalogue variables", "/catalogue/variables"),
			destatis("metadata", "table", "Table metadata", "/metadata/table"),
			destatis("metadata", "timeseries", "Time series metadata", "/metadata/timeseries"),
			destatis("data", "table", "Table data", "/data/table"),
			destatis("data", "timeseries", "Time series data", "/data/timeseries"),
			destatis("find", "search", "Search", "/find/find"),
		},
	}
	simplecli.Run(app)
}

func destatis(group string, action string, summary string, path string) simplecli.Command {
	return simplecli.Command{
		Path:        []string{group, action},
		Summary:     summary,
		Description: "Requires --username and --password. Use --param key=value for table, statistic, area, and other published query parameters.",
		OutputHint:  "json",
		AllowParams: true,
		Flags: []simplecli.Flag{
			{Name: "username", Description: "Destatis username", Required: true},
			{Name: "password", Description: "Destatis password", Required: true},
		},
		BuildURL: func(flags map[string]string, params url.Values) (string, error) {
			params.Set("username", flags["username"])
			params.Set("password", flags["password"])
			return simplecli.StaticURL("https://www-genesis.destatis.de/genesisWS/rest/2020" + path)(flags, params)
		},
	}
}
