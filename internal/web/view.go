package web

import "github.com/Monero-Team/monero-team/internal/content/strings"

// view is the data model passed to every template render. It bundles the
// shared chrome (brand, navigation, locale) with page-specific fields. Fields
// are populated from the strings package so templates never embed literal copy.
type view struct {
	Locale     string
	Brand      strings.Branding
	Meta       strings.MetaText
	Nav        []strings.Section
	Utility    []strings.Section
	Footer     strings.FooterText
	ActivePath string

	// Page-specific.
	Home       strings.HomeText
	Section    strings.Section
	ComingSoon strings.ComingSoonText
	Directory  directoryView
}

// newView builds a view with the shared chrome populated. activePath is the
// canonical path used server-side to mark the active navbar item.
func newView(activePath string) view {
	return view{
		Locale:     strings.Locale,
		Brand:      strings.Brand,
		Meta:       strings.Meta,
		Nav:        strings.Nav,
		Utility:    strings.Utility,
		Footer:     strings.Footer,
		ActivePath: activePath,
	}
}
