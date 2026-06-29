package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/Monero-Team/monero-team/data"
	"github.com/Monero-Team/monero-team/internal/directory"
)

// loadSeed loads the real embedded dataset for assertions about /dir content.
func loadSeed(t *testing.T) *directory.Store {
	t.Helper()
	s, err := directory.Load(data.Files)
	if err != nil {
		t.Fatalf("directory.Load: %v", err)
	}
	if s.Len() == 0 {
		t.Fatal("seed dataset is empty")
	}
	return s
}

func TestDirectoryReturns200(t *testing.T) {
	h := newTestServer(t)
	if rec := get(t, h, "/dir"); rec.Code != http.StatusOK {
		t.Fatalf("GET /dir: got %d, want 200", rec.Code)
	}
}

// TestDirectoryRendersEverySeedResource checks that every seed resource appears
// by name with its sentence-case status label and status CSS class.
func TestDirectoryRendersEverySeedResource(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir").Body.String()
	store := loadSeed(t)

	if got := store.Len(); got != 6 {
		t.Errorf("seed dataset has %d resources, want 6", got)
	}
	for _, r := range store.All() {
		if !strings.Contains(body, r.Name) {
			t.Errorf("/dir missing resource name %q", r.Name)
		}
		if label := statusLabel(r.Status); !strings.Contains(body, ">"+label+"<") {
			t.Errorf("/dir missing status label %q for %q", label, r.Slug)
		}
		if class := "status--" + r.Status; !strings.Contains(body, class) {
			t.Errorf("/dir missing status class %q for %q", class, r.Slug)
		}
	}
}

// TestDirectoryScamWarning checks a scam entry is present, stays in the list,
// and carries the explicit warning.
func TestDirectoryScamWarning(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir").Body.String()
	store := loadSeed(t)

	var scam *directory.Resource
	for _, r := range store.All() {
		if r.Status == directory.StatusScam {
			scam = r
			break
		}
	}
	if scam == nil {
		t.Fatal("expected a scam entry in the seed dataset")
	}
	if !strings.Contains(body, scam.Name) {
		t.Errorf("scam entry %q not listed", scam.Name)
	}
	if !strings.Contains(body, "Reported — do not use") {
		t.Error(`scam warning "Reported — do not use" not rendered`)
	}
	if !strings.Contains(body, "dirrow--scam") {
		t.Error("scam row dimming class not rendered")
	}
}

// TestDirectoryRatingPlaceholder checks the "not rated" placeholder is present
// and no invented star or numeric rating leaks in.
func TestDirectoryRatingPlaceholder(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir").Body.String()

	if !strings.Contains(body, "not rated") {
		t.Error(`missing "not rated" placeholder`)
	}
	for _, artefact := range []string{"★", "☆", "/5", "out of 5"} {
		if strings.Contains(body, artefact) {
			t.Errorf("unexpected rating artefact %q in /dir", artefact)
		}
	}
	if m := regexp.MustCompile(`\d(\.\d+)?\s*(stars?|/\s*5)`).FindString(body); m != "" {
		t.Errorf("unexpected numeric rating %q in /dir", m)
	}
}

// TestDirectoryExternalLinksHardened checks every external link carries rel.
func TestDirectoryExternalLinksHardened(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir").Body.String()

	// Both the name link and the Open link are external; match anchors that
	// point at an external origin (regardless of attribute order) and require
	// each to carry the rel attribute.
	external := regexp.MustCompile(`<a [^>]*href="https?://[^"]*"[^>]*>`).FindAllString(body, -1)
	if len(external) < 2 {
		t.Fatalf("/dir rendered %d external links, want at least 2 (name + Open)", len(external))
	}
	for _, a := range external {
		if !strings.Contains(a, `rel="noopener noreferrer"`) {
			t.Errorf("external link missing rel=\"noopener noreferrer\": %s", a)
		}
	}
}

// TestDirectoryHasNoScript checks the page works without JavaScript.
func TestDirectoryHasNoScript(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/dir").Body.String()
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("/dir contains a <script> tag; the page must work without JS")
	}
}

// TestDirectoryReplacesComingSoon checks /dir is no longer the skeleton while
// another section (/news) still is.
func TestDirectoryReplacesComingSoon(t *testing.T) {
	h := newTestServer(t)

	dir := get(t, h, "/dir").Body.String()
	if strings.Contains(dir, "Coming soon") {
		t.Error("/dir still shows the coming-soon skeleton")
	}
	if !strings.Contains(dir, "Resource directory") {
		t.Error("/dir does not render the directory list heading")
	}

	digest := get(t, h, "/digest").Body.String()
	if !strings.Contains(digest, "Coming soon") {
		t.Error("/digest should still show the coming-soon skeleton")
	}
}

// TestPrimaryURLPriority checks the clearnet → onion → i2p selection in Go.
func TestPrimaryURLPriority(t *testing.T) {
	clear, onion, i2p := "https://example.test", "http://abc.onion", "http://abc.i2p"
	cases := []struct {
		name string
		in   directory.Links
		want string
	}{
		{"all present prefers clearnet", directory.Links{Clearnet: &clear, Onion: &onion, I2P: &i2p}, clear},
		{"no clearnet prefers onion", directory.Links{Onion: &onion, I2P: &i2p}, onion},
		{"only i2p", directory.Links{I2P: &i2p}, i2p},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := primaryURL(&directory.Resource{Links: c.in}); got != c.want {
				t.Errorf("primaryURL = %q, want %q", got, c.want)
			}
		})
	}
}
