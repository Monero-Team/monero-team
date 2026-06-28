package news

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetcher retrieves the raw bytes of a feed URL. The HTTP implementation is
// used in production; tests supply a fixture implementation reading from
// testdata.
type fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

const (
	fetchTimeout = 10 * time.Second
	maxBodyBytes = 5 << 20 // 5 MiB cap on a feed body
	userAgent    = "MoneroTeam-news/1.0 (+https://github.com/Monero-Team/monero-team)"
)

// httpFetcher fetches feeds over HTTP with a timeout, a descriptive
// User-Agent, and a hard limit on the response body size.
type httpFetcher struct {
	client *http.Client
}

func newHTTPFetcher() *httpFetcher {
	return &httpFetcher{client: &http.Client{Timeout: fetchTimeout}}
}

func (f *httpFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml;q=0.9, text/xml;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news: %s: unexpected status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("news: %s: reading body: %w", url, err)
	}
	return body, nil
}
