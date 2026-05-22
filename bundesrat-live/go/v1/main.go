package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "bundesratctl",
		Description: "CLI for Bundesrat live information feeds.",
		Commands: []simplecli.Command{
			bundesratXML([]string{"startlist"}, "Endpoint overview", "https://www.bundesrat.de/iOS/v3/startlist_table.xml"),
			bundesratXML([]string{"news"}, "Current news", "https://www.bundesrat.de/iOS/v3/01_Aktuelles/aktuelles_table.xml"),
			bundesratXML([]string{"dates"}, "Appointments and dates", "https://www.bundesrat.de/iOS/v3/02_Termine/termine_table.xml"),
			bundesratXML([]string{"plenum", "compact"}, "Plenary compact feed", "https://www.bundesrat.de/iOS/v3/03_Plenum/plenum_kompakt_table.xml"),
			bundesratXML([]string{"plenum", "current"}, "Current plenary feed", "https://www.bundesrat.de/iOS/SharedDocs/3_Plenum/plenum_aktuelleSitzung_table.xml"),
			bundesratXML([]string{"plenum", "chronological"}, "Chronological plenary feed", "https://www.bundesrat.de/iOS/SharedDocs/3_Plenum/plenum_toChronologisch_table.xml"),
			bundesratXML([]string{"plenum", "next"}, "Next plenary sessions", "https://www.bundesrat.de/iOS/SharedDocs/3_Plenum/plenum_naechsteSitzungen.xml"),
			bundesratXML([]string{"members"}, "Members feed", "https://www.bundesrat.de/iOS/SharedDocs/2_Mitglieder/mitglieder_table.xml"),
			bundesratXML([]string{"votes"}, "Vote distribution", "https://www.bundesrat.de/iOS/v3/06_Stimmen/stimmverteilung.xml"),
			bundesratXML([]string{"presidium"}, "Presidium feed", "https://www.bundesrat.de/iOS/v3/05_Bundesrat/Praesidium/bundesrat_praesidium.xml"),
		},
	}
	simplecli.Run(app)
}

func bundesratXML(path []string, summary string, rawURL string) simplecli.Command {
	return simplecli.Command{
		Path:          path,
		Summary:       summary,
		Description:   "Returns the published Bundesrat XML feed. The CLI adds view=renderXml by default.",
		OutputHint:    "xml",
		AllowParams:   true,
		DefaultParams: map[string]string{"view": "renderXml"},
		BuildURL:      simplecli.StaticURL(rawURL),
	}
}
