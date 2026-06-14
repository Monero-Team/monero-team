package web

import "net/http"

// contentSecurityPolicy locks the page to same-origin resources only and
// forbids scripts entirely — the platform's no-JS baseline is enforced at the
// header level, not merely by convention.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'none'; " +
	"style-src 'self'; " +
	"font-src 'self'; " +
	"img-src 'self'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

// permissionsPolicy disables every powerful browser feature we do not use.
const permissionsPolicy = "accelerometer=(), autoplay=(), camera=(), " +
	"display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), " +
	"gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), " +
	"publickey-credentials-get=(), screen-wake-lock=(), usb=(), " +
	"xr-spatial-tracking=(), browsing-topics=()"

// securityHeaders applies the privacy and security response headers to every
// request. No cookies are ever set anywhere in the application.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", permissionsPolicy)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
