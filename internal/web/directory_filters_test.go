package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Monero-Team/monero-team/internal/directory"
)

// dirGet fetches /dir with a raw query (e.g. "category=wallet&status=admitted").
func dirGet(t *testing.T, h http.Handler, query string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	path := "/dir"
	if query != "" {
		path += "?" + query
	}
	rec := get(t, h, path)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: got %d, want 200", path, rec.Code)
	}
	return rec, rec.Body.String()
}

// present reports whether a resource's row is rendered (name inside its anchor).
func present(body string, r *directory.Resource) bool {
	return strings.Contains(body, ">"+r.Name+"<")
}

// assertExactly checks that exactly the resources satisfying want() are rendered.
func assertExactly(t *testing.T, body string, all []*directory.Resource, want func(*directory.Resource) bool) {
	t.Helper()
	for _, r := range all {
		got := present(body, r)
		if want(r) && !got {
			t.Errorf("expected %q (%s) to be listed", r.Name, r.Slug)
		}
		if !want(r) && got {
			t.Errorf("expected %q (%s) to be filtered out", r.Name, r.Slug)
		}
	}
}

func TestDirFilterCategorySingle(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "category=wallet")
	assertExactly(t, body, all, func(r *directory.Resource) bool { return r.Category == "wallet" })
}

func TestDirFilterCategoryOR(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "category=wallet&category=exchange")
	assertExactly(t, body, all, func(r *directory.Resource) bool {
		return r.Category == "wallet" || r.Category == "exchange"
	})
}

func TestDirFilterStatusVerifiedExcludesScamQuestionable(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "status=verified")
	// No seed entry is verified, so the result must contain only verified
	// resources (none) — and crucially no scam/questionable.
	assertExactly(t, body, all, func(r *directory.Resource) bool { return r.Status == "verified" })
	for _, r := range all {
		if r.Status == directory.StatusScam || r.Status == directory.StatusQuestionable {
			if present(body, r) {
				t.Errorf("status=verified leaked a %s entry: %q", r.Status, r.Name)
			}
		}
	}
}

func TestDirFilterAccessTor(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "access=tor")
	assertExactly(t, body, all, func(r *directory.Resource) bool {
		for _, a := range r.Access {
			if a == directory.AccessTor {
				return true
			}
		}
		return false
	})
}

func TestDirFilterKycNo(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "kyc=no")
	assertExactly(t, body, all, func(r *directory.Resource) bool { return !r.KYC })
}

func TestDirFilterAND(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	// wallet AND admitted = the admitted wallets.
	_, body := dirGet(t, h, "category=wallet&status=admitted")
	assertExactly(t, body, all, func(r *directory.Resource) bool {
		return r.Category == "wallet" && r.Status == "admitted"
	})
	// wallet AND scam = nothing (no wallet is scam).
	_, body2 := dirGet(t, h, "category=wallet&status=scam")
	assertExactly(t, body2, all, func(r *directory.Resource) bool { return false })
}

func TestDirFilterGarbageIgnored(t *testing.T) {
	h := newTestServer(t)
	all := loadSeed(t).All()
	_, body := dirGet(t, h, "category=zzz&status=bogus&access=carrier-pigeon&kyc=maybe")
	// Unknown values are dropped → no active filters → full list.
	assertExactly(t, body, all, func(r *directory.Resource) bool { return true })
	if strings.Contains(body, "active-pill") {
		t.Error("garbage filters should not produce active-filter pills")
	}
}

func TestDirEmptyMatchBlock(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "category=wallet&status=scam")
	if !strings.Contains(body, "No resources match these filters") {
		t.Error("empty-match block not rendered for a no-match filter")
	}
	if !strings.Contains(body, `class="dir__empty-match-reset" href="/dir"`) {
		t.Error("empty-match reset link missing or not pointing at /dir")
	}
	if strings.Contains(body, "dirrow__name") {
		t.Error("no rows should render on an empty match")
	}
}

