package news

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// defaultPerSourceLimit caps how many items are retained per source.
const defaultPerSourceLimit = 20

// SourceHealth reports the last fetch outcome for one source.
type SourceHealth struct {
	Source    string
	LastOK    time.Time
	LastErr   time.Time
	LastError string
	ItemCount int
}

// Store is a thread-safe, in-memory collection of news items grouped by source.
// Reads (All/Latest) flatten the sources, de-duplicate, and sort by publish
// time descending.
type Store struct {
	mu        sync.RWMutex
	perSource int
	bySource  map[string][]NewsItem
	health    map[string]*SourceHealth
}

// NewStore returns an empty store. A non-positive perSourceLimit falls back to
// the default.
func NewStore(perSourceLimit int) *Store {
	if perSourceLimit <= 0 {
		perSourceLimit = defaultPerSourceLimit
	}
	return &Store{
		perSource: perSourceLimit,
		bySource:  make(map[string][]NewsItem),
		health:    make(map[string]*SourceHealth),
	}
}

// Merge replaces the stored items for a source with a freshly fetched set:
// de-duplicated within the set, sorted newest-first, and capped at the
// per-source limit. It records a successful fetch in the source's health.
func (s *Store) Merge(source string, items []NewsItem) {
	cleaned := dedupe(items)
	sortByPublishedDesc(cleaned)
	if len(cleaned) > s.perSource {
		cleaned = cleaned[:s.perSource]
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.bySource[source] = cleaned
	h := s.healthLocked(source)
	h.LastOK = time.Now().UTC()
	h.ItemCount = len(cleaned)
}

// SetError records a failed fetch for a source without disturbing its items.
func (s *Store) SetError(source string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.healthLocked(source)
	h.LastErr = time.Now().UTC()
	if err != nil {
		h.LastError = err.Error()
	}
}

// healthLocked returns (creating if needed) the health record for source. The
// caller must hold s.mu.
func (s *Store) healthLocked(source string) *SourceHealth {
	h, ok := s.health[source]
	if !ok {
		h = &SourceHealth{Source: source}
		s.health[source] = h
	}
	return h
}

// All returns every stored item across sources, de-duplicated (by link, then by
// normalized title) and sorted newest-first.
func (s *Store) All() []NewsItem {
	s.mu.RLock()
	var all []NewsItem
	for _, src := range s.bySource {
		all = append(all, src...)
	}
	s.mu.RUnlock()

	sortByPublishedDesc(all)
	return dedupe(all)
}

// Latest returns the n most recent items (fewer if not available). A
// non-positive n returns nil.
func (s *Store) Latest(n int) []NewsItem {
	if n <= 0 {
		return nil
	}
	all := s.All()
	if len(all) > n {
		all = all[:n]
	}
	return all
}

// Health returns a snapshot of each source's health, sorted by source name.
func (s *Store) Health() []SourceHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SourceHealth, 0, len(s.health))
	for _, h := range s.health {
		out = append(out, *h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out
}

// dedupe removes duplicate items, keeping the first occurrence (callers sort
// newest-first beforehand so the newest copy wins). Primary key is the link;
// the normalized title is a secondary key so the same headline from different
// links collapses too.
func dedupe(items []NewsItem) []NewsItem {
	seenLink := make(map[string]bool, len(items))
	seenTitle := make(map[string]bool, len(items))
	out := make([]NewsItem, 0, len(items))
	for _, it := range items {
		link := strings.TrimSpace(it.Link)
		title := normalizeTitle(it.Title)
		if link != "" && seenLink[link] {
			continue
		}
		if title != "" && seenTitle[title] {
			continue
		}
		if link != "" {
			seenLink[link] = true
		}
		if title != "" {
			seenTitle[title] = true
		}
		out = append(out, it)
	}
	return out
}

// sortByPublishedDesc sorts newest-first, with stable, deterministic tie-breaks
// (title then link) so equal/zero timestamps order predictably. Zero times sort
// last.
func sortByPublishedDesc(items []NewsItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if !a.Published.Equal(b.Published) {
			return a.Published.After(b.Published)
		}
		if a.Title != b.Title {
			return a.Title < b.Title
		}
		return a.Link < b.Link
	})
}

func normalizeTitle(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
