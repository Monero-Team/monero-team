// Package web wires the Monero Team HTTP application: routing, middleware,
// embedded templates, and embedded static assets. It depends only on the Go
// standard library.
package web

import (
	"log"
	"net/http"

	"github.com/Monero-Team/monero-team/data"
	cstrings "github.com/Monero-Team/monero-team/internal/content/strings"
	"github.com/Monero-Team/monero-team/internal/directory"
	"github.com/Monero-Team/monero-team/internal/news"
)

// NewHandler parses the embedded templates, loads and validates the embedded
// directory dataset, and returns the application's root http.Handler with
// privacy/security middleware applied to every route. The news store is owned
// by the caller (shared with the background collector) and read by /news. It
// returns an error if templates fail to compile or the dataset fails
// validation, so callers can fail fast at startup.
func NewHandler(newsStore *news.Store) (http.Handler, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	dir, err := directory.Load(data.Files)
	if err != nil {
		return nil, err
	}
	log.Printf("directory: loaded %d resources", dir.Len())

	// Build the path → section lookup from the single source of truth so the
	// router and the templates can never drift out of sync.
	sections := make(map[string]cstrings.Section, len(cstrings.Nav)+len(cstrings.Utility))
	for _, s := range cstrings.Nav {
		sections[s.Path] = s
	}
	for _, s := range cstrings.Utility {
		sections[s.Path] = s
	}

	h := &handler{tmpl: tmpl, sections: sections, dir: dir, news: newsStore}

	mux := http.NewServeMux()
	// Exact-match home; "/{$}" prevents "/" from acting as a catch-all so
	// unknown paths fall through to a 404.
	mux.HandleFunc("GET /{$}", h.home)
	for path := range sections {
		switch path {
		case directoryPath:
			// /dir renders the directory list.
			mux.HandleFunc("GET "+path, h.directory)
		case newsPath:
			// /news renders the news feed.
			mux.HandleFunc("GET "+path, h.newsFeed)
		default:
			// Every other section is still the coming-soon skeleton.
			mux.HandleFunc("GET "+path, h.section)
		}
	}
	// Resource detail pages. "/dir/{$}" (bare "/dir/") redirects to the list;
	// "/dir/{slug}" renders one resource or the styled 404.
	mux.HandleFunc("GET /dir/{$}", h.dirIndexRedirect)
	mux.HandleFunc("GET /dir/{slug}", h.resource)
	// Resource submission: show the form, and accept it (validate-only — the
	// server persists nothing).
	mux.HandleFunc("GET /submit", h.submitGet)
	mux.HandleFunc("POST /submit", h.submitPost)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.Handle("GET /static/", h.assets())
	// Catch-all for any other GET path → reusable styled 404. The more
	// specific "GET /{$}" above still serves the home page for exactly "/".
	mux.HandleFunc("GET /", h.notFound)

	return securityHeaders(mux), nil
}
