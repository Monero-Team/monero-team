package web

import (
	"io/fs"
	"net/http"
	"strings"

	cstrings "github.com/Monero-Team/monero-team/internal/content/strings"
	"github.com/Monero-Team/monero-team/internal/directory"
	"github.com/Monero-Team/monero-team/internal/news"
)

// handler holds the parsed templates, the resolved sections, the read-only
// directory store, and the news store, and serves the application's routes.
type handler struct {
	tmpl     templateSet
	sections map[string]cstrings.Section
	dir      *directory.Store
	news     *news.Store
}

// home renders the landing page.
func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	v := newView("/")
	v.Home = cstrings.Home
	h.tmpl.render(w, "home", v)
}

// directoryPath is the canonical path of the directory list section.
const directoryPath = "/dir"

// directory renders the directory list: one row-card per resource in the
// store's canonical order (name, slug). Fully server-rendered, no JavaScript.
func (h *handler) directory(w http.ResponseWriter, r *http.Request) {
	sec := h.sections[directoryPath]
	v := newView(sec.Path)
	v.Section = sec
	v.Directory = buildDirectoryView(h.dir, r.URL.Query())
	h.tmpl.render(w, "directory", v)
}

// section renders the coming-soon skeleton for whichever top-level or utility
// section matches the request path. The active navbar item is resolved here,
// server-side, from the canonical path.
func (h *handler) section(w http.ResponseWriter, r *http.Request) {
	sec, ok := h.sections[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	v := newView(sec.Path)
	v.Section = sec
	v.ComingSoon = cstrings.ComingSoon
	h.tmpl.render(w, "section", v)
}

// resource renders the detail page for /dir/{slug}. An unknown slug yields the
// styled 404 page with a 404 status.
func (h *handler) resource(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	res, ok := h.dir.BySlug(slug)
	if !ok {
		h.notFound(w, r)
		return
	}
	v := newView(directoryPath) // keep the Directory nav item active
	v.Resource = buildResourceDetail(res)
	h.tmpl.render(w, "resource", v)
}

// dirIndexRedirect sends /dir/ (no slug) back to the canonical list at /dir.
func (h *handler) dirIndexRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, directoryPath, http.StatusMovedPermanently)
}

// notFound renders the reusable styled 404 page with a 404 status.
func (h *handler) notFound(w http.ResponseWriter, r *http.Request) {
	v := newView("")
	h.tmpl.renderStatus(w, "not-found", v, http.StatusNotFound)
}

// healthz is a dependency-free liveness probe.
func (h *handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// assetExtTypes pins MIME types for the asset extensions we serve, so the
// response never depends on the host's mime registry (which may not know
// woff2). Content-Type is set explicitly rather than sniffed.
var assetExtTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
}

// assets serves embedded static files under /static/ with explicit MIME and
// long-lived cache headers. Directory listing is disabled.
func (h *handler) assets() http.Handler {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		// assetsFS is embedded at build time; a failure here is a programmer
		// error, not a runtime condition.
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		// Reject directory requests; only files are served.
		if clean == "" || strings.HasSuffix(clean, "/") {
			http.NotFound(w, r)
			return
		}
		if ct := assetExtTypes[ext(clean)]; ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		// Aggressive immutable caching is correct because asset URLs are
		// content-versioned: app.css is referenced as "/static/app.css?v=<hash>"
		// (see assetURL), so changing its bytes changes the URL and bypasses any
		// cached copy. Fonts are referenced unversioned but never change, so the
		// same header is safe for them. The query string is ignored here — the
		// current file is always served, so stale "?v=" values are harmless.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	}))
}

// ext returns the lowercase file extension including the dot, or "".
func ext(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return strings.ToLower(name[i:])
	}
	return ""
}
