package web

import (
	"strings"

	"github.com/Monero-Team/monero-team/internal/directory"
)

// ResourceLink is one labelled external link on a resource detail page. Field
// names match the resource-detail partial.
type ResourceLink struct {
	Label string
	URL   string
}

// resourceDetail is the presentation model for /dir/<slug>. All display logic
// (status label, sentence-case category, link labelling) is computed here so
// the template stays logic-free. Field names match the resource-detail partial.
type resourceDetail struct {
	Name        string
	Status      string // raw: verified | admitted | questionable | scam
	StatusLabel string // sentence case
	Category    string // sentence case
	SubCategory string // unused for now (no sub-categories in the data model)
	Country     string // 2-letter code, or ""
	Desc        string
	KYC         string // "No KYC" | "KYC required"
	Access      string // access types joined by " · "
	Links       []ResourceLink
	Tags        []string
}

// buildResourceDetail maps a stored resource into its detail view.
func buildResourceDetail(r *directory.Resource) resourceDetail {
	return resourceDetail{
		Name:        r.Name,
		Status:      r.Status,
		StatusLabel: statusLabel(r.Status),
		Category:    capitalizeFirst(r.Category),
		SubCategory: "",
		Country:     country(r),
		Desc:        r.Description,
		KYC:         kycLabel(r.KYC),
		Access:      strings.Join(r.Access, " · "),
		Links:       resourceLinks(r),
		Tags:        r.Tags,
	}
}

// resourceLinks lists every present link as a (Label, URL) pair, in the order
// Clearnet → Tor → I2P.
func resourceLinks(r *directory.Resource) []ResourceLink {
	var links []ResourceLink
	if nonEmpty(r.Links.Clearnet) {
		links = append(links, ResourceLink{Label: "Clearnet", URL: *r.Links.Clearnet})
	}
	if nonEmpty(r.Links.Onion) {
		links = append(links, ResourceLink{Label: "Tor (.onion)", URL: *r.Links.Onion})
	}
	if nonEmpty(r.Links.I2P) {
		links = append(links, ResourceLink{Label: "I2P", URL: *r.Links.I2P})
	}
	return links
}

// capitalizeFirst upper-cases the first byte of an ASCII enum value (e.g.
// "wallet" → "Wallet").
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
