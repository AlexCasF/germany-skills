package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName             = "bundestagctl"
	baseURL             = "https://www.bundestag.de"
	speakerURL          = baseURL + "/static/appdata/plenum/v2/speaker.xml"
	conferencesURL      = baseURL + "/static/appdata/plenum/v2/conferences.xml"
	committeesURL       = baseURL + "/xml/v2/ausschuesse/index.xml"
	committeeURLPattern = baseURL + "/xml/v2/ausschuesse/%s.xml"
	membersURL          = baseURL + "/xml/v2/mdb/index.xml"
	memberURLPattern    = baseURL + "/xml/v2/mdb/biografien/%s.xml"
	articleURLPattern   = baseURL + "/blueprint/servlet/content/%s/asAppV2NewsarticleXml"
	videoURL            = "http://webtv.bundestag.de/iptv/player/macros/_x_s-144277506/bttv/mobile/feed_vod.xml"
	openAPIURL          = "https://github.com/bundesAPI/bundestag-api"
	openDataURL         = baseURL + "/services/opendata"
	imprintURL          = baseURL + "/services/impressum"
	mediaTermsURL       = baseURL + "/mediathek/nutzungsbedingungen-247892"
	privacyURL          = baseURL + "/en/service/privacy"
	defaultLimit        = 10
	safeLimit           = 100
	defaultTimeout      = 45 * time.Second
)

type parsedArgs struct {
	flags       map[string]string
	params      url.Values
	positionals []string
}

type cliError struct {
	exitCode int
	code     string
	message  string
}

func (e cliError) Error() string {
	return e.message
}

type httpError struct {
	statusCode int
	body       string
	url        string
}

func (e httpError) Error() string {
	return fmt.Sprintf("upstream status %d from %s: %s", e.statusCode, e.url, truncate(stripSpace(e.body), 300))
}

type textNode struct {
	Status string `xml:"status,attr"`
	Text   string `xml:",chardata"`
}

type memberIndexXML struct {
	DocumentStand string           `xml:"dokumentInfo>dokumentStand"`
	Members       []memberListItem `xml:"mdbs>mdb"`
}

type memberListItem struct {
	Fraction             string       `xml:"fraktion,attr"`
	ID                   textNode     `xml:"mdbID"`
	Name                 textNode     `xml:"mdbName"`
	BioURL               string       `xml:"mdbBioURL"`
	InfoXMLURL           string       `xml:"mdbInfoXMLURL"`
	State                string       `xml:"mdbLand"`
	Constituency         constituency `xml:"mdbWahlkreis"`
	ElectionType         string       `xml:"mdbGewaehlt"`
	PhotoURL             string       `xml:"mdbFotoURL"`
	LargePhotoURL        string       `xml:"mdbFotoGrossURL"`
	PhotoLastChanged     string       `xml:"mdbFotoLastChanged"`
	LastChanged          string       `xml:"lastChanged"`
	ChangedDateTime      string       `xml:"changedDateTime"`
	ImageAltText         string       `xml:"imageAltText"`
	MitmischenInfoXMLURL string       `xml:"mdbInfoXMLURLMitmischen"`
}

type constituency struct {
	Number string `xml:"mdbWahlkreisNummer"`
	Name   string `xml:"mdbWahlkreisName"`
	URL    string `xml:"mdbWahlkreisURL"`
}

type memberBiographyXML struct {
	DocumentStand string         `xml:"dokumentInfo>dokumentStand"`
	Info          memberBioInfo  `xml:"mdbInfo"`
	Media         memberBioMedia `xml:"mdbMedien"`
	RawXMLName    xml.Name       `xml:"mdb"`
}

type memberBioInfo struct {
	ArticleID       string       `xml:"articleId"`
	SourceURL       string       `xml:"sourceURL"`
	ID              textNode     `xml:"mdbID"`
	ExitDate        string       `xml:"mdbAustrittsdatum"`
	LastName        string       `xml:"mdbZuname"`
	FirstName       string       `xml:"mdbVorname"`
	AcademicTitle   string       `xml:"mdbAkademischerTitel"`
	BirthDate       string       `xml:"mdbGeburtsdatum"`
	Religion        string       `xml:"mdbReligionKonfession"`
	Profession      string       `xml:"mdbBeruf"`
	Gender          string       `xml:"mdbGeschlecht"`
	Fraction        string       `xml:"mdbFraktion"`
	Party           string       `xml:"mdbPartei"`
	State           string       `xml:"mdbLand"`
	Constituency    constituency `xml:"mdbWahlkreis"`
	ElectionType    string       `xml:"mdbGewaehlt"`
	BioURL          string       `xml:"mdbBioURL"`
	BiographyHTML   string       `xml:"mdbBiografischeInformationen"`
	InterestingHTML string       `xml:"mdbWissenswertes"`
	HomepageURL     string       `xml:"mdbHomepageURL"`
	DisclosureHTML  string       `xml:"mdbVeroeffentlichungspflichtigeAngaben"`
	OtherWebsites   []websiteRef `xml:"mdbSonstigeWebsites>mdbSonstigeWebsite"`
}

type websiteRef struct {
	Title string `xml:"mdbSonstigeWebsiteTitel" json:"title"`
	URL   string `xml:"mdbSonstigeWebsiteURL" json:"url"`
}

type memberBioMedia struct {
	Photo              memberPhoto `xml:"mdbFoto"`
	SpeechesURL        string      `xml:"mdbRedenVorPlenumURL"`
	SpeechesRSS        string      `xml:"mdbRedenVorPlenumRSS"`
	MediaLibraryMember string      `xml:"mdbMediathekURL"`
}

type memberPhoto struct {
	URL       string `xml:"mdbFotoURL"`
	Copyright string `xml:"mdbFotoCopyright"`
	LargeURL  string `xml:"mdbFotoGrossURL"`
	AltText   string `xml:"imageAltText"`
}

type committeeIndexXML struct {
	DocumentStand string              `xml:"dokumentInfo>dokumentStand"`
	Committees    []committeeListItem `xml:"ausschuesse>ausschuss"`
	StartArticle  articleBrief        `xml:"ausschuessestartartikel"`
}

type committeeListItem struct {
	ID              string `xml:"id,attr"`
	Name            string `xml:"ausschussName"`
	ShortName       string `xml:"ausschussKurzName"`
	TeaserHTML      string `xml:"ausschussTeaser"`
	DetailXMLURL    string `xml:"ausschussDetailXML"`
	ImageURL        string `xml:"imageURL"`
	LargeImageURL   string `xml:"imageGrossURL"`
	ImageCopyright  string `xml:"imageCopyright"`
	ImageAltText    string `xml:"imageAltText"`
	LastChanged     string `xml:"lastChanged"`
	ChangedDateTime string `xml:"changedDateTime"`
}

