package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Monero-Team/monero-team/internal/directory"
)

func TestSearchSingleTerm(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "q=cake")
	assertExactly(t, body, all, func(r *directory.Resource) bool {
		return parseSearch(url.Values{"q": {"cake"}}).match(r)
	})
	if !strings.Contains(body, "Cake Wallet") {
		t.Error("q=cake should list Cake Wallet")
	}
}

func TestSearchMatchesCategoryAndTags(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	// "wallet" should match every wallet — via category and/or name/tags.
	_, body := dirGet(t, h, "q=wallet")
	for _, r := range all {
		if r.Category == "wallet" && !present(body, r) {
			t.Errorf("q=wallet should list wallet %q", r.Name)
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	h := newTestServer(t)
	lower := get(t, h, "/dir?q=cake").Body.String()
	upper := get(t, h, "/dir?q=CAKE").Body.String()
	for _, body := range []string{lower, upper} {
		if !strings.Contains(body, "Cake Wallet") {
			t.Error("case-insensitive search should find Cake Wallet")
		}
	}
}

func TestSearchMultiTermAND(t *testing.T) {
	h := newTestServer(t)
	hit := get(t, h, "/dir?q=cake+wallet").Body.String()
	if !strings.Contains(hit, "Cake Wallet") {
		t.Error("q=cake wallet should match Cake Wallet")
	}
	miss := get(t, h, "/dir?q=wallet+zzz").Body.String()
	if !strings.Contains(miss, "No resources match these filters") {
		t.Error("q=wallet zzz should match nothing (AND across terms)")
	}
}

func TestSearchEmptyMatch(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "q=zzznomatch")
	if !strings.Contains(body, "No resources match these filters") {
		t.Error("no-match search should render the empty-match block")
	}
	if !strings.Contains(body, `class="dir__empty-match-reset" href="/dir"`) {
		t.Error("empty-match reset should point at /dir")
	}
}

func TestSearchComposesWithFacetsAND(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "q=cake&kyc=no")
	sel := parseSelection(url.Values{"kyc": {"no"}}, setOf(loadSeed(t).Categories()))
	search := parseSearch(url.Values{"q": {"cake"}})
	assertExactly(t, body, all, func(r *directory.Resource) bool {
		return sel.match(r) && search.match(r)
	})
}

func TestSearchPillRemovesOnlyQuery(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "q=cake&kyc=no")
	// Search pill present.
	if !strings.Contains(body, "Search: cake") {
		t.Error("search pill 'Search: cake' missing")
	}
	// Its remove URL drops q but keeps kyc=no.
	if !strings.Contains(body, `href="/dir?kyc=no"`) {
		t.Error("search pill remove URL should keep kyc=no and drop q")
	}
	// The facet pill's remove URL keeps q.
	if !strings.Contains(body, `href="/dir?q=cake"`) {
		t.Error("facet pill remove URL should preserve q=cake")
	}
}

func TestSearchPreservedInSidebarAndForm(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "q=cake")
	// Sidebar filter form carries q as a hidden input.
	if !strings.Contains(body, `<input type="hidden" name="q" value="cake">`) {
		t.Error("filter sidebar should carry q as a hidden input")
	}
	// The search input reflects the query.
	if !strings.Contains(body, `name="q"`) || !strings.Contains(body, `value="cake"`) {
		t.Error("search input should reflect the query value")
	}
}

func TestSearchFacetHiddenInputs(t *testing.T) {
	h := newTestServer(t)
	// When a facet is active, the search form carries it as a hidden input so
	// searching preserves the filter.
	_, body := dirGet(t, h, "kyc=no")
	if !strings.Contains(body, `<input type="hidden" name="kyc" value="no">`) {
		t.Error("search form should carry active facet kyc=no as a hidden input")
	}
}

func TestSearchQueryEscapedInValue(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir?q=%3Cscript%3E").Body.String()
	if strings.Contains(body, "<script>") {
		t.Error("query must be HTML-escaped in the input value")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("escaped query should appear as &lt;script&gt;")
	}
}

func TestSearchNoScriptAndHeaders(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir?q=cake")
	body := rec.Body.String()
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("/dir?q must contain no <script>")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("CSP must keep script-src 'none', got %q", csp)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("/dir?q set cookies: %v", cookies)
	}
}

// TestSearchNormalization unit-tests the term parsing limits.
func TestSearchNormalization(t *testing.T) {
	long := strings.Repeat("a", 100)
	if got := parseSearch(url.Values{"q": {long}}); len([]rune(got.raw)) != maxQueryLen {
		t.Errorf("query length = %d, want capped at %d", len([]rune(got.raw)), maxQueryLen)
	}
	many := "a b c d e f g h i j"
	if got := parseSearch(url.Values{"q": {many}}); len(got.terms) != maxTerms {
		t.Errorf("term count = %d, want capped at %d", len(got.terms), maxTerms)
	}
	if got := parseSearch(url.Values{"q": {"   "}}); got.active() {
		t.Error("whitespace-only query should not be active")
	}
}
