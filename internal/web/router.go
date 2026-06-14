// Package web wires the Monero Team HTTP application: routing, middleware,
// embedded templates, and embedded static assets. It depends only on the Go
// standard library.
package web

import (
	"net/http"

	cstrings "github.com/Monero-Team/monero-team/internal/content/strings"
)

// NewHandler parses the embedded templates and returns the application's root
// http.Handler with privacy/security middleware applied to every route. It
// returns an error if templates fail to compile, so callers can fail fast at
// startup.
func NewHandler() (http.Handler, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	// Build the path → section lookup from the single source of truth so the
	// router and the templates can never drift out of sync.
	sections := make(map[string]cstrings.Section, len(cstrings.Nav)+len(cstrings.Utility))
	for _, s := range cstrings.Nav {
		sections[s.Path] = s
	}
	for _, s := range cstrings.Utility {
		sections[s.Path] = s
	}

	h := &handler{tmpl: tmpl, sections: sections}

	mux := http.NewServeMux()
	// Exact-match home; "/{$}" prevents "/" from acting as a catch-all so
	// unknown paths fall through to a 404.
	mux.HandleFunc("GET /{$}", h.home)
	for path := range sections {
		mux.HandleFunc("GET "+path, h.section)
	}
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.Handle("GET /static/", h.assets())

	return securityHeaders(mux), nil
}