type committeeDetailXML struct {
	DocumentStand  string                `xml:"dokumentInfo>dokumentStand"`
	ID             string                `xml:"ausschussId"`
	Name           string                `xml:"ausschussName"`
	SourceURL      string                `xml:"ausschussSourceURL"`
	TaskHTML       string                `xml:"ausschussAufgabe"`
	ContactHTML    string                `xml:"ausschussKontakt"`
	ChairID        string                `xml:"ausschussVorsitzId"`
	ImageURL       string                `xml:"ausschussBildURL"`
	ImageCopyright string                `xml:"ausschussBildCopyright"`
	ImageAltText   string                `xml:"imageAltText"`
	Members        []committeeMemberItem `xml:"ausschussMitglieder>mdb"`
	News           []committeeNewsItem   `xml:"newslist>news"`
}

type committeeMemberItem struct {
	Fraction     string   `xml:"fraktion,attr"`
	ID           textNode `xml:"mdbID"`
	Name         textNode `xml:"mdbName"`
	BioURL       string   `xml:"mdbBioURL"`
	InfoXMLURL   string   `xml:"mdbInfoXMLURL"`
	State        string   `xml:"mdbLand"`
	Role         string   `xml:"role"`
	LastChanged  string   `xml:"lastChanged"`
	PhotoURL     string   `xml:"mdbFotoURL"`
	ImageAltText string   `xml:"imageAltText"`
}

type committeeNewsItem struct {
	ArticleID       string   `xml:"articleId,attr"`
	Date            string   `xml:"date"`
	Title           string   `xml:"title"`
	TeaserHTML      string   `xml:"teaser"`
	DetailsXML      string   `xml:"detailsXML"`
	VideoURL        string   `xml:"video-stream>url"`
	LastChanged     string   `xml:"lastchanged"`
	ChangedDateTime string   `xml:"changedDateTime"`
	Fields          []string `xml:"politikfelder>politikfeld"`
}

type articleBrief struct {
	Date            string `xml:"date"`
	Title           string `xml:"title"`
	TextHTML        string `xml:"text"`
	ChangedDateTime string `xml:"changedDateTime"`
}

type conferencesXML struct {
	Days []conferenceDay `xml:"tagesordnung"`
}

type conferenceDay struct {
	Date          string            `xml:"date"`
	Active        string            `xml:"active"`
	SessionNumber string            `xml:"sitzungsnummer"`
	Name          string            `xml:"name"`
	Items         []discussionPoint `xml:"diskussionspunkte>diskussionspunkt"`
}

type discussionPoint struct {
	StartTime string `xml:"startzeit"`
	EndTime   string `xml:"endzeit"`
	Status    string `xml:"status"`
	Title     string `xml:"titel"`
	ArticleID string `xml:"articleId"`
	Top       string `xml:"top"`
}

type speakerXML struct {
	TopicNumber string        `xml:"topicNumber"`
	Live        string        `xml:"live"`
	Speakers    []speakerItem `xml:"speakers>speaker"`
}

type speakerItem struct {
	FirstName string `xml:"firstName"`
	LastName  string `xml:"lastName"`
	Name      string `xml:"name"`
	Fraction  string `xml:"fraction"`
	Party     string `xml:"party"`
	ID        string `xml:"id"`
}

type articleXML struct {
	ArticleID       string   `xml:"articleId"`
	SourceURL       string   `xml:"sourceURL"`
	Date            string   `xml:"date"`
	Title           string   `xml:"title"`
	TextHTML        string   `xml:"text"`
	Fields          []string `xml:"politikfelder>politikfeld"`
	ChangedDateTime string   `xml:"changedDateTime"`
	ImageURL        string   `xml:"imageURL"`
	ImageCopyright  string   `xml:"imageCopyright"`
	ImageAltText    string   `xml:"imageAltText"`
}

type videoFeedXML struct {
	Groups []videoGroup `xml:"group"`
}

type videoGroup struct {
	Type    string        `xml:"type,attr"`
	Streams []videoStream `xml:"stream"`
}

type videoStream struct {
	Bandwidth string `xml:"bandwidth,attr"`
	Href      string `xml:"href,attr"`
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 || isHelp(args[0]) {
		printRootHelp()
		return
	}
	if isHelp(args[len(args)-1]) {
		printHelp(args[:len(args)-1])
		return
	}

	var err error
	switch {
	case args[0] == "doctor":
		err = runDoctor(args[1:])
	case args[0] == "examples":
		printExamples()
	case match(args, "plenum", "speaker"):
		err = runPlenumSpeaker(args[2:])
	case match(args, "plenum", "conferences"):
		err = runPlenumConferences(args[2:])
	case match(args, "plenum", "agenda"):
		err = runPlenumConferences(args[2:])
	case match(args, "members", "list"):
		err = runMembersList(args[2:])
	case match(args, "members", "search"):
		err = runMembersSearch(args[2:])
	case match(args, "members", "biography"):
		err = runMemberBiography(args[2:])
	case match(args, "members", "dossier"):
		err = runMemberDossier(args[2:])
	case match(args, "committees", "list"):
		err = runCommitteesList(args[2:])
	case match(args, "committees", "search"):
		err = runCommitteesSearch(args[2:])
	case match(args, "committees", "get"):
		err = runCommitteeGet(args[2:])
	case match(args, "committees", "dossier"):
		err = runCommitteeDossier(args[2:])
	case match(args, "article", "get"):
		err = runArticleGet(args[2:])
	case match(args, "article", "page"):
		err = runArticlePage(args[2:])
	case match(args, "video", "feed"):
		err = runVideoFeed(args[2:])
	case args[0] == "source":
		err = runSource(args[1:])
	default:
		err = cliError{2, "unknown_command", "unknown command; run bundestagctl --help"}
	}
	if err != nil {
		var ce cliError
		if errors.As(err, &ce) {
			fail(ce.exitCode, ce.code, ce.message)
		}
		fail(1, "unexpected_error", err.Error())
	}
}

func printRootHelp() {
	fmt.Println(`bundestagctl -- Bundestag live/site XML research CLI

Purpose
  Discover and normalize public Bundestag live/site XML feeds for current
  plenary agenda data, members, biographies, committees, article details, and
  video feed metadata.

Use this when
  - you need current or near-current Bundestag website/app data
  - you need official member profile URLs, biographies, committees, or side-job
    disclosure snippets from Bundestag profile XML
  - you need agenda article IDs from the plenary live app surface

Do not use this when
  - you need complete parliamentary proceedings, printed papers, or plenary
    protocol history; use dipctl for that archive-grade research
  - you need Bundesrat proceedings; use bundesratctl

Fast paths
  bundestagctl doctor
  bundestagctl members search --name "Amthor" --limit 3
  bundestagctl members dossier --name "Amthor" --grep "Tätigkeiten"
  bundestagctl committees search --term "Arbeit" --limit 5
  bundestagctl committees dossier --id a11 --member-limit 5
  bundestagctl plenum conferences --limit 2 --item-limit 3
  bundestagctl article get --article-id 1174778
  bundestagctl article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778" --grep "Meinungsfreiheit"

Endpoint-compatible commands
  plenum speaker
  plenum conferences
  committees list
  committees get --id a11
  members list
  members biography --id 2022
  article get --article-id 1174778
  video feed --content-id 7529016

Research commands
  doctor
  examples
  members search
  members dossier
  committees search
  committees dossier
  article page
  source

Output guarantees
  Commands emit JSON envelopes with status, request, summary/items, sources,
  warnings, and nextActions. Pass --raw on endpoint commands for raw XML.`)
}

