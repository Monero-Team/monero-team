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
	"directory": {"templates/directory.html", "templates/directory-list.html", "templates/directory-row.html", "templates/filter-sidebar.html", "templates/active-filters.html"},
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

// parseTemplates compiles all page templates from the embedded FS. It returns
// an error if any template is missing or malformed so the server fails fast at
// startup rather than at request time.
func parseTemplates() (templateSet, error) {
	set := make(templateSet, len(pages))
	for name, content := range pages {
		files := append(append([]string{}, shared...), content...)
		tmpl, err := template.New("base").ParseFS(templatesFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		set[name] = tmpl
	}
	return set, nil
}

// render executes the named page's "base" template against data and writes the
// result. The page is rendered into a buffer first so that a template error
// surfaces as a 500 with no partial output, rather than a half-written 200.
func (s templateSet) render(w http.ResponseWriter, name string, data any) {
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
	_, _ = buf.WriteTo(w)
}
