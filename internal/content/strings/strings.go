// Package strings is the single source of truth for user-facing UI copy.
//
// All English ("en" locale) text rendered by templates lives here so that a
// future i18n layer can swap the active locale without touching templates or
// handlers. Nothing in this package performs I/O or pulls external data.
package strings

// Locale identifies the active UI language. Phase 1 ships English only.
const Locale = "en"

// Branding holds product-level naming and taglines.
type Branding struct {
	Name    string
	Tagline string
}

// Brand is the active product branding.
var Brand = Branding{
	Name:    "Monero Team",
	Tagline: "A privacy-first index of the Monero ecosystem.",
}

// Section is one navigable area of the site. Path is the canonical URL and is
// also used server-side to resolve the active navbar item.
type Section struct {
	Path  string
	Label string
	// Summary is the one-line description shown on the section's skeleton page.
	Summary string
}

// Nav is the primary navigation. It contains exactly the five top-level
// sections; the homepage is reached via the wordmark, and utility pages live
// in the footer. Order here is the order rendered in the navbar.
var Nav = []Section{
	{Path: "/dir", Label: "Directory", Summary: "A curated, audited index of Monero wallets, services, and tools."},
	{Path: "/news", Label: "News", Summary: "Signal over noise — developments across the Monero ecosystem."},
	{Path: "/digest", Label: "Digest", Summary: "Periodic summaries of what changed and why it matters."},
	{Path: "/exchanges", Label: "Exchanges", Summary: "Where to acquire and trade XMR, with privacy trade-offs noted."},
	{Path: "/explorer", Label: "Explorer", Summary: "Inspect the Monero blockchain without third-party trackers."},
}

// Utility holds secondary sections that exist as routes but are not part of the
// five-item primary navbar. They are surfaced in the footer.
var Utility = []Section{
	{Path: "/mining", Label: "Mining", Summary: "Pools, solo mining, and decentralization of hash power."},
	{Path: "/tools", Label: "Tools", Summary: "Self-hostable utilities for working with Monero."},
}

// HomeText holds copy specific to the landing page.
type HomeText struct {
	Title       string
	Heading     string
	Lede        string
	SectionsLed string
}

// Home is the active landing-page copy.
var Home = HomeText{
	Title:       "Monero Team",
	Heading:     "The Monero ecosystem, audited and ad-free.",
	Lede:        "An open, self-hosted index built privacy-first: no JavaScript required, no cookies, no third-party requests. Everything here is served from a single auditable binary.",
	SectionsLed: "Explore the sections",
}

// ComingSoonText holds copy for not-yet-populated skeleton pages.
type ComingSoonText struct {
	Badge string
	Body  string
}

// ComingSoon is the active skeleton-page copy.
var ComingSoon = ComingSoonText{
	Badge: "Coming soon",
	Body:  "This section is being built. The reading and navigation experience works without JavaScript, and nothing on this page reaches an external server.",
}

// FooterText holds footer copy and privacy assurances.
type FooterText struct {
	PrivacyNote string
	NoJS        string
	NoCookies   string
	SelfHosted  string
	Sections    string
	More        string
}

// Footer is the active footer copy.
var Footer = FooterText{
	PrivacyNote: "No tracking. No cookies. No third-party requests.",
	NoJS:        "Works without JavaScript",
	NoCookies:   "No cookies",
	SelfHosted:  "Self-hosted & auditable",
	Sections:    "Sections",
	More:        "More",
}

// MetaText holds document-level metadata copy.
type MetaText struct {
	Description string
	SkipToMain  string
}

// Meta is the active document metadata copy.
var Meta = MetaText{
	Description: "A privacy-first, self-hosted index of the Monero ecosystem.",
	SkipToMain:  "Skip to main content",
}
