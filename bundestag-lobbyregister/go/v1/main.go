package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "lobbyregisterctl",
		Description: "CLI for the Bundestag lobby register API.",
		Commands: []simplecli.Command{
			{
				Path:        []string{"search"},
				Summary:     "Search public lobby register entries",
				Description: "Use --param key=value for q, sort, page, and other published search filters.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.lobbyregister.bundestag.de/sucheDetailJson"),
			},
		},
	}
	simplecli.Run(app)
}