func printHelp(path []string) {
	switch strings.Join(path, " ") {
	case "members search":
		fmt.Println(`bundestagctl members search

Search the official Bundestag member index and return compact rows.

Examples
  bundestagctl members search --name "Amthor" --limit 3
  bundestagctl members search --term "AfD" --limit 5

Flags
  --name <text>       Search member name
  --term <text>       Search across name, party, fraction, state, constituency
  --limit <n>         Result cap, defaults to 10, safe max 100
  --include-raw       Include the matching raw member records`)
	case "members dossier":
		fmt.Println(`bundestagctl members dossier

Build a compact source-rich dossier for one Bundestag member.

Examples
  bundestagctl members dossier --id 2022
  bundestagctl members dossier --name "Amthor" --grep "Tätigkeiten"

Flags
  --id <mdb-id>       Bundestag member ID
  --name <text>       Resolve the member by name if ID is unknown
  --grep <term>       Return matching biography/disclosure snippets
  --include-raw       Include raw parsed biography fields`)
	case "committees dossier":
		fmt.Println(`bundestagctl committees dossier

Build a compact committee dossier with task text, members, news, and sources.

Examples
  bundestagctl committees dossier --id a11 --member-limit 5

Flags
  --id <id>           Committee ID such as a11
  --member-limit <n>  Member cap, defaults to 10, safe max 100
  --news-limit <n>    News cap, defaults to 5, safe max 50
  --grep <term>       Return snippets from task/news text`)
	case "article page":
		fmt.Println(`bundestagctl article page

Fetch and normalize a human-facing Bundestag article page or article XML ID.

Examples
  bundestagctl article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778"
  bundestagctl article page --article-id 1174778 --grep "Meinungsfreiheit"

Flags
  --url <url>         Public Bundestag article URL
  --article-id <id>   Article ID; extracted from --url when omitted
  --grep <term>       Return source snippets matching term`)
	case "plenum conferences":
		fmt.Println(`bundestagctl plenum conferences

Normalize the Bundestag live plenary conference overview.

Examples
  bundestagctl plenum conferences --limit 2 --item-limit 3
  bundestagctl plenum conferences --raw

Flags
  --limit <n>         Sitting-day cap, defaults to 10
  --item-limit <n>    Agenda-item cap per sitting day, defaults to 10
  --raw               Print the original XML only`)
	default:
		printRootHelp()
	}
}

func printExamples() {
	fmt.Println(`bundestagctl examples

1. Check health and usage hints:
   bundestagctl doctor

2. Search for a member:
   bundestagctl members search --name "Amthor" --limit 3

3. Expand a member into an official source dossier:
   bundestagctl members dossier --id 2022 --grep "Tätigkeiten"

4. Search current Bundestag committees:
   bundestagctl committees search --term "Arbeit" --limit 5

5. Expand a committee with members and recent related news:
   bundestagctl committees dossier --id a11 --member-limit 5 --news-limit 3

6. Inspect upcoming plenary agenda items and article IDs:
   bundestagctl plenum conferences --limit 2 --item-limit 5

7. Expand an agenda/article ID:
   bundestagctl article get --article-id 1174778

8. Fetch the public article page for citation snippets:
   bundestagctl article page --url "https://www.bundestag.de/dokumente/textarchiv/2026/kw21-de-demokratie-1174778" --grep "Meinungsfreiheit"

9. Fetch raw XML when you need the complete upstream record:
   bundestagctl members biography --id 2022 --raw

10. Prefer dipctl for full parliamentary proceedings and historical plenary protocols.`)
}

func runDoctor(argv []string) error {
	parsed := parseArgs(argv)
	limit := limitFlag(parsed, 3, 10)
	checks := []struct {
		name string
		url  string
	}{
		{"speaker", speakerURL},
		{"conferences", conferencesURL},
		{"committees", committeesURL},
		{"members", membersURL},
	}
	payload := envelope("doctor", baseURL, map[string]any{"limit": limit})
	summary := map[string]any{
		"authRequired":       false,
		"publishedRateLimit": "No exact published request quota was found for these public Bundestag XML feeds. Use small limits, cache repeated index calls, and back off on 429/5xx responses.",
		"fairUseHints": []string{
			"Use search commands before fetching detail records.",
			"Avoid repeated full member index downloads during one run; cache results when orchestrating.",
			"Use --limit and --item-limit on broad feeds.",
			"Treat video/media URLs under Bundestag media terms and cite Deutscher Bundestag as source.",
		},
	}
	var endpointChecks []map[string]any
	status := "ok"
	for _, check := range checks {
		code, contentType, body, err := fetchRaw(check.url)
		item := map[string]any{
			"name":        check.name,
			"url":         check.url,
			"statusCode":  code,
			"contentType": contentType,
			"bodyPreview": truncate(stripSpace(string(body)), 180),
		}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
			status = "degraded"
		} else {
			item["ok"] = true
		}
		endpointChecks = append(endpointChecks, item)
		if len(endpointChecks) >= limit {
			break
		}
	}
	summary["endpoints"] = endpointChecks
	payload["status"] = status
	payload["summary"] = summary
	payload["sources"] = defaultSources()
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		`bundestagctl members search --name "Amthor" --limit 3`,
		"bundestagctl committees search --term Arbeit --limit 5",
		"bundestagctl plenum conferences --limit 2 --item-limit 3",
	}
	emit(payload)
	return nil
}

