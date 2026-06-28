package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

// pages maps a logical page name to the content template file(s) it needs. The
// first file defines the "title" and "main" blocks consumed by base.html; any
// further files supply page-specific partials.
var pages = map[string][]string{
	"home":      {"templates/home.html"},
	"section":   {"templates/section.html"},
	"directory": {"templates/directory.html", "templates/directory-list.html", "templates/directory-row.html", "templates/filter-sidebar.html", "templates/active-filters.html", "templates/dir-search.html"},
	"resource":  {"templates/resource.html", "templates/resource-detail.html"},
	"not-found": {"templates/not-found.html"},
	"submit":    {"templates/submit.html", "templates/submit-form.html", "templates/submit-success.html"},
}

// templateSet holds one fully-parsed template per page. Because every page
// content file redefines the same block names ("title", "main"), each page is
// parsed into its own *template.Template rather than a shared set.
type templateSet map[string]*template.Template

// shared lists the layout and partial templates included with every page.
var shared = []string{
	"templates/base.html",
	"templates/navbar.html",
	"templates/footer.html",
}

// templateFuncs are the helpers available to every template. "asset" maps a
// logical asset name to its cache-busting URL so templates never hard-code the
// versioned path.
var templateFuncs = template.FuncMap{
	"asset": assetURL,
}

// parseTemplates compiles all page templates from the embedded FS. It returns
// an error if any template is missing or malformed so the server fails fast at
// startup rather than at request time.
func parseTemplates() (templateSet, error) {
	set := make(templateSet, len(pages))
	for name, content := range pages {
		files := append(append([]string{}, shared...), content...)
		tmpl, err := template.New("base").Funcs(templateFuncs).ParseFS(templatesFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		set[name] = tmpl
	}
	return set, nil
}

// render executes the named page's "base" template against data with a 200
// status.
func (s templateSet) render(w http.ResponseWriter, name string, data any) {
	s.renderStatus(w, name, data, http.StatusOK)
}

// renderStatus executes the named page's "base" template against data and
// writes the result with the given status code. The page is rendered into a
// buffer first so that a template error surfaces as a 500 with no partial
// output, rather than a half-written response.
func (s templateSet) renderStatus(w http.ResponseWriter, name string, data any, code int) {
	tmpl, ok := s[name]
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = buf.WriteTo(w)
}
