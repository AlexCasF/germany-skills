package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "abgeordnetenwatchctl",
		Description: "CLI for abgeordnetenwatch parliamentary and politician data.",
		Commands: []simplecli.Command{
			awList("parliaments", "List parliaments"),
			awGet("parliaments", "Parliament detail"),
			awList("parliament-periods", "List parliament periods"),
			awGet("parliament-periods", "Parliament period detail"),
			awList("politicians", "List politicians"),
			awGet("politicians", "Politician detail"),
			awList("candidacies-mandates", "List candidacies and mandates"),
			awGet("candidacies-mandates", "Candidacy or mandate detail"),
			awList("polls", "List polls"),
			awGet("polls", "Poll detail"),
		},
	}
	simplecli.Run(app)
}

func awList(entity string, summary string) simplecli.Command {
	return simplecli.Command{
		Path:        []string{entity, "list"},
		Summary:     summary,
		Description: "Use --param key=value for filters, sorting, pagination, and related_data.",
		OutputHint:  "json",
		AllowParams: true,
		BuildURL:    simplecli.StaticURL("https://www.abgeordnetenwatch.de/api/v2/" + entity),
	}
}

func awGet(entity string, summary string) simplecli.Command {
	return simplecli.Command{
		Path:        []string{entity, "get"},
		Summary:     summary,
		Description: "Requires --id. Use --param key=value for related_data or other supported query parameters.",
		OutputHint:  "json",
		AllowParams: true,
		Flags: []simplecli.Flag{
			{Name: "id", Description: "Entity identifier", Required: true},
		},
		BuildURL: simplecli.TemplatedURL("https://www.abgeordnetenwatch.de/api/v2/"+entity+"/{id}", map[string]string{"id": "id"}),
	}
}