func runMembersList(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchXMLWithParams(membersURL, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	members, err := parseMemberIndex(raw)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	items := compactMembers(members.Members, limit, flagBool(parsed, "include-raw"))
	payload := envelope("members list", requestURL, map[string]any{"limit": limit})
	payload["summary"] = map[string]any{"totalMembers": len(members.Members), "returned": len(items), "documentStand": members.DocumentStand}
	payload["items"] = items
	payload["sources"] = sources("Bundestag member XML index", membersURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{`bundestagctl members search --name "Amthor" --limit 3`}
	emit(payload)
	return nil
}

func runMembersSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["name"], parsed.flags["term"], parsed.flags["q"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "members search requires --name or --term"}
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	raw, requestURL, err := fetchXMLWithParams(membersURL, nil)
	if err != nil {
		return err
	}
	index, err := parseMemberIndex(raw)
	if err != nil {
		return err
	}
	var matches []memberListItem
	needle := strings.ToLower(term)
	for _, member := range index.Members {
		if strings.Contains(strings.ToLower(memberSearchText(member)), needle) {
			matches = append(matches, member)
		}
	}
	payload := envelope("members search", requestURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"term": term, "matches": len(matches), "returned": minInt(limit, len(matches)), "searchedMembers": len(index.Members), "documentStand": index.DocumentStand}
	payload["items"] = compactMembers(matches, limit, flagBool(parsed, "include-raw"))
	payload["sources"] = sources("Bundestag member XML index", membersURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForMembers(matches)
	emit(payload)
	return nil
}

func runMemberBiography(argv []string) error {
	parsed := parseArgs(argv)
	id, err := requiredFlag(parsed, "id", "members biography requires --id")
	if err != nil {
		return err
	}
	raw, requestURL, err := fetchXMLWithParams(fmt.Sprintf(memberURLPattern, url.PathEscape(id)), parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	bio, err := parseMemberBiography(raw)
	if err != nil {
		return err
	}
	payload := envelope("members biography", requestURL, map[string]any{"id": id})
	payload["summary"] = compactBiography(bio, parsed.flags["grep"])
	payload["items"] = []any{memberEvidence(bio, parsed.flags["grep"])}
	payload["sources"] = sourcesForMemberBiography(bio, requestURL)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{
		fmt.Sprintf("bundestagctl members dossier --id %s --grep Tätigkeiten", id),
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = bio
	}
	emit(payload)
	return nil
}

func runMemberDossier(argv []string) error {
	parsed := parseArgs(argv)
	id := parsed.flags["id"]
	var resolved *memberListItem
	if id == "" {
		term := firstNonEmpty(parsed.flags["name"], parsed.flags["term"], strings.Join(parsed.positionals, " "))
		if term == "" {
			return cliError{2, "missing_member", "members dossier requires --id or --name"}
		}
		member, err := resolveMemberByName(term)
		if err != nil {
			return err
		}
		resolved = &member
		id = member.ID.Text
	}
	raw, requestURL, err := fetchXMLWithParams(fmt.Sprintf(memberURLPattern, url.PathEscape(id)), nil)
	if err != nil {
		return err
	}
	bio, err := parseMemberBiography(raw)
	if err != nil {
		return err
	}
	payload := envelope("members dossier", requestURL, map[string]any{"id": id, "name": parsed.flags["name"], "grep": parsed.flags["grep"]})
	payload["summary"] = compactBiography(bio, parsed.flags["grep"])
	payload["items"] = []any{memberEvidence(bio, parsed.flags["grep"])}
	payload["sources"] = sourcesForMemberBiography(bio, requestURL)
	payload["warnings"] = append(defaultWarnings(), "Member biography and disclosure fields are based on Bundestag profile XML; disclosure text may reflect self-reported data and Bundestag publication rules.")
	payload["nextActions"] = []string{
		fmt.Sprintf("bundestagctl members biography --id %s --raw", id),
		fmt.Sprintf("bundestagctl source --url %q", firstNonEmpty(bio.Info.SourceURL, bio.Info.BioURL)),
	}
	if resolved != nil {
		payload["resolvedFromIndex"] = compactMember(*resolved, false)
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = bio
	}
	emit(payload)
	return nil
}

func runCommitteesList(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchXMLWithParams(committeesURL, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	index, err := parseCommitteeIndex(raw)
	if err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	items := compactCommittees(index.Committees, limit, flagBool(parsed, "include-raw"))
	payload := envelope("committees list", requestURL, map[string]any{"limit": limit})
	payload["summary"] = map[string]any{"totalCommittees": len(index.Committees), "returned": len(items), "documentStand": index.DocumentStand}
	payload["items"] = items
	payload["sources"] = sources("Bundestag committee XML index", committeesURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{"bundestagctl committees search --term Arbeit --limit 5"}
	emit(payload)
	return nil
}

func runCommitteesSearch(argv []string) error {
	parsed := parseArgs(argv)
	term := firstNonEmpty(parsed.flags["term"], parsed.flags["q"], parsed.flags["name"], strings.Join(parsed.positionals, " "))
	if term == "" {
		return cliError{2, "missing_term", "committees search requires --term"}
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	raw, requestURL, err := fetchXMLWithParams(committeesURL, nil)
	if err != nil {
		return err
	}
	index, err := parseCommitteeIndex(raw)
	if err != nil {
		return err
	}
	var matches []committeeListItem
	needle := strings.ToLower(term)
	for _, committee := range index.Committees {
		if strings.Contains(strings.ToLower(committeeSearchText(committee)), needle) {
			matches = append(matches, committee)
		}
	}
	payload := envelope("committees search", requestURL, map[string]any{"term": term, "limit": limit})
	payload["summary"] = map[string]any{"term": term, "matches": len(matches), "returned": minInt(limit, len(matches)), "searchedCommittees": len(index.Committees)}
	payload["items"] = compactCommittees(matches, limit, flagBool(parsed, "include-raw"))
	payload["sources"] = sources("Bundestag committee XML index", committeesURL, "api_endpoint")
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForCommittees(matches)
	emit(payload)
	return nil
}

func runCommitteeGet(argv []string) error {
	parsed := parseArgs(argv)
	id, err := requiredFlag(parsed, "id", "committees get requires --id")
	if err != nil {
		return err
	}
	raw, requestURL, err := fetchXMLWithParams(fmt.Sprintf(committeeURLPattern, url.PathEscape(id)), parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	detail, err := parseCommitteeDetail(raw)
	if err != nil {
		return err
	}
	memberLimit := limitFlagName(parsed, "member-limit", defaultLimit, safeLimit)
	newsLimit := limitFlagName(parsed, "news-limit", 5, 50)
	payload := envelope("committees get", requestURL, map[string]any{"id": id, "memberLimit": memberLimit, "newsLimit": newsLimit})
	payload["summary"] = compactCommitteeDetail(detail, memberLimit, newsLimit, parsed.flags["grep"])
	payload["items"] = []any{committeeEvidence(detail, memberLimit, newsLimit, parsed.flags["grep"])}
	payload["sources"] = sourcesForCommittee(detail, requestURL)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{fmt.Sprintf("bundestagctl committees dossier --id %s --member-limit 5", id)}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = detail
	}
	emit(payload)
	return nil
}

func runCommitteeDossier(argv []string) error {
	return runCommitteeGet(argv)
}

func runPlenumSpeaker(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchXMLWithParams(speakerURL, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var feed speakerXML
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return err
	}
	payload := envelope("plenum speaker", requestURL, nil)
	payload["summary"] = map[string]any{"live": feed.Live, "topicNumber": feed.TopicNumber, "speakerCount": len(feed.Speakers)}
	payload["items"] = feed.Speakers
	payload["sources"] = sources("Bundestag current speaker XML", speakerURL, "api_endpoint")
	payload["warnings"] = append(defaultWarnings(), "The current speaker feed can be empty when no plenary sitting is live.")
	payload["nextActions"] = []string{"bundestagctl plenum conferences --limit 2 --item-limit 5"}
	if flagBool(parsed, "include-raw") {
		payload["rawXml"] = string(raw)
	}
	emit(payload)
	return nil
}

func runPlenumConferences(argv []string) error {
	parsed := parseArgs(argv)
	raw, requestURL, err := fetchXMLWithParams(conferencesURL, parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var feed conferencesXML
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return err
	}
	limit := limitFlag(parsed, defaultLimit, safeLimit)
	itemLimit := limitFlagName(parsed, "item-limit", defaultLimit, safeLimit)
	items := compactConferenceDays(feed.Days, limit, itemLimit)
	payload := envelope("plenum conferences", requestURL, map[string]any{"limit": limit, "itemLimit": itemLimit})
	payload["summary"] = map[string]any{"totalDays": len(feed.Days), "returned": len(items)}
	payload["items"] = items
	payload["sources"] = sources("Bundestag plenary conference XML", conferencesURL, "api_endpoint")
	payload["warnings"] = append(defaultWarnings(), "Agenda article IDs point to Bundestag article XML/page records, not full plenary protocols.")
	payload["nextActions"] = nextActionsForConferenceDays(feed.Days)
	if flagBool(parsed, "include-raw") {
		payload["rawXml"] = string(raw)
	}
	emit(payload)
	return nil
}

func runArticleGet(argv []string) error {
	parsed := parseArgs(argv)
	id := firstNonEmpty(parsed.flags["article-id"], parsed.flags["id"], firstPosition(parsed))
	if id == "" && parsed.flags["url"] != "" {
		id = articleIDFromURL(parsed.flags["url"])
	}
	if id == "" {
		return cliError{2, "missing_article_id", "article get requires --article-id or --url"}
	}
	raw, requestURL, err := fetchXMLWithParams(fmt.Sprintf(articleURLPattern, url.PathEscape(id)), parsed.params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	article, err := parseArticle(raw)
	if err != nil {
		return err
	}
	payload := envelope("article get", requestURL, map[string]any{"articleId": id, "grep": parsed.flags["grep"]})
	payload["summary"] = compactArticle(article, parsed.flags["grep"])
	payload["items"] = []any{articleEvidence(article, parsed.flags["grep"])}
	payload["sources"] = sourcesForArticle(article, requestURL)
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = []string{}
	if article.SourceURL != "" {
		payload["nextActions"] = append(payload["nextActions"].([]string), fmt.Sprintf("bundestagctl article page --url %q", article.SourceURL))
	}
	if flagBool(parsed, "include-raw") {
		payload["raw"] = article
	}
	emit(payload)
	return nil
}

func runArticlePage(argv []string) error {
	parsed := parseArgs(argv)
	sourceURL := parsed.flags["url"]
	id := firstNonEmpty(parsed.flags["article-id"], parsed.flags["id"])
	if sourceURL == "" && id != "" {
		articleRaw, _, err := fetchXMLWithParams(fmt.Sprintf(articleURLPattern, url.PathEscape(id)), nil)
		if err != nil {
			return err
		}
		article, err := parseArticle(articleRaw)
		if err != nil {
			return err
		}
		sourceURL = article.SourceURL
	}
	if sourceURL == "" {
		return cliError{2, "missing_url", "article page requires --url or --article-id"}
	}
	if !strings.HasPrefix(sourceURL, baseURL) {
		return cliError{2, "unsafe_url", "article page only accepts www.bundestag.de URLs"}
	}
	code, contentType, body, err := fetchRaw(sourceURL)
	if err != nil {
		return err
	}
	text := stripHTML(string(body))
	title := htmlTitle(string(body))
	snippets := grepSnippets(text, parsed.flags["grep"], 5, 650)
	payload := envelope("article page", sourceURL, map[string]any{"url": sourceURL, "grep": parsed.flags["grep"]})
	payload["summary"] = map[string]any{"url": sourceURL, "statusCode": code, "contentType": contentType, "title": title, "textLength": len(text), "snippetCount": len(snippets)}
	payload["items"] = snippets
	payload["sources"] = sources("Bundestag public article page", sourceURL, "public_page")
	payload["warnings"] = append(defaultWarnings(), "Public HTML page extraction is best-effort; use article get for structured XML metadata when possible.")
	payload["nextActions"] = []string{}
	if articleID := articleIDFromURL(sourceURL); articleID != "" {
		payload["nextActions"] = []string{fmt.Sprintf("bundestagctl article get --article-id %s", articleID)}
	}
	emit(payload)
	return nil
}

func runVideoFeed(argv []string) error {
	parsed := parseArgs(argv)
	contentID := firstNonEmpty(parsed.flags["content-id"], parsed.flags["contentid"], parsed.params.Get("contentId"), parsed.params.Get("contentid"))
	params := parsed.params
	if contentID != "" && params.Get("contentId") == "" {
		params.Set("contentId", contentID)
	}
	raw, requestURL, err := fetchXMLWithParams(videoURL, params)
	if err != nil {
		return err
	}
	if flagBool(parsed, "raw") {
		fmt.Print(string(raw))
		return nil
	}
	var feed videoFeedXML
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return err
	}
	payload := envelope("video feed", requestURL, map[string]any{"contentId": contentID})
	payload["summary"] = map[string]any{"contentId": contentID, "groups": len(feed.Groups), "streamCount": countVideoStreams(feed.Groups)}
	payload["items"] = compactVideoGroups(feed.Groups)
	payload["sources"] = []map[string]any{
		{"title": "Bundestag WebTV feed", "url": requestURL, "kind": "api_endpoint"},
		{"title": "Bundestag audio/video terms", "url": mediaTermsURL, "kind": "terms"},
	}
	payload["warnings"] = append(defaultWarnings(), "Video/audio material is governed by Bundestag media terms; cite Deutscher Bundestag and avoid misleading edits.")
	payload["nextActions"] = []string{"bundestagctl plenum conferences --limit 2 --item-limit 5"}
	if flagBool(parsed, "include-raw") {
		payload["rawXml"] = string(raw)
	}
	emit(payload)
	return nil
}

func runSource(argv []string) error {
	parsed := parseArgs(argv)
	sourceURL := firstNonEmpty(parsed.flags["url"], firstPosition(parsed))
	if sourceURL == "" {
		return cliError{2, "missing_url", "source requires --url"}
	}
	payload := envelope("source", sourceURL, map[string]any{"url": sourceURL})
	payload["summary"] = map[string]any{"url": sourceURL, "kind": sourceKind(sourceURL), "citation": "Deutscher Bundestag, " + sourceURL}
	payload["sources"] = sources("Bundestag source", sourceURL, sourceKind(sourceURL))
	payload["warnings"] = defaultWarnings()
	payload["nextActions"] = nextActionsForSourceURL(sourceURL)
	emit(payload)
	return nil
}

func parseMemberIndex(raw []byte) (memberIndexXML, error) {
	var index memberIndexXML
	err := xml.Unmarshal(raw, &index)
	return index, err
}

func parseMemberBiography(raw []byte) (memberBiographyXML, error) {
	var bio memberBiographyXML
	err := xml.Unmarshal(raw, &bio)
	return bio, err
}

func parseCommitteeIndex(raw []byte) (committeeIndexXML, error) {
	var index committeeIndexXML
	err := xml.Unmarshal(raw, &index)
	return index, err
}

func parseCommitteeDetail(raw []byte) (committeeDetailXML, error) {
	var detail committeeDetailXML
	err := xml.Unmarshal(raw, &detail)
	return detail, err
}

func parseArticle(raw []byte) (articleXML, error) {
	var article articleXML
	err := xml.Unmarshal(raw, &article)
	return article, err
}

func resolveMemberByName(term string) (memberListItem, error) {
	raw, _, err := fetchXMLWithParams(membersURL, nil)
	if err != nil {
		return memberListItem{}, err
	}
	index, err := parseMemberIndex(raw)
	if err != nil {
		return memberListItem{}, err
	}
	needle := strings.ToLower(term)
	for _, member := range index.Members {
		if strings.Contains(strings.ToLower(member.Name.Text), needle) {
			return member, nil
		}
	}
	return memberListItem{}, cliError{2, "member_not_found", "member not found: " + term}
}

func compactMembers(members []memberListItem, limit int, includeRaw bool) []map[string]any {
	out := []map[string]any{}
	for _, member := range members[:minInt(limit, len(members))] {
		out = append(out, compactMember(member, includeRaw))
	}
	return out
}

func compactMember(member memberListItem, includeRaw bool) map[string]any {
	item := map[string]any{
		"id":           member.ID.Text,
		"status":       firstNonEmpty(member.ID.Status, member.Name.Status),
		"name":         member.Name.Text,
		"fraction":     member.Fraction,
		"state":        member.State,
		"constituency": compactConstituency(member.Constituency),
		"electionType": member.ElectionType,
		"bioUrl":       member.BioURL,
		"infoXmlUrl":   member.InfoXMLURL,
		"lastChanged":  member.LastChanged,
		"sources": []map[string]any{
			{"title": "Bundestag member profile", "url": member.BioURL, "kind": "public_profile"},
			{"title": "Bundestag member biography XML", "url": member.InfoXMLURL, "kind": "api_endpoint"},
		},
		"nextActions": []string{
			fmt.Sprintf("bundestagctl members dossier --id %s", member.ID.Text),
			fmt.Sprintf("bundestagctl members biography --id %s --raw", member.ID.Text),
		},
	}
	if includeRaw {
		item["raw"] = member
	}
	return item
}

func compactConstituency(c constituency) map[string]any {
	return map[string]any{"number": c.Number, "name": c.Name, "url": c.URL}
}

func compactBiography(bio memberBiographyXML, grep string) map[string]any {
	name := fullName(bio.Info)
	return map[string]any{
		"id":                 bio.Info.ID.Text,
		"status":             bio.Info.ID.Status,
		"name":               name,
		"party":              bio.Info.Party,
		"fraction":           bio.Info.Fraction,
		"state":              bio.Info.State,
		"profession":         stripSpace(bio.Info.Profession),
		"birthDate":          bio.Info.BirthDate,
		"constituency":       compactConstituency(bio.Info.Constituency),
		"electionType":       bio.Info.ElectionType,
		"profileUrl":         firstNonEmpty(bio.Info.SourceURL, bio.Info.BioURL),
		"homepageUrl":        bio.Info.HomepageURL,
		"speechesUrl":        bio.Media.SpeechesURL,
		"speechesRss":        bio.Media.SpeechesRSS,
		"biographySnippets":  grepSnippets(stripHTML(bio.Info.BiographyHTML), grep, 3, 650),
		"disclosureSnippets": grepSnippets(stripHTML(bio.Info.DisclosureHTML), grep, 5, 650),
	}
}

func memberEvidence(bio memberBiographyXML, grep string) map[string]any {
	return map[string]any{
		"biography":   truncate(stripHTML(bio.Info.BiographyHTML), 1500),
		"disclosures": disclosureSections(bio.Info.DisclosureHTML, grep),
		"websites":    bio.Info.OtherWebsites,
		"media": map[string]any{
			"photoUrl":    bio.Media.Photo.URL,
			"photoSource": bio.Media.Photo.Copyright,
			"speechesUrl": bio.Media.SpeechesURL,
			"speechesRss": bio.Media.SpeechesRSS,
		},
	}
}

func disclosureSections(value string, grep string) []map[string]any {
	text := stripHTML(value)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	snippets := grepSnippets(text, grep, 8, 650)
	if len(snippets) == 0 && grep == "" {
		return []map[string]any{{"text": truncate(text, 1800)}}
	}
	return snippets
}

func compactCommittees(committees []committeeListItem, limit int, includeRaw bool) []map[string]any {
	out := []map[string]any{}
	for _, committee := range committees[:minInt(limit, len(committees))] {
		item := map[string]any{
			"id":           committee.ID,
			"name":         committee.Name,
			"shortName":    committee.ShortName,
			"teaser":       truncate(stripHTML(committee.TeaserHTML), 420),
			"detailXmlUrl": committee.DetailXMLURL,
			"lastChanged":  committee.LastChanged,
			"imageUrl":     committee.ImageURL,
			"imageSource":  committee.ImageCopyright,
			"sources":      sources("Bundestag committee XML", committee.DetailXMLURL, "api_endpoint"),
			"nextActions":  []string{fmt.Sprintf("bundestagctl committees dossier --id %s --member-limit 5", committee.ID)},
		}
		if includeRaw {
			item["raw"] = committee
		}
		out = append(out, item)
	}
	return out
}

func compactCommitteeDetail(detail committeeDetailXML, memberLimit int, newsLimit int, grep string) map[string]any {
	return map[string]any{
		"id":           detail.ID,
		"name":         detail.Name,
		"sourceUrl":    detail.SourceURL,
		"chairId":      detail.ChairID,
		"memberCount":  len(detail.Members),
		"newsCount":    len(detail.News),
		"taskSnippets": grepSnippets(stripHTML(detail.TaskHTML), grep, 3, 650),
		"contact":      stripHTML(detail.ContactHTML),
		"membersShown": minInt(memberLimit, len(detail.Members)),
		"newsShown":    minInt(newsLimit, len(detail.News)),
	}
}

func committeeEvidence(detail committeeDetailXML, memberLimit int, newsLimit int, grep string) map[string]any {
	return map[string]any{
		"task":    truncate(stripHTML(detail.TaskHTML), 1200),
		"contact": stripHTML(detail.ContactHTML),
		"members": compactCommitteeMembers(detail.Members, memberLimit),
		"news":    compactCommitteeNews(detail.News, newsLimit, grep),
	}
}

func compactCommitteeMembers(members []committeeMemberItem, limit int) []map[string]any {
	out := []map[string]any{}
	for _, member := range members[:minInt(limit, len(members))] {
		out = append(out, map[string]any{
			"id":          member.ID.Text,
			"name":        member.Name.Text,
			"fraction":    member.Fraction,
			"state":       member.State,
			"role":        member.Role,
			"bioUrl":      member.BioURL,
			"infoXmlUrl":  member.InfoXMLURL,
			"lastChanged": member.LastChanged,
			"nextActions": []string{fmt.Sprintf("bundestagctl members dossier --id %s", member.ID.Text)},
		})
	}
	return out
}

func compactCommitteeNews(news []committeeNewsItem, limit int, grep string) []map[string]any {
	out := []map[string]any{}
	for _, item := range news[:minInt(limit, len(news))] {
		text := stripHTML(item.TeaserHTML)
		if grep != "" && !strings.Contains(strings.ToLower(text+" "+item.Title), strings.ToLower(grep)) {
			continue
		}
		out = append(out, map[string]any{
			"articleId":       item.ArticleID,
			"date":            item.Date,
			"title":           item.Title,
			"teaser":          truncate(text, 500),
			"detailsXml":      item.DetailsXML,
			"videoUrl":        item.VideoURL,
			"fields":          item.Fields,
			"changedDateTime": item.ChangedDateTime,
			"nextActions":     []string{fmt.Sprintf("bundestagctl article get --article-id %s", item.ArticleID)},
		})
	}
	return out
}

func compactConferenceDays(days []conferenceDay, limit int, itemLimit int) []map[string]any {
	out := []map[string]any{}
	for _, day := range days[:minInt(limit, len(days))] {
		items := []map[string]any{}
		for _, item := range day.Items[:minInt(itemLimit, len(day.Items))] {
			items = append(items, map[string]any{
				"startTime":   item.StartTime,
				"endTime":     item.EndTime,
				"status":      item.Status,
				"title":       item.Title,
				"articleId":   item.ArticleID,
				"top":         item.Top,
				"nextActions": nextActionsForArticleID(item.ArticleID),
			})
		}
		out = append(out, map[string]any{
			"date":          day.Date,
			"active":        day.Active,
			"sessionNumber": day.SessionNumber,
			"name":          day.Name,
			"itemCount":     len(day.Items),
			"items":         items,
		})
	}
	return out
}

func compactArticle(article articleXML, grep string) map[string]any {
	text := stripHTML(article.TextHTML)
	return map[string]any{
		"articleId":       article.ArticleID,
		"date":            article.Date,
		"title":           article.Title,
		"sourceUrl":       article.SourceURL,
		"fields":          article.Fields,
		"changedDateTime": article.ChangedDateTime,
		"textLength":      len(text),
		"snippets":        grepSnippets(text, grep, 5, 650),
	}
}

func articleEvidence(article articleXML, grep string) map[string]any {
	text := stripHTML(article.TextHTML)
	return map[string]any{
		"text":         truncate(text, 1800),
		"snippets":     grepSnippets(text, grep, 5, 650),
		"imageUrl":     article.ImageURL,
		"imageSource":  article.ImageCopyright,
		"imageAltText": article.ImageAltText,
	}
}

func compactVideoGroups(groups []videoGroup) []map[string]any {
	out := []map[string]any{}
	for _, group := range groups {
		streams := []map[string]any{}
		for _, stream := range group.Streams {
			streams = append(streams, map[string]any{"bandwidth": stream.Bandwidth, "href": stream.Href})
		}
		out = append(out, map[string]any{"type": group.Type, "streams": streams})
	}
	return out
}

func countVideoStreams(groups []videoGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Streams)
	}
	return total
}

func sourcesForMemberBiography(bio memberBiographyXML, requestURL string) []map[string]any {
	out := []map[string]any{
		{"title": "Bundestag member biography XML", "url": requestURL, "kind": "api_endpoint"},
	}
	if source := firstNonEmpty(bio.Info.SourceURL, bio.Info.BioURL); source != "" {
		out = append(out, map[string]any{"title": "Bundestag member profile", "url": source, "kind": "public_profile"})
	}
	if bio.Media.SpeechesURL != "" {
		out = append(out, map[string]any{"title": "Bundestag mediathek speeches filter", "url": bio.Media.SpeechesURL, "kind": "media_search"})
	}
	if bio.Media.SpeechesRSS != "" {
		out = append(out, map[string]any{"title": "Bundestag speeches RSS", "url": bio.Media.SpeechesRSS, "kind": "rss"})
	}
	return out
}

func sourcesForCommittee(detail committeeDetailXML, requestURL string) []map[string]any {
	out := []map[string]any{{"title": "Bundestag committee detail XML", "url": requestURL, "kind": "api_endpoint"}}
	if detail.SourceURL != "" {
		out = append(out, map[string]any{"title": "Bundestag committee page", "url": detail.SourceURL, "kind": "public_page"})
	}
	return out
}

func sourcesForArticle(article articleXML, requestURL string) []map[string]any {
	out := []map[string]any{{"title": "Bundestag article XML", "url": requestURL, "kind": "api_endpoint"}}
	if article.SourceURL != "" {
		out = append(out, map[string]any{"title": "Bundestag public article page", "url": article.SourceURL, "kind": "public_page"})
	}
	return out
}

func sources(title, sourceURL, kind string) []map[string]any {
	if sourceURL == "" {
		return nil
	}
	return []map[string]any{{"title": title, "url": sourceURL, "kind": kind}}
}

func defaultSources() []map[string]any {
	return []map[string]any{
		{"title": "Bundestag live XML OpenAPI wrapper", "url": openAPIURL, "kind": "openapi_reference"},
		{"title": "Deutscher Bundestag Open Data", "url": openDataURL, "kind": "official_context"},
		{"title": "Bundestag website terms/imprint", "url": imprintURL, "kind": "terms"},
		{"title": "Bundestag audio/video terms", "url": mediaTermsURL, "kind": "terms"},
		{"title": "Bundestag privacy policy", "url": privacyURL, "kind": "privacy"},
	}
}

func defaultWarnings() []string {
	return []string{
		"No exact public rate limit for these Bundestag XML feeds was found; use small limits and avoid repeated broad index pulls.",
		"This live/site XML surface is not the full parliamentary archive. Use dipctl for complete proceedings, printed papers, and plenary protocol research.",
		"Official Bundestag profile/disclosure data can include self-reported fields; preserve source URLs and timestamps in final citations.",
		"Website, image, and video materials may have separate usage terms; inspect the relevant source page/terms before republication.",
	}
}

func nextActionsForMembers(members []memberListItem) []string {
	var actions []string
	for _, member := range members[:minInt(3, len(members))] {
		actions = append(actions, fmt.Sprintf("bundestagctl members dossier --id %s", member.ID.Text))
	}
	if len(actions) == 0 {
		return []string{`bundestagctl members search --name "Amthor" --limit 3`}
	}
	return actions
}

func nextActionsForCommittees(committees []committeeListItem) []string {
	var actions []string
	for _, committee := range committees[:minInt(3, len(committees))] {
		actions = append(actions, fmt.Sprintf("bundestagctl committees dossier --id %s --member-limit 5", committee.ID))
	}
	if len(actions) == 0 {
		return []string{"bundestagctl committees search --term Arbeit --limit 5"}
	}
	return actions
}

func nextActionsForConferenceDays(days []conferenceDay) []string {
	var actions []string
	for _, day := range days {
		for _, item := range day.Items {
			if item.ArticleID != "" {
				actions = append(actions, fmt.Sprintf("bundestagctl article get --article-id %s", item.ArticleID))
			}
			if len(actions) >= 3 {
				return actions
			}
		}
	}
	return []string{"bundestagctl plenum speaker"}
}

func nextActionsForArticleID(articleID string) []string {
	if articleID == "" {
		return nil
	}
	return []string{fmt.Sprintf("bundestagctl article get --article-id %s", articleID)}
}

func nextActionsForSourceURL(sourceURL string) []string {
	if strings.Contains(sourceURL, "/xml/v2/mdb/biografien/") {
		return []string{fmt.Sprintf("bundestagctl members biography --id %s", articleIDFromURL(sourceURL))}
	}
	if id := articleIDFromURL(sourceURL); id != "" {
		return []string{fmt.Sprintf("bundestagctl article get --article-id %s", id), fmt.Sprintf("bundestagctl article page --url %q", sourceURL)}
	}
	return []string{}
}

func memberSearchText(member memberListItem) string {
	return strings.Join([]string{
		member.ID.Text,
		member.Name.Text,
		member.Fraction,
		member.State,
		member.Constituency.Number,
		member.Constituency.Name,
		member.ElectionType,
		member.BioURL,
	}, " ")
}

func committeeSearchText(committee committeeListItem) string {
	return strings.Join([]string{
		committee.ID,
		committee.Name,
		committee.ShortName,
		stripHTML(committee.TeaserHTML),
		committee.DetailXMLURL,
	}, " ")
}

func fullName(info memberBioInfo) string {
	return stripSpace(strings.Join([]string{info.AcademicTitle, info.FirstName, info.LastName}, " "))
}

func fetchXMLWithParams(base string, params url.Values) ([]byte, string, error) {
	requestURL := withParams(base, params)
	status, _, body, err := fetchRaw(requestURL)
	if err != nil {
		return nil, requestURL, err
	}
	if status < 200 || status >= 300 {
		return nil, requestURL, httpError{status, string(body), requestURL}
	}
	return body, requestURL, nil
}

func fetchRaw(requestURL string) (int, string, []byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("User-Agent", "democracy-researcher/bundestagctl-2.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp.StatusCode, resp.Header.Get("Content-Type"), body, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, resp.Header.Get("Content-Type"), body, httpError{resp.StatusCode, string(body), requestURL}
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func withParams(base string, params url.Values) string {
	if len(params) == 0 {
		return base
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func parseArgs(args []string) parsedArgs {
	parsed := parsedArgs{flags: map[string]string{}, params: url.Values{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			parsed.positionals = append(parsed.positionals, arg)
			continue
		}
		keyValue := strings.TrimPrefix(arg, "--")
		key := keyValue
		value := "true"
		if idx := strings.Index(keyValue, "="); idx >= 0 {
			key = keyValue[:idx]
			value = keyValue[idx+1:]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "param" {
			if idx := strings.Index(value, "="); idx > 0 {
				parsed.params.Add(value[:idx], value[idx+1:])
			}
			continue
		}
		parsed.flags[key] = value
	}
	return parsed
}

func requiredFlag(parsed parsedArgs, key string, message string) (string, error) {
	value := firstNonEmpty(parsed.flags[key], firstPosition(parsed))
	if value == "" {
		return "", cliError{2, "missing_" + strings.ReplaceAll(key, "-", "_"), message}
	}
	return value, nil
}

func limitFlag(parsed parsedArgs, fallback, maxValue int) int {
	return limitFlagName(parsed, "limit", fallback, maxValue)
}

func limitFlagName(parsed parsedArgs, name string, fallback, maxValue int) int {
	value := fallback
	if raw := firstNonEmpty(parsed.flags[name]); raw != "" {
		parsedValue, err := strconv.Atoi(raw)
		if err == nil && parsedValue > 0 {
			value = parsedValue
		}
	}
	if value > maxValue && !flagBool(parsed, "allow-large-output") {
		fail(2, "limit_exceeds_safe_max", fmt.Sprintf("%s %d exceeds safe max %d; pass --allow-large-output to override", name, value, maxValue))
	}
	return value
}

func flagBool(parsed parsedArgs, key string) bool {
	value := strings.ToLower(parsed.flags[key])
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func envelope(command, requestURL string, request any) map[string]any {
	return map[string]any{
		"status":      "ok",
		"tool":        appName,
		"command":     command,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"request":     map[string]any{"method": "GET", "url": requestURL, "params": request},
		"summary":     map[string]any{},
		"items":       []any{},
		"sources":     []any{},
		"warnings":    []string{},
		"nextActions": []string{},
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func fail(exitCode int, code, message string) {
	emit(map[string]any{
		"status":      "error",
		"tool":        appName,
		"retrievedAt": time.Now().UTC().Format(time.RFC3339),
		"error":       map[string]any{"code": code, "message": message},
	})
	os.Exit(exitCode)
}

func isHelp(value string) bool {
	return value == "--help" || value == "-h" || value == "help"
}

func match(args []string, expected ...string) bool {
	if len(args) < len(expected) {
		return false
	}
	for index, value := range expected {
		if args[index] != value {
			return false
		}
	}
	return true
}

func firstPosition(parsed parsedArgs) string {
	if len(parsed.positionals) == 0 {
		return ""
	}
	return parsed.positionals[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func sourceKind(sourceURL string) string {
	switch {
	case strings.Contains(sourceURL, "/xml/"):
		return "api_endpoint"
	case strings.Contains(sourceURL, "webtv.bundestag.de"):
		return "media_feed"
	case strings.Contains(sourceURL, "/mediathek"):
		return "media_page"
	case strings.Contains(sourceURL, "/abgeordnete/"):
		return "public_profile"
	default:
		return "public_page"
	}
}

func articleIDFromURL(value string) string {
	re := regexp.MustCompile(`(\d{5,})(?:\.xml)?/?(?:$|[?#])`)
	matches := re.FindStringSubmatch(value)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

var (
	scriptStylePattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTagPattern     = regexp.MustCompile(`<[^>]+>`)
	spacePattern       = regexp.MustCompile(`\s+`)
	sentencePattern    = regexp.MustCompile(`(?m)([^.!?。！？]*[^.!?。！？]*[.!?。！？])`)
	titlePattern       = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

func stripHTML(value string) string {
	value = scriptStylePattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.ReplaceAll(value, "<br/>", " ")
	value = strings.ReplaceAll(value, "<br>", " ")
	value = htmlTagPattern.ReplaceAllString(value, " ")
	return stripSpace(value)
}

func stripSpace(value string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func truncate(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}

func grepSnippets(text string, grep string, limit int, maxLen int) []map[string]any {
	text = stripSpace(text)
	if text == "" {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(grep))
	if needle == "" {
		return []map[string]any{{"text": truncate(text, maxLen)}}
	}
	lower := strings.ToLower(text)
	var out []map[string]any
	seen := map[string]bool{}
	searchFrom := 0
	for len(out) < limit {
		idx := strings.Index(lower[searchFrom:], needle)
		if idx < 0 {
			break
		}
		idx += searchFrom
		start := idx - maxLen/2
		if start < 0 {
			start = 0
		}
		end := start + maxLen
		if end > len(text) {
			end = len(text)
		}
		snippet := strings.TrimSpace(text[start:end])
		key := snippet
		if len(key) > 180 {
			key = key[:180]
		}
		if !seen[key] {
			out = append(out, map[string]any{
				"grep": grep,
				"text": snippet,
			})
			seen[key] = true
		}
		searchFrom = idx + len(needle)
	}
	return out
}

func htmlTitle(value string) string {
	matches := titlePattern.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return stripHTML(matches[1])
}

func decodeXMLMap(raw []byte) map[string]any {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	stack := []map[string]any{}
	var root map[string]any
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch typed := token.(type) {
		case xml.StartElement:
			node := map[string]any{"name": typed.Name.Local, "attrs": map[string]string{}, "children": []any{}, "text": ""}
			for _, attr := range typed.Attr {
				node["attrs"].(map[string]string)[attr.Name.Local] = attr.Value
			}
			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent["children"] = append(parent["children"].([]any), node)
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				node := stack[len(stack)-1]
				node["text"] = strings.TrimSpace(fmt.Sprint(node["text"]) + string(typed))
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return map[string]any{}
	}
	return root
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
