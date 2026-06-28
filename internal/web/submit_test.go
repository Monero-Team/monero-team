package web

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// postForm performs an in-process POST with form-encoded values.
func postForm(t *testing.T, h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// validSubmission is a form body that passes validation.
func validSubmission() url.Values {
	return url.Values{
		"name":        {"Acme Wallet"},
		"category":    {"wallet"},
		"access":      {"clearnet"},
		"clearnet":    {"https://acme.example"},
		"kyc":         {"no"},
		"description": {"A privacy-respecting test wallet."},
		"tags":        {"Desktop, Open-Source, desktop"}, // mixed case + dup → normalized
	}
}

func TestSubmitGetRendersForm(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/submit")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /submit: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`method="POST" action="/submit"`,
		`name="name"`, `name="category"`, `name="description"`,
		`value="wallet"`, // category option from the enum
		`name="access" value="clearnet"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /submit missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("/submit contains <script>")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("CSP missing script-src 'none': %q", csp)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("/submit set cookies")
	}
}

func TestSubmitPostSuccess(t *testing.T) {
	h := newTestServer(t)
	rec := postForm(t, h, "/submit", validSubmission())
	if rec.Code != http.StatusOK {
		t.Fatalf("POST valid: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Submission ready",
		"data/directory/acme-wallet.json",
		"mailto:mail@monero.team",
		`&#34;status&#34;: &#34;pending&#34;`, // status pending, HTML-escaped in the textarea
	} {
		if !strings.Contains(body, want) {
			t.Errorf("success screen missing %q", want)
		}
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("POST set cookies")
	}
}

// TestSubmitJSONIsValidEntry checks the generated JSON parses, carries the
// pending status, and reflects the normalizations.
func TestSubmitJSONIsValidEntry(t *testing.T) {
	res := buildResource(readForm(validSubmission()))
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("generated JSON does not parse: %v", err)
	}
	if got["status"] != "pending" {
		t.Errorf("status = %v, want pending", got["status"])
	}
	if got["slug"] != "acme-wallet" {
		t.Errorf("slug = %v, want acme-wallet", got["slug"])
	}
	tags, _ := got["tags"].([]any)
	if len(tags) != 2 { // "desktop","open-source" — lowercased + de-duped
		t.Errorf("tags = %v, want 2 normalized tags", got["tags"])
	}
}

func TestSubmitPostInvalidKeepsValuesAndShowsErrors(t *testing.T) {
	h := newTestServer(t)
	form := url.Values{
		"name":        {"Sticky Name"}, // valid, must be echoed back
		"category":    {""},            // invalid
		"description": {""},            // invalid
		// no access, no links → more errors
	}
	rec := postForm(t, h, "/submit", form)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST invalid: got %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Submission ready") {
		t.Error("invalid POST should not show the success screen")
	}
	if !strings.Contains(body, `value="Sticky Name"`) {
		t.Error("invalid POST did not preserve the entered name")
	}
	// Inline error markup must appear.
	if !strings.Contains(body, `class="field__error"`) {
		t.Error("invalid POST rendered no inline field errors")
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("POST set cookies")
	}
}

// TestSubmitDoesNotLogContent guards the privacy promise: submission values
// must never reach the logs.
func TestSubmitDoesNotLogContent(t *testing.T) {
	h := newTestServer(t)

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	form := validSubmission()
	form.Set("name", "Zzx Secret Marker 8472")
	form.Set("description", "Confidential marker 5519 should not be logged.")
	postForm(t, h, "/submit", form)

	if logs := buf.String(); strings.Contains(logs, "Secret Marker 8472") || strings.Contains(logs, "5519") {
		t.Errorf("submission content leaked into logs:\n%s", logs)
	}
}

func TestFooterHasSubmitLink(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, `<a href="/submit">Add a site</a>`) {
		t.Error(`footer missing "Add a site" link to /submit`)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Acme Wallet":      "acme-wallet",
		"  Feather  ":      "feather",
		"Cake!! Wallet 2":  "cake-wallet-2",
		"--Already-Slug--": "already-slug",
		"!!!":              "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// readForm is a tiny adapter so the JSON test can reuse readSubmitForm without
// constructing an *http.Request.
func readForm(v url.Values) submitForm {
	access := map[string]bool{}
	for _, a := range v["access"] {
		access[a] = true
	}
	return submitForm{
		Name:     strings.TrimSpace(v.Get("name")),
		Category: strings.TrimSpace(v.Get("category")),
		Country:  strings.TrimSpace(v.Get("country")),
		Access:   access,
		KYC:      v.Get("kyc"),
		Tags:     strings.TrimSpace(v.Get("tags")),
		Links: submitLinks{
			Clearnet: strings.TrimSpace(v.Get("clearnet")),
			Onion:    strings.TrimSpace(v.Get("onion")),
			I2P:      strings.TrimSpace(v.Get("i2p")),
		},
		Description: strings.TrimSpace(v.Get("description")),
	}
}
