package web

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	cstrings "github.com/Monero-Team/monero-team/internal/content/strings"
	"github.com/Monero-Team/monero-team/internal/news"
)

// newTestServer builds the application handler (with an empty news store) or
// fails the test.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return newTestServerWithNews(t, news.NewStore(0))
}

// newTestServerWithNews builds the handler backed by a specific news store.
func newTestServerWithNews(t *testing.T, store *news.Store) http.Handler {
	t.Helper()
	h, err := NewHandler(store)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

// get performs an in-process GET and returns the recorder.
func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// allRoutes lists every HTML route that must return 200.
func allRoutes() []string {
	paths := []string{"/"}
	for _, s := range cstrings.Nav {
		paths = append(paths, s.Path)
	}
	for _, s := range cstrings.Utility {
		paths = append(paths, s.Path)
	}
	return paths
}

func TestRoutesReturn200(t *testing.T) {
	h := newTestServer(t)
	for _, path := range append(allRoutes(), "/healthz") {
		rec := get(t, h, path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: got %d, want 200", path, rec.Code)
		}
	}
}

func TestStaticAssetsReturn200(t *testing.T) {
	h := newTestServer(t)
	assets := []struct {
		path        string
		contentType string
	}{
		// app.css is requested via its cache-busting URL (the form templates emit).
		{assetURL("app.css"), "text/css; charset=utf-8"},
		{"/static/fonts/inter-tight-400.woff2", "font/woff2"},
		{"/static/fonts/jetbrains-mono-500.woff2", "font/woff2"},
	}
	for _, a := range assets {
		rec := get(t, h, a.path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: got %d, want 200", a.path, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); ct != a.contentType {
			t.Errorf("GET %s: Content-Type %q, want %q", a.path, ct, a.contentType)
		}
		if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Errorf("GET %s: Cache-Control not immutable, got %q", a.path, cc)
		}
	}
}

func TestNavbarHasExactlyFiveSections(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/").Body.String()

	nav, ok := between(body, `<ul class="nav-links">`, "</ul>")
	if !ok {
		t.Fatal("could not locate navbar list in rendered HTML")
	}
	if n := strings.Count(nav, "<li>"); n != 5 {
		t.Errorf("navbar has %d sections, want exactly 5", n)
	}
	// The five canonical sections must each be present.
	for _, s := range cstrings.Nav {
		if !strings.Contains(nav, `href="`+s.Path+`"`) {
			t.Errorf("navbar missing section %q (%s)", s.Label, s.Path)
		}
	}
}

func TestNoCookiesAnywhere(t *testing.T) {
	h := newTestServer(t)
	for _, path := range append(allRoutes(), "/healthz", "/static/app.css") {
		rec := get(t, h, path)
		if got := rec.Header().Values("Set-Cookie"); len(got) != 0 {
			t.Errorf("GET %s set cookies: %v", path, got)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Errorf("GET %s set cookies", path)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/")
	want := map[string]string{
		"Content-Security-Policy": "script-src 'none'",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"Permissions-Policy":      "geolocation=()",
	}
	for header, substr := range want {
		if v := rec.Header().Get(header); !strings.Contains(v, substr) {
			t.Errorf("%s = %q, want to contain %q", header, v, substr)
		}
	}
}

// externalLoader matches markup that would make the browser *load* a resource
// from an external origin. It mirrors scripts/check-no-external-origins.sh:
// outbound navigation (<a href="https://…">) is allowed — the directory links
// out by design — but stylesheets, scripts, images, and CSS url()s must be
// same-origin.
var externalLoader = regexp.MustCompile(`(?i)(<link[^>]+href="https?://|<script[^>]+src="https?://|<img[^>]+src="https?://|url\(\s*["']?https?://)`)

func TestNoExternalOrigins(t *testing.T) {
	h := newTestServer(t)
	for _, path := range allRoutes() {
		body := get(t, h, path).Body.String()
		if loc := externalLoader.FindString(body); loc != "" {
			t.Errorf("GET %s: rendered HTML loads an external resource: %q", path, loc)
		}
	}
}

func TestUnknownPathReturns404(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /does-not-exist: got %d, want 404", rec.Code)
	}
}

func TestActiveNavResolvedServerSide(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir")
	body := rec.Body.String()
	if !strings.Contains(body, `href="/dir" aria-current="page"`) {
		t.Error("active section /dir not marked aria-current in navbar")
	}
}

// between returns the substring of s strictly between the first occurrence of
// start and the first subsequent occurrence of end.
func between(s, start, end string) (string, bool) {
	i := strings.Index(s, start)
	if i < 0 {
		return "", false
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j < 0 {
		return "", false
	}
	return s[i : i+j], true
}
