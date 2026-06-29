package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Monero-Team/monero-team/internal/news"
)

// newsStoreWith builds a store seeded with one source's items.
func newsStoreWith(items ...news.NewsItem) *news.Store {
	s := news.NewStore(0)
	s.Merge("Test Source", items)
	return s
}

func TestNewsFeedRendersItems(t *testing.T) {
	store := newsStoreWith(
		news.NewsItem{Title: "Lead headline", Source: "Monero Observer", Link: "https://news.test/lead", Published: time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)},
		news.NewsItem{Title: "Second headline", Source: "Monero Project", Link: "https://news.test/second", Published: time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)},
	)
	h := newTestServerWithNews(t, store)

	rec := get(t, h, "/news")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /news: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Lead headline", "Second headline",
		"Monero Observer", "Monero Project",
		"28 Jun 2026", `datetime="2026-06-28"`, // date label + ISO attribute
		`href="https://news.test/lead" rel="noopener noreferrer"`,
		`href="https://news.test/second" rel="noopener noreferrer"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/news missing %q", want)
		}
	}

	// First item is the lead (larger), the rest are rows.
	if !strings.Contains(body, "news-lead") {
		t.Error("/news did not render a lead item")
	}
	lead, ok := between(body, `class="news-lead"`, "</article>")
	if !ok || !strings.Contains(lead, "Lead headline") {
		t.Error("first (newest) item should be the lead")
	}
	if !strings.Contains(body, "news-row") {
		t.Error("/news did not render any standard rows")
	}

	// No images, no script (no-JS), no cookies, CSP intact.
	if strings.Contains(strings.ToLower(body), "<img") {
		t.Error("/news must not contain <img>")
	}
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("/news must not contain <script>")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("CSP missing script-src 'none': %q", csp)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("/news set cookies")
	}
}

func TestNewsFeedEmptyState(t *testing.T) {
	h := newTestServerWithNews(t, news.NewStore(0))
	rec := get(t, h, "/news")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /news (empty): got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No news yet") {
		t.Error("empty store should render the empty state")
	}
	if strings.Contains(body, "news-lead") {
		t.Error("empty store should not render a lead")
	}
}

func TestNewsFeedZeroDateRenders(t *testing.T) {
	store := newsStoreWith(
		news.NewsItem{Title: "Undated headline", Source: "Mystery", Link: "https://news.test/undated"}, // zero Published
	)
	h := newTestServerWithNews(t, store)
	rec := get(t, h, "/news")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /news: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Undated headline") {
		t.Error("undated item should still render")
	}
	// An undated item renders an empty datetime attribute rather than a bogus date.
	if !strings.Contains(body, `datetime=""`) {
		t.Error("zero publish time should yield an empty datetime")
	}
}

func TestNewsReplacesComingSoon(t *testing.T) {
	h := newTestServer(t)

	newsBody := get(t, h, "/news").Body.String()
	if strings.Contains(newsBody, "Coming soon") {
		t.Error("/news still shows the coming-soon skeleton")
	}
	if !strings.Contains(newsBody, `class="news__title"`) {
		t.Error("/news does not render the feed heading")
	}

	digest := get(t, h, "/digest").Body.String()
	if !strings.Contains(digest, "Coming soon") {
		t.Error("/digest should still show the coming-soon skeleton")
	}
}

func TestBuildNewsViewMapping(t *testing.T) {
	v := buildNewsView([]news.NewsItem{
		{Title: "T", Source: "S", Link: "L", Published: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Title: "U", Source: "S", Link: "M"}, // zero time
	})
	if v.Active != "news" {
		t.Errorf("Active = %q, want news", v.Active)
	}
	if v.Items[0] != (newsRow{Source: "S", Date: "2026-01-02", DateLabel: "2 Jan 2026", Headline: "T", URL: "L"}) {
		t.Errorf("row[0] = %+v", v.Items[0])
	}
	if v.Items[1].Date != "" || v.Items[1].DateLabel != "" {
		t.Errorf("zero-time row should have empty date fields: %+v", v.Items[1])
	}
}
