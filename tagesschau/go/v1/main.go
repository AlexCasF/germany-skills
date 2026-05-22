package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "tagesschauctl",
		Description: "CLI for Tagesschau public JSON feeds.",
		Commands: []simplecli.Command{
			{
				Path:       []string{"homepage"},
				Summary:    "Homepage selections and breaking items",
				OutputHint: "json",
				BuildURL:   simplecli.StaticURL("https://www.tagesschau.de/api2u/homepage/"),
			},
			{
				Path:        []string{"news"},
				Summary:     "Filtered news feed",
				Description: "Use --param key=value for published filter parameters.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.tagesschau.de/api2u/news/"),
			},
			{
				Path:        []string{"channels"},
				Summary:     "Channels feed",
				Description: "Use --param key=value for published filter parameters.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.tagesschau.de/api2u/channels/"),
			},
			{
				Path:        []string{"search"},
				Summary:     "Search feed",
				Description: "Use --param searchText=value and other published parameters.",
				OutputHint:  "json",
				AllowParams: true,
				BuildURL:    simplecli.StaticURL("https://www.tagesschau.de/api2u/search/"),
			},
		},
	}
	simplecli.Run(app)
}
