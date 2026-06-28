package news

import (
	"context"
	"log"
	"time"
)

const (
	// DefaultInterval is the poll interval when none (or a non-positive one) is
	// configured.
	DefaultInterval = 20 * time.Minute
	// MinInterval is the floor enforced on any configured interval, to avoid
	// hammering feed servers.
	MinInterval = 5 * time.Minute
)

// Scheduler periodically refreshes all sources into a Store. Each refresh is
// fail-soft: one source's error/timeout/non-XML response is recorded in health
// and does not stop the others.
type Scheduler struct {
	sources  []Source
	fetcher  fetcher
	store    *Store
	interval time.Duration
	logger   *log.Logger
}

// NewScheduler builds a scheduler using the production HTTP fetcher.
func NewScheduler(sources []Source, store *Store, interval time.Duration) *Scheduler {
	return newScheduler(sources, newHTTPFetcher(), store, interval)
}

func newScheduler(sources []Source, f fetcher, store *Store, interval time.Duration) *Scheduler {
	switch {
	case interval <= 0:
		interval = DefaultInterval
	case interval < MinInterval:
		interval = MinInterval
	}
	return &Scheduler{
		sources:  sources,
		fetcher:  f,
		store:    store,
		interval: interval,
		logger:   log.Default(),
	}
}

// Start launches the refresh loop in a background goroutine: an immediate
// refresh, then one every interval, until ctx is cancelled (graceful shutdown).
func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		s.refreshAll(ctx)
		t := time.NewTicker(s.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.refreshAll(ctx)
			}
		}
	}()
}

// refreshAll fetches and merges every source once, then logs a one-line
// summary. Failures are recorded per source (in health) and never abort the
// sweep; the summary's "M/K" reflects how many sources succeeded. Only counts
// are logged — never article titles, links, or any content.
func (s *Scheduler) refreshAll(ctx context.Context) {
	total := len(s.sources)
	ok, fetched := 0, 0
	for _, src := range s.sources {
		if ctx.Err() != nil {
			return
		}
		body, err := s.fetcher.Fetch(ctx, src.FeedURL)
		if err != nil {
			s.store.SetError(src.Name, err)
			continue
		}
		items, err := parseFeed(body, src.Name)
		if err != nil {
			s.store.SetError(src.Name, err)
			continue
		}
		s.store.Merge(src.Name, items)
		ok++
		fetched += len(items)
	}
	s.logf("news: fetched %d items from %d/%d sources", fetched, ok, total)
}

func (s *Scheduler) logf(format string, args ...any) {
	if s.logger != nil {
		s.logger.Printf(format, args...)
	}
}
