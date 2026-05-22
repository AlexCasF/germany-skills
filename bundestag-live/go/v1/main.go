package main

import "github.com/AlexCasF/democracy-researcher/cli/pkg/simplecli"

func main() {
	app := simplecli.App{
		Name:        "bundestagctl",
		Description: "CLI for Bundestag live information feeds.",
		Commands: []simplecli.Command{
			staticXML([]string{"plenum", "speaker"}, "Current speaker feed", "https://www.bundestag.de/static/appdata/plenum/v2/speaker.xml"),
			staticXML([]string{"plenum", "conferences"}, "Plenary conference overview", "https://www.bundestag.de/static/appdata/plenum/v2/conferences.xml"),
			staticXML([]string{"committees", "list"}, "Committee index", "https://www.bundestag.de/xml/v2/ausschuesse/index.xml"),
			templatedXML([]string{"committees", "get"}, "Committee detail", "https://www.bundestag.de/xml/v2/ausschuesse/{id}.xml", "id", "Committee identifier"),
			staticXML([]string{"members", "list"}, "Member index", "https://www.bundestag.de/xml/v2/mdb/index.xml"),
			templatedXML([]string{"members", "biography"}, "Member biography", "https://www.bundestag.de/xml/v2/mdb/biografien/{id}.xml", "id", "Member identifier"),
			templatedXML([]string{"article", "get"}, "Article detail", "https://www.bundestag.de/blueprint/servlet/content/{article_id}/asAppV2NewsarticleXml", "article_id", "Article identifier"),
			staticXML([]string{"video", "feed"}, "Video feed", "http://webtv.bundestag.de/iptv/player/macros/_x_s-144277506/bttv/mobile/feed_vod.xml"),
		},
	}
	simplecli.Run(app)
}

func staticXML(path []string, summary string, rawURL string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: "Returns the published XML feed. Optional --param key=value values are appended to the query string.",
		OutputHint:  "xml",
		AllowParams: true,
		BuildURL:    simplecli.StaticURL(rawURL),
	}
}

func templatedXML(path []string, summary string, rawURL string, flagName string, flagDescription string) simplecli.Command {
	return simplecli.Command{
		Path:        path,
		Summary:     summary,
		Description: "Returns the published XML feed for the requested identifier.",
		OutputHint:  "xml",
		AllowParams: true,
		Flags: []simplecli.Flag{
			{Name: flagName, Description: flagDescription, Required: true},
		},
		BuildURL: simplecli.TemplatedURL(rawURL, map[string]string{
			"id":         flagName,
			"article_id": flagName,
		}),
	}
}
