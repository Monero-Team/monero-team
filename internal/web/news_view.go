package web

import (
	"net/http"

	"github.com/Monero-Team/monero-team/internal/news"
)

// newsPath is the canonical path of the news feed section.
const newsPath = "/news"

// newsRow is the presentation model consumed by the news-feed partial. Field
// names match the partial exactly; all formatting happens here so the template
// stays logic-free.
type newsRow struct {
	Source    string
	Date      string // ISO-8601 (for <time datetime>); "" when unknown
	DateLabel string // human-readable; "" when unknown
	Headline  string
	URL       string
}

// newsView is the page model passed to the news-feed partial.
type newsView struct {
	Items  []newsRow
	Active string
}

// buildNewsView maps store items (already newest-first) into presentation rows.
// A zero publish time yields empty Date/DateLabel so the row renders without a
// timestamp rather than showing a bogus date.
func buildNewsView(items []news.NewsItem) newsView {
	rows := make([]newsRow, 0, len(items))
	for _, it := range items {
		var date, label string
		if !it.Published.IsZero() {
			date = it.Published.Format("2006-01-02")
			label = it.Published.Format("2 Jan 2006")
		}
		rows = append(rows, newsRow{
			Source:    it.Source,
			Date:      date,
			DateLabel: label,
			Headline:  it.Title,
			URL:       it.Link,
		})
	}
	return newsView{Items: rows, Active: "news"}
}

// newsFeed renders the news feed from the latest stored items. The store is
// populated by the background collector; an empty store renders the calm
// empty state, never an error.
func (h *handler) newsFeed(w http.ResponseWriter, r *http.Request) {
	sec := h.sections[newsPath]
	v := newView(sec.Path)
	v.Section = sec
	var items []news.NewsItem
	if h.news != nil {
		items = h.news.Latest(50)
	}
	v.News = buildNewsView(items)
	h.tmpl.render(w, "news", v)
}
