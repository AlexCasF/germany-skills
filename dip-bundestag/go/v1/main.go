package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "dipctl",
		Description: "CLI for the DIP Bundestag API.",
		Commands: []simplecli.Command{
			dipList("vorgang", "Bundestag proceedings"),
			dipGet("vorgang", "Proceeding detail"),
			dipList("drucksache", "Printed papers"),
			dipGet("drucksache", "Printed paper detail"),
			dipList("plenarprotokoll", "Plenary protocols"),
			dipGet("plenarprotokoll", "Plenary protocol detail"),
			dipList("person", "Person master data"),
			dipGet("person", "Person detail"),
			dipList("aktivitaet", "Activities"),
			dipGet("aktivitaet", "Activity detail"),
		},
	}
	simplecli.Run(app)
}

func dipList(entity string, summary string) simplecli.Command {
	return simplecli.Command{
		Path:        []string{entity, "list"},
		Summary:     summary,
		Description: "Requires --apikey. Use --param key=value for API-specific filters.",
		OutputHint:  "json",
		AllowParams: true,
		Flags: []simplecli.Flag{
			{Name: "apikey", Description: "DIP API key", Required: true},
		},
		BuildURL: simplecli.StaticURL("https://search.dip.bundestag.de/api/v1/" + entity),
		Headers: func(flags map[string]string) map[string]string {
			return map[string]string{"Authorization": "ApiKey " + flags["apikey"]}
		},
	}
}

func dipGet(entity string, summary string) simplecli.Command {
	return simplecli.Command{
		Path:        []string{entity, "get"},
		Summary:     summary,
		Description: "Requires --id and --apikey.",
		OutputHint:  "json",
		AllowParams: true,
		Flags: []simplecli.Flag{
			{Name: "id", Description: "Entity identifier", Required: true},
			{Name: "apikey", Description: "DIP API key", Required: true},
		},
		BuildURL: simplecli.TemplatedURL("https://search.dip.bundestag.de/api/v1/"+entity+"/{id}", map[string]string{"id": "id"}),
		Headers: func(flags map[string]string) map[string]string {
			return map[string]string{"Authorization": "ApiKey " + flags["apikey"]}
		},
	}
}
