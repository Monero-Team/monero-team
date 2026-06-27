package web

import (
	"strings"

	"github.com/Monero-Team/monero-team/internal/directory"
)

// directoryRow is the presentation model consumed by the directory-row /
// directory-list partials. Field names match the partials exactly; all display
// logic (primary URL, status label, KYC/access text) is computed here so the
// templates stay logic-free.
type directoryRow struct {
	Status      string // raw: verified | admitted | questionable | scam
	StatusLabel string // sentence case: Verified | Admitted | Questionable | Scam
	Name        string
	URL         string // primary external link (clearnet → onion → i2p)
	Slug        string
	Country     string // 2-letter code, or "" when unknown
	Desc        string
	KYC         string // "No KYC" or "KYC required"
	Access      string // access types joined by " · "
	Tags        []string
	Rated       bool   // always false for now (no ratings yet)
	RatingText  string // "" for now
}

// directoryView is the page model passed to the directory page and its
// partials (directory-list, filter-sidebar, active-filters, dir-empty-match).
type directoryView struct {
	Resources []directoryRow
	Count     int
	Active    string

	// Filtering (Stage 4) and search (Stage 6).
	Filters       Filters
	ActiveFilters []ActiveFilter
	Search        Search
	ClearURL      string
	ApplyAction   string
	ResultCount   int
	Filtered      bool // true when any filter or the search query is active
}

// buildDirectoryRows maps the store's resources (already sorted by name, slug)
// into presentation rows.
func buildDirectoryRows(rs []*directory.Resource) []directoryRow {
	rows := make([]directoryRow, 0, len(rs))
	for _, r := range rs {
		rows = append(rows, directoryRow{
			Status:      r.Status,
			StatusLabel: statusLabel(r.Status),
			Name:        r.Name,
			URL:         primaryURL(r),
			Slug:        r.Slug,
			Country:     country(r),
			Desc:        r.Description,
			KYC:         kycLabel(r.KYC),
			Access:      strings.Join(r.Access, " · "),
			Tags:        r.Tags,
			Rated:       false,
			RatingText:  "",
		})
	}
	return rows
}

// primaryURL selects the resource's primary link by priority:
// clearnet → onion → i2p (first non-empty). Stage 2 validation guarantees at
// least one link is present.
func primaryURL(r *directory.Resource) string {
	switch {
	case nonEmpty(r.Links.Clearnet):
		return *r.Links.Clearnet
	case nonEmpty(r.Links.Onion):
		return *r.Links.Onion
	case nonEmpty(r.Links.I2P):
		return *r.Links.I2P
	}
	return ""
}

// statusLabel returns the sentence-case label for a status (e.g. "verified" →
// "Verified"). Statuses are ASCII enum values.
func statusLabel(status string) string {
	if status == "" {
		return ""
	}
	return strings.ToUpper(status[:1]) + status[1:]
}

func kycLabel(kyc bool) string {
	if kyc {
		return "KYC required"
	}
	return "No KYC"
}

func country(r *directory.Resource) string {
	if r.Country != nil {
		return *r.Country
	}
	return ""
}

func nonEmpty(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}