func TestDirActiveFilterPills(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "category=wallet")
	// Single active filter → removable pill that clears back to /dir.
	if !strings.Contains(body, "Category: wallet") {
		t.Error("active pill label 'Category: wallet' missing")
	}
	if !strings.Contains(body, `class="active-pill" href="/dir"`) {
		t.Error("active pill is not an <a> with the correct remove URL")
	}
}

func TestDirActiveFilterRemoveURLMultiple(t *testing.T) {
	h := newTestServer(t)
	_, body := dirGet(t, h, "category=wallet&status=admitted")
	// Removing one filter must preserve the other in the URL.
	if !strings.Contains(body, `href="/dir?status=admitted"`) {
		t.Error("remove-category URL should keep status=admitted")
	}
	if !strings.Contains(body, `href="/dir?category=wallet"`) {
		t.Error("remove-status URL should keep category=wallet")
	}
}

func TestDirFilterOptionCountsAreGlobal(t *testing.T) {
	store := loadSeed(t)
	wallets := 0
	for _, r := range store.All() {
		if r.Category == "wallet" {
			wallets++
		}
	}

	walletCount := func(v directoryView) int {
		for _, o := range v.Filters.Categories {
			if o.Value == "wallet" {
				return o.Count
			}
		}
		t.Fatal("wallet option missing")
		return -1
	}

	// The wallet option count must be the global total whether or not a
	// different filter is selected.
	none, _ := url.ParseQuery("")
	other, _ := url.ParseQuery("category=exchange")
	if got := walletCount(buildDirectoryView(store, none)); got != wallets {
		t.Errorf("wallet count (no filter) = %d, want %d", got, wallets)
	}
	if got := walletCount(buildDirectoryView(store, other)); got != wallets {
		t.Errorf("wallet count (other filter) = %d, want %d", got, wallets)
	}
}

func TestDirNoScriptAndHeaders(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/dir?category=wallet")
	body := rec.Body.String()

	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("/dir must contain no <script> (no-JS)")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("CSP must keep script-src 'none', got %q", csp)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("/dir set cookies: %v", cookies)
	}
	if !strings.Contains(body, `rel="noopener noreferrer"`) {
		t.Error(`card links must still carry rel="noopener noreferrer"`)
	}
}

// TestSelectionMatch unit-tests the filter predicate directly, including the
// tor/verified cases the clearnet-only seed cannot exercise positively.
func TestSelectionMatch(t *testing.T) {
	verifiedTor := &directory.Resource{Category: "wallet", Status: "verified", KYC: false, Access: []string{"clearnet", "tor"}}
	admittedClear := &directory.Resource{Category: "exchange", Status: "admitted", KYC: true, Access: []string{"clearnet"}}

	validCat := setOf([]string{"wallet", "exchange"})
	mk := func(raw string) selection {
		q, _ := url.ParseQuery(raw)
		return parseSelection(q, validCat)
	}

	cases := []struct {
		name  string
		query string
		res   *directory.Resource
		want  bool
	}{
		{"status verified matches", "status=verified", verifiedTor, true},
		{"status verified rejects admitted", "status=verified", admittedClear, false},
		{"access tor matches", "access=tor", verifiedTor, true},
		{"access tor rejects clearnet-only", "access=tor", admittedClear, false},
		{"kyc yes matches", "kyc=yes", admittedClear, true},
		{"kyc yes rejects non-kyc", "kyc=yes", verifiedTor, false},
		{"AND across dims", "category=wallet&status=verified", verifiedTor, true},
		{"AND fails one dim", "category=wallet&status=admitted", verifiedTor, false},
		{"OR within dim", "category=wallet&category=exchange", admittedClear, true},
		{"garbage ignored = no constraint", "category=zzz", admittedClear, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mk(c.query).match(c.res); got != c.want {
				t.Errorf("match = %v, want %v", got, c.want)
			}
		})
	}
}
