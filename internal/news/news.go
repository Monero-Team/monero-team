// Package news collects headlines from a curated set of Monero/privacy RSS and
// Atom feeds into an in-memory, read-only store. It depends only on the Go
// standard library.
//
// Privacy/copyright posture: only the headline, source, publish time, and link
// are kept — never the article body, description, or excerpt. The store holds
// pointers to where to read, not the content itself.
package news

import "time"

// NewsItem is one headline. These four fields are the only data retained from a
// feed entry; article bodies/descriptions are deliberately discarded.
type NewsItem struct {
	Title     string
	Source    string
	Published time.Time
	Link      string
}

// Source is one curated feed.
type Source struct {
	Name    string
	FeedURL string
}

// Sources is the maintainer-curated list of feeds. It is intentionally small
// and conservative: only well-known, official Monero/privacy feeds belong here.
//
// Maintainers: add or correct entries here. Do not add a URL you cannot verify
// — leave a TODO instead. The test suite does not depend on this list (tests
// drive the parser/store/scheduler with local fixtures), so an empty or
// partial list never breaks CI.
var Sources = []Source{
	// Official Monero project blog (getmonero.org). Verified canonical feed.
	{Name: "Monero Project", FeedURL: "https://www.getmonero.org/feed.xml"},

	// TODO(maintainer): add further verified feeds, e.g. the Monero subreddit
	// RSS, Monero Observer, or privacy-focused outlets — only once each feed
	// URL has been confirmed. Do not guess URLs.
}
