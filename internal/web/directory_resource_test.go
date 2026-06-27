package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Monero-Team/monero-team/internal/directory"
)

func TestResourcePageOK(t *testing.T) {
	h := newTestServer(t)
	store := loadSeed(t)
	r, ok := store.BySlug("cake-wallet")
	if !ok {
		t.Fatal("seed slug cake-wallet not found")
	}

	rec := get(t, h, "/dir/"+r.Slug)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /dir/%s: got %d, want 200", r.Slug, rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		r.Name,
		statusLabel(r.Status), // status label
		r.Description,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("resource page missing %q", want)
		}
	}
	// Every present link must be shown (mono address = the URL text itself).
	for _, l := range resourceLinks(r) {
		if !strings.Contains(body, l.Label) {
			t.Errorf("resource page missing link label %q", l.Label)
		}
		if !strings.Contains(body, l.URL) {
			t.Errorf("resource page missing link URL %q", l.URL)
		}
		if !strings.Contains(body, `href="`+l.URL+`" rel="noopener noreferrer"`) {
			t.Errorf("link %q not rendered as external with rel", l.URL)
		}
	}
}

func TestResourcePageNotFound(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /dir/does-not-exist: got %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Resource not found") {
		t.Error("404 page missing 'Resource not found'")
	}
	if !strings.Contains(body, `href="/dir"`) {
		t.Error("404 page missing a link back to the directory")
	}
}

func TestResourcePageScamWarning(t *testing.T) {
	h := newTestServer(t)
	store := loadSeed(t)
	var scam *directory.Resource
	for _, r := range store.All() {
		if r.Status == directory.StatusScam {
			scam = r
			break
		}
	}
	if scam == nil {
		t.Fatal("no scam entry in seed dataset")
	}
	body := get(t, h, "/dir/"+scam.Slug).Body.String()
	if !strings.Contains(body, "Reported — do not use") {
		t.Error("scam resource page missing 'Reported — do not use'")
	}
	if !strings.Contains(body, "resource--scam") {
		t.Error("scam resource page missing the scam dimming class")
	}
}

func TestResourcePageQuestionable(t *testing.T) {
	h := newTestServer(t)
	store := loadSeed(t)
	var q *directory.Resource
	for _, r := range store.All() {
		if r.Status == directory.StatusQuestionable {
			q = r
			break
		}
	}
	if q == nil {
		t.Skip("no questionable entry in seed dataset")
	}
	body := get(t, h, "/dir/"+q.Slug).Body.String()
	if !strings.Contains(body, "resource--questionable") {
		t.Error("questionable resource page missing the dimming class")
	}
}

func TestDirIndexRedirect(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir/")
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("GET /dir/: got %d, want 301", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/dir" {
		t.Errorf("GET /dir/: Location = %q, want /dir", loc)
	}
}

// TestListNameLinksAreInternal verifies the Stage 5 row change: the resource
// name links to the internal detail page (no rel), while Open stays external.
func TestListNameLinksAreInternal(t *testing.T) {
	h := newTestServer(t)
	store := loadSeed(t)
	body := get(t, h, "/dir").Body.String()

	for _, r := range store.All() {
		nameLink := `<a href="/dir/` + r.Slug + `">` + r.Name + `</a>`
		if !strings.Contains(body, nameLink) {
			t.Errorf("name link not internal/exact for %s: want %q", r.Slug, nameLink)
		}
	}
	// The internal deep-link must not carry rel.
	if strings.Contains(body, `href="/dir/cake-wallet" rel=`) {
		t.Error("internal name link should not carry a rel attribute")
	}
	// Open remains an external link with rel.
	if !strings.Contains(body, `class="open-link"`) || !strings.Contains(body, `rel="noopener noreferrer"`) {
		t.Error("Open link must remain external with rel=\"noopener noreferrer\"")
	}
}

func TestResourcePageNoScriptAndHeaders(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir/cake-wallet")
	body := rec.Body.String()

	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("resource page must contain no <script>")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("CSP must keep script-src 'none', got %q", csp)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("resource page set cookies: %v", cookies)
	}
	// 404 page must also be script-free and cookie-free.
	nf := get(t, h, "/dir/nope")
	if strings.Contains(strings.ToLower(nf.Body.String()), "<script") {
		t.Error("404 page must contain no <script>")
	}
	if cookies := nf.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("404 page set cookies: %v", cookies)
	}
}
