// Command server runs the Monero Team web application: a single, self-contained
// binary serving privacy-first HTML with no external dependencies.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Monero-Team/monero-team/internal/news"
	"github.com/Monero-Team/monero-team/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run builds the handler and serves it until an interrupt/terminate signal,
// then shuts down gracefully. It returns a non-nil error only on a genuine
// failure (not on a clean shutdown).
func run() error {
	addr := resolveAddr()

	// The news store is shared between the read path (/news) and the
	// background collector started below.
	newsStore := news.NewStore(0)

	handler, err := web.NewHandler(newsStore)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen up front so a bind failure is reported before we claim to be
	// serving, and so the chosen port is known even when :0 is requested.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the background news collector. It does not block startup (a cold,
	// empty cache is expected) and stops gracefully when ctx is cancelled. This
	// is the only outbound network egress; the request-serving path makes none.
	news.NewScheduler(news.Sources, newsStore, 0).Start(ctx)
	log.Printf("news: scheduler started (%d sources)", len(news.Sources))

	errCh := make(chan error, 1)
	go func() {
		log.Printf("monero.team listening on http://%s", ln.Addr())
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Print("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		log.Print("shutdown complete")
		return nil
	}
}

// resolveAddr determines the listen address. Precedence: -addr flag, then the
// PORT or ADDR environment variable, then the default ":8080". A bare numeric
// value (flag or PORT) is treated as a port and bound on all interfaces.
func resolveAddr() string {
	flagAddr := flag.String("addr", "", "listen address (host:port or :port); overrides PORT/ADDR env")
	flag.Parse()

	switch {
	case *flagAddr != "":
		return normalizeAddr(*flagAddr)
	case os.Getenv("ADDR") != "":
		return normalizeAddr(os.Getenv("ADDR"))
	case os.Getenv("PORT") != "":
		return normalizeAddr(os.Getenv("PORT"))
	default:
		return ":8080"
	}
}

// normalizeAddr turns a bare port (e.g. "8080") into ":8080" and leaves a full
// host:port address untouched.
func normalizeAddr(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ":8080"
	}
	if !strings.Contains(v, ":") {
		return ":" + v
	}
	return v
}
