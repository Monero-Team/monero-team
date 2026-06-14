package web

import "embed"

// templatesFS holds the HTML templates compiled into the binary.
//
//go:embed templates/*.html
var templatesFS embed.FS

// assetsFS holds the static assets (CSS, self-hosted fonts) compiled into the
// binary. Everything served under /static/ originates here — nothing is
// fetched from an external host at build time or runtime.
//
//go:embed assets/app.css assets/fonts
var assetsFS embed.FS
