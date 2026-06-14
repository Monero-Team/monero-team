package web

import (
	"io/fs"
	"net/http"
	"strings"

	cstrings "github.com/Monero-Team/monero-team/internal/content/strings"
)

// handler holds the parsed templates and serves the application's routes.
type handler struct {
	tmpl     templateSet
	sections map[string]cstrings.Section
}

// home renders the landing page.
func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	v := newView("/")
	v.Home = cstrings.Home
	h.tmpl.render(w, "home", v)
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
		// Assets are content-addressable enough for a long cache during Phase 1;
		// they are immutable for a given binary build.
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
