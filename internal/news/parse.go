package news

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// --- RSS 2.0 ---

type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Items   []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	// dc:date — matched by local name regardless of namespace.
	DCDate string `xml:"date"`
}

// --- Atom ---

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// parseFeed parses RSS 2.0 or Atom XML into NewsItems, stamping each with the
// given source name. Only title, link, and publish time are extracted — never
// the body or description. The format is detected by attempting RSS first
// (whose root <rss> tag makes Unmarshal fail on an Atom document) then Atom.
func parseFeed(data []byte, source string) ([]NewsItem, error) {
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err == nil {
		return rssToItems(rss, source), nil
	}
	var atom atomFeed
	if err := xml.Unmarshal(data, &atom); err == nil {
		return atomToItems(atom, source), nil
	}
	return nil, fmt.Errorf("news: %s: unrecognized or malformed feed (not RSS 2.0 or Atom)", source)
}

func rssToItems(f rssFeed, source string) []NewsItem {
	items := make([]NewsItem, 0, len(f.Items))
	for _, it := range f.Items {
		date := it.PubDate
		if strings.TrimSpace(date) == "" {
			date = it.DCDate
		}
		items = append(items, NewsItem{
			Title:     strings.TrimSpace(it.Title),
			Source:    source,
			Link:      strings.TrimSpace(it.Link),
			Published: parseTime(date),
		})
	}
	return items
}

func atomToItems(f atomFeed, source string) []NewsItem {
	items := make([]NewsItem, 0, len(f.Entries))
	for _, e := range f.Entries {
		date := e.Published
		if strings.TrimSpace(date) == "" {
			date = e.Updated
		}
		items = append(items, NewsItem{
			Title:     strings.TrimSpace(e.Title),
			Source:    source,
			Link:      atomLinkHref(e.Links),
			Published: parseTime(date),
		})
	}
	return items
}

// atomLinkHref picks the entry's canonical link: prefer rel="alternate" (or an
// unset rel), otherwise the first link with an href.
func atomLinkHref(links []atomLink) string {
	for _, l := range links {
		if l.Href != "" && (l.Rel == "alternate" || l.Rel == "") {
			return strings.TrimSpace(l.Href)
		}
	}
	for _, l := range links {
		if l.Href != "" {
			return strings.TrimSpace(l.Href)
		}
	}
	return ""
}

// dateLayouts are tried in order. A value that matches none yields the zero
// time, which sorts the item to the end of the list.
var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	time.RFC822Z,
	time.RFC822,
}

// parseTime is tolerant of the common feed date formats. On failure it returns
// the zero time rather than an error.
func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
