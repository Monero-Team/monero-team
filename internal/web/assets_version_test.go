package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// stylesheetHref extracts the href of the app.css <link> from rendered HTML.
var stylesheetHref = regexp.MustCompile(`<link rel="stylesheet" href="([^"]+)"`)

func TestStylesheetLinkIsVersioned(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/").Body.String()

	m := stylesheetHref.FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no stylesheet <link> found in rendered HTML")
	}
	href := m[1]
	if !strings.HasPrefix(href, "/static/app.css?v=") {
		t.Errorf("stylesheet href %q is not a versioned /static/app.css URL", href)
	}
	// The version must be the actual content hash, not a placeholder.
	if want := assetURL("app.css"); href != want {
		t.Errorf("stylesheet href = %q, want %q", href, want)
	}
	if !strings.Contains(href, assetVersions["app.css"]) {
		t.Errorf("stylesheet href %q does not carry the app.css content hash", href)
	}
}

func TestVersionedStylesheetServed(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, assetURL("app.css"))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET versioned app.css: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/css; charset=utf-8", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want it to contain immutable", cc)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("versioned static response set cookies: %v", cookies)
	}
}

// TestAssetHashTracksContent is the core guarantee: the version string is a
// function of the file's bytes, so different content yields a different URL.
func TestAssetHashTracksContent(t *testing.T) {
	got := assetVersions["app.css"]
	if len(got) != assetHashLen {
		t.Fatalf("app.css hash %q has length %d, want %d", got, len(got), assetHashLen)
	}

	// Recompute over a one-byte-different copy of the embedded FS content and
	// confirm the hash changes.
	orig, err := assetsFS.ReadFile("assets/app.css")
	if err != nil {
		t.Fatalf("read embedded app.css: %v", err)
	}
	mutated := append(append([]byte{}, orig...), '\n')
	if sameHash := shortHash(orig) == shortHash(mutated); sameHash {
		t.Error("hash did not change when content changed")
	}
	if shortHash(orig) != got {
		t.Errorf("recomputed hash %q != registered %q", shortHash(orig), got)
	}
}

func TestUnknownAssetFallsBack(t *testing.T) {
	if got := assetURL("does-not-exist.css"); got != "/static/does-not-exist.css" {
		t.Errorf("assetURL(unknown) = %q, want unversioned fallback", got)
	}
}
