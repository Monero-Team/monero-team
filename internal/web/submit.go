package web

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/Monero-Team/monero-team/internal/directory"
)

const (
	submitPath  = "/submit"
	submitEmail = "submit@monero.team"
)

// submitLinks holds the three optional link inputs as raw strings.
type submitLinks struct {
	Clearnet string
	Onion    string
	I2P      string
}

// submitForm is the sticky form state echoed back to the template so a failed
// submission keeps everything the user typed. Field names match submit-form.html.
type submitForm struct {
	Name        string
	Category    string
	Country     string
	Access      map[string]bool // keys clearnet/tor/i2p
	KYC         string          // "no" | "yes" | ""
	Tags        string
	Links       submitLinks
	Description string
}

// submitResult is the success-screen payload.
type submitResult struct {
	JSON     string
	Filename string
	Email    string
	Name     string // resource name, used for the mailto subject
}

// submitView is the page model for /submit (form state or success screen).
type submitView struct {
	Success    bool
	Action     string
	Categories []string
	Form       submitForm
	Errors     map[string]string // form field key → message
	Result     submitResult
}

func newSubmitView() submitView {
	return submitView{
		Action:     submitPath,
		Categories: directory.Categories,
		Form:       submitForm{Access: map[string]bool{}},
		Errors:     map[string]string{},
	}
}

// submitGet renders the empty intake form.
func (h *handler) submitGet(w http.ResponseWriter, r *http.Request) {
	h.renderSubmit(w, http.StatusOK, newSubmitView())
}

// submitPost validates the submission. On error it re-renders the form with
// inline messages and the user's values preserved; on success it shows the
// ready-to-send JSON. Nothing is persisted, and the submission content is never
// logged.
func (h *handler) submitPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sv := newSubmitView()
		sv.Errors["form"] = "Could not read the submitted form."
		h.renderSubmit(w, http.StatusBadRequest, sv)
		return
	}

	form := readSubmitForm(r)
	res := buildResource(form)

	if ferrs := directory.ValidateSubmission(res); len(ferrs) > 0 {
		sv := newSubmitView()
		sv.Form = form
		sv.Errors = mapFieldErrors(ferrs)
		h.renderSubmit(w, http.StatusBadRequest, sv)
		return
	}

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		sv := newSubmitView()
		sv.Form = form
		sv.Errors["form"] = "Could not format the submission."
		h.renderSubmit(w, http.StatusInternalServerError, sv)
		return
	}

	sv := newSubmitView()
	sv.Success = true
	sv.Result = submitResult{
		JSON:     string(out),
		Filename: "data/directory/" + res.Slug + ".json",
		Email:    submitEmail,
		Name:     res.Name,
	}
	h.renderSubmit(w, http.StatusOK, sv)
}

func (h *handler) renderSubmit(w http.ResponseWriter, status int, sv submitView) {
	v := newView(submitPath)
	v.Submit = sv
	w.WriteHeader(status)
	h.tmpl.render(w, "submit", v)
}

// readSubmitForm extracts and trims the posted fields into sticky form state.
func readSubmitForm(r *http.Request) submitForm {
	access := map[string]bool{}
	for _, a := range r.Form["access"] {
		switch a {
		case directory.AccessClearnet, directory.AccessTor, directory.AccessI2P:
			access[a] = true
		}
	}
	return submitForm{
		Name:     strings.TrimSpace(r.FormValue("name")),
		Category: strings.TrimSpace(r.FormValue("category")),
		Country:  strings.TrimSpace(r.FormValue("country")),
		Access:   access,
		KYC:      r.FormValue("kyc"),
		Tags:     strings.TrimSpace(r.FormValue("tags")),
		Links: submitLinks{
			Clearnet: strings.TrimSpace(r.FormValue("clearnet")),
			Onion:    strings.TrimSpace(r.FormValue("onion")),
			I2P:      strings.TrimSpace(r.FormValue("i2p")),
		},
		Description: strings.TrimSpace(r.FormValue("description")),
	}
}

// buildResource turns sticky form state into a directory.Resource with a
// generated slug and the placeholder status "pending" (a moderator assigns the
// real status). Normalizations: country uppercased, access in canonical order,
// tags lowercased/de-duplicated, empty links omitted.
func buildResource(f submitForm) *directory.Resource {
	r := &directory.Resource{
		Slug:        slugify(f.Name),
		Name:        f.Name,
		Category:    f.Category,
		Status:      "pending",
		KYC:         f.KYC == "yes",
		Description: f.Description,
	}
	if f.Country != "" {
		c := strings.ToUpper(f.Country)
		r.Country = &c
	}
	for _, a := range directory.AccessTypes {
		if f.Access[a] {
			r.Access = append(r.Access, a)
		}
	}
	r.Tags = parseTags(f.Tags)
	if f.Links.Clearnet != "" {
		v := f.Links.Clearnet
		r.Links.Clearnet = &v
	}
	if f.Links.Onion != "" {
		v := f.Links.Onion
		r.Links.Onion = &v
	}
	if f.Links.I2P != "" {
		v := f.Links.I2P
		r.Links.I2P = &v
	}
	return r
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify derives a directory slug from a name: lowercase, non-alphanumeric
// runs collapsed to single hyphens, trimmed. May return "" for an empty or
// symbol-only name; the name's own validation then reports the real problem.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlugChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// parseTags splits a comma-separated string into lowercase, de-duplicated,
// non-empty tags.
func parseTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, t := range strings.Split(s, ",") {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// fieldErrorKey maps a directory validation field to the form's error key.
var fieldErrorKey = map[string]string{
	"name":           "name",
	"description":    "description",
	"category":       "category",
	"country":        "country",
	"access":         "access",
	"tags":           "tags",
	"links.clearnet": "links_clearnet",
	"links.onion":    "links_onion",
	"links.i2p":      "links_i2p",
	"links":          "form", // "at least one link" — no single field owns it
	"slug":           "name", // slug is generated from the name
}

// mapFieldErrors converts directory field errors into the form's error map,
// keeping the first message per field.
func mapFieldErrors(ferrs []directory.FieldError) map[string]string {
	out := make(map[string]string, len(ferrs))
	for _, fe := range ferrs {
		key, ok := fieldErrorKey[fe.Field]
		if !ok {
			key = "form"
		}
		if _, exists := out[key]; !exists {
			out[key] = capitalize(fe.Msg)
		}
	}
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
