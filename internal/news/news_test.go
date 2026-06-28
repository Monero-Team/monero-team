package news

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureFetcher serves local testdata files (or a configured error) per URL.
type fixtureFetcher struct {
	files map[string]string // url → testdata filename
	errs  map[string]error  // url → error to return
}

func (f fixtureFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	if err := f.errs[url]; err != nil {
		return nil, err
	}
	name, ok := f.files[url]
	if !ok {
		return nil, fmt.Errorf("no fixture for %s", url)
	}
	return os.ReadFile(filepath.Join("testdata", name))
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseRSS(t *testing.T) {
	items, err := parseFeed(readFixture(t, "rss.xml"), "Blog")
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[0].Title != "Monero 0.18 released" || items[0].Link != "https://example.test/0-18" {
		t.Errorf("item[0] = %+v", items[0])
	}
	if items[0].Source != "Blog" {
		t.Errorf("source = %q, want Blog", items[0].Source)
	}
	if items[0].Published.IsZero() {
		t.Error("item[0] should have a parsed publish time")
	}
	if !items[2].Published.IsZero() {
		t.Error("undated item should have zero publish time")
	}
}

func TestParseAtom(t *testing.T) {
	items, err := parseFeed(readFixture(t, "atom.xml"), "Privacy")
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Link != "https://atom.test/one" { // rel="alternate" preferred
		t.Errorf("item[0].Link = %q", items[0].Link)
	}
	if items[1].Link != "https://atom.test/two" { // only link present
		t.Errorf("item[1].Link = %q", items[1].Link)
	}
	if items[1].Published.IsZero() { // falls back to <updated>
		t.Error("item[1] should use <updated> as publish time")
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := parseFeed(readFixture(t, "malformed.xml"), "Bad"); err == nil {
		t.Error("expected error for malformed feed")
	}
}

func TestParseTime(t *testing.T) {
	for _, in := range []string{
		"Mon, 02 Jan 2006 15:04:05 -0700", // RFC1123Z
		"Tue, 03 Jan 2006 12:00:00 GMT",   // RFC1123
		"2023-05-01T10:00:00Z",            // RFC3339
		"02 Jan 06 15:04 MST",             // RFC822
	} {
		if parseTime(in).IsZero() {
			t.Errorf("parseTime(%q) returned zero", in)
		}
	}
	if !parseTime("not a date").IsZero() {
		t.Error("garbage date should parse to zero time")
	}
	if !parseTime("").IsZero() {
		t.Error("empty date should parse to zero time")
	}
}

func at(sec int) time.Time { return time.Unix(int64(sec), 0).UTC() }

func TestStoreMergeDedupeSort(t *testing.T) {
	s := NewStore(0) // default limit
	s.Merge("s1", []NewsItem{
		{Title: "Alpha", Link: "https://x/1", Published: at(300)},
		{Title: "Beta", Link: "https://x/2", Published: at(100)},
		{Title: "Alpha again", Link: "https://x/1", Published: at(200)}, // dup link
		{Title: "Undated", Link: "https://x/3"},                         // zero → last
	})

	all := s.All()
	if len(all) != 3 {
		t.Fatalf("got %d items, want 3 after dedupe: %+v", len(all), all)
	}
	if all[0].Link != "https://x/1" || all[1].Link != "https://x/2" || all[2].Link != "https://x/3" {
		t.Errorf("wrong order: %+v", all)
	}
	if all[2].Title != "Undated" {
		t.Error("zero-time item should sort last")
	}
}

func TestStorePerSourceLimit(t *testing.T) {
	s := NewStore(1)
	s.Merge("s1", []NewsItem{
		{Title: "New", Link: "l-new", Published: at(300)},
		{Title: "Old", Link: "l-old", Published: at(100)},
	})
	all := s.All()
	if len(all) != 1 || all[0].Link != "l-new" {
		t.Errorf("per-source limit not applied (kept newest): %+v", all)
	}
}

func TestStoreCrossSourceDedupe(t *testing.T) {
	s := NewStore(0)
	s.Merge("s1", []NewsItem{{Title: "Shared", Link: "https://dup", Published: at(200)}})
	s.Merge("s2", []NewsItem{{Title: "Shared", Link: "https://dup", Published: at(100)}})
	if all := s.All(); len(all) != 1 {
		t.Errorf("cross-source duplicate not collapsed: %+v", all)
	}
}

func TestStoreLatest(t *testing.T) {
	s := NewStore(0)
	s.Merge("s1", []NewsItem{
		{Title: "A", Link: "a", Published: at(300)},
		{Title: "B", Link: "b", Published: at(200)},
		{Title: "C", Link: "c", Published: at(100)},
	})
	if got := s.Latest(2); len(got) != 2 || got[0].Link != "a" || got[1].Link != "b" {
		t.Errorf("Latest(2) = %+v", got)
	}
	if got := s.Latest(0); got != nil {
		t.Errorf("Latest(0) = %+v, want nil", got)
	}
}

func TestStoreHealth(t *testing.T) {
	s := NewStore(0)
	s.Merge("ok", []NewsItem{{Title: "X", Link: "x", Published: at(1)}})
	s.SetError("bad", fmt.Errorf("boom"))

	h := s.Health()
	if len(h) != 2 {
		t.Fatalf("got %d health records, want 2", len(h))
	}
	// Sorted by source name: "bad" before "ok".
	if h[0].Source != "bad" || h[0].LastError != "boom" || h[0].LastErr.IsZero() {
		t.Errorf("bad health = %+v", h[0])
	}
	if h[1].Source != "ok" || h[1].ItemCount != 1 || h[1].LastOK.IsZero() {
		t.Errorf("ok health = %+v", h[1])
	}
}

func TestSchedulerRefreshFailSoft(t *testing.T) {
	sources := []Source{
		{Name: "RSS", FeedURL: "http://feeds/rss"},
		{Name: "Atom", FeedURL: "http://feeds/atom"},
		{Name: "Malformed", FeedURL: "http://feeds/bad"},
		{Name: "Down", FeedURL: "http://feeds/down"},
	}
	f := fixtureFetcher{
		files: map[string]string{
			"http://feeds/rss":  "rss.xml",
			"http://feeds/atom": "atom.xml",
			"http://feeds/bad":  "malformed.xml",
		},
		errs: map[string]error{
			"http://feeds/down": fmt.Errorf("connection refused"),
		},
	}
	store := NewStore(0)
	sch := newScheduler(sources, f, store, 0)

	sch.refreshAll(context.Background())

	// Healthy sources contributed their items; bad sources did not abort it.
	if got := len(store.All()); got != 5 { // 3 RSS + 2 Atom
		t.Errorf("merged %d items, want 5", got)
	}

	byName := map[string]SourceHealth{}
	for _, h := range store.Health() {
		byName[h.Source] = h
	}
	if byName["RSS"].ItemCount != 3 || byName["RSS"].LastOK.IsZero() {
		t.Errorf("RSS health = %+v", byName["RSS"])
	}
	if byName["Malformed"].LastError == "" || byName["Malformed"].LastErr.IsZero() {
		t.Errorf("Malformed should have recorded a parse error: %+v", byName["Malformed"])
	}
	if byName["Down"].LastError == "" {
		t.Errorf("Down should have recorded a fetch error: %+v", byName["Down"])
	}
}

func TestSchedulerStartStop(t *testing.T) {
	f := fixtureFetcher{files: map[string]string{"http://feeds/rss": "rss.xml"}}
	store := NewStore(0)
	sch := newScheduler([]Source{{Name: "RSS", FeedURL: "http://feeds/rss"}}, f, store, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	sch.Start(ctx)

	// The immediate refresh should populate the store shortly.
	deadline := time.Now().Add(2 * time.Second)
	for len(store.All()) == 0 {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("store not populated after Start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel() // graceful stop; must not panic or deadlock
	time.Sleep(10 * time.Millisecond)
}

func TestNewSchedulerEnforcesMinInterval(t *testing.T) {
	s := newScheduler(nil, fixtureFetcher{}, NewStore(0), time.Second)
	if s.interval != MinInterval {
		t.Errorf("interval = %v, want clamped to %v", s.interval, MinInterval)
	}
	if d := newScheduler(nil, fixtureFetcher{}, NewStore(0), 0).interval; d != DefaultInterval {
		t.Errorf("zero interval = %v, want default %v", d, DefaultInterval)
	}
}
