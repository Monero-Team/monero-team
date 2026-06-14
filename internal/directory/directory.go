// Package directory defines the Monero Team resource directory: the data model,
// a hand-written validator, and a read-only in-memory store loaded once at
// startup from embedded JSON. It depends only on the standard library.
package directory

// Resource is one directory entry. One JSON file under data/directory/ maps to
// exactly one Resource; the file name (without .json) must equal Slug.
type Resource struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Status      string   `json:"status"`
	Country     *string  `json:"country"` // nullable; ISO-3166-1 alpha-2 when set
	Access      []string `json:"access"`
	KYC         bool     `json:"kyc"`
	Tags        []string `json:"tags"`
	Links       Links    `json:"links"`
	Description string   `json:"description"`
}

// Links holds the reachability endpoints for a resource. Each is nullable; at
// least one must be present, and the set must be consistent with Access.
type Links struct {
	Clearnet *string `json:"clearnet"`
	Onion    *string `json:"onion"`
	I2P      *string `json:"i2p"`
}

// Field-length limits.
const (
	maxNameLen        = 120
	maxDescriptionLen = 280
)

// Category enum — the kind of resource.
const (
	CategoryWallet      = "wallet"
	CategoryExchange    = "exchange"
	CategoryMerchant    = "merchant"
	CategoryMining      = "mining"
	CategoryVPN         = "vpn"
	CategoryService     = "service"
	CategoryTool        = "tool"
	CategoryEducational = "educational"
	CategoryNode        = "node"
)

// Status enum — the moderation state. Real, named entities must never be
// assigned a status above "admitted" without verification (see the package
// README and data/directory/README.md).
const (
	StatusVerified     = "verified"
	StatusAdmitted     = "admitted"
	StatusQuestionable = "questionable"
	StatusScam         = "scam"
)

// Access enum — how the resource can be reached.
const (
	AccessClearnet = "clearnet"
	AccessTor      = "tor"
	AccessI2P      = "i2p"
)

// Enum value lists, in canonical order. These are the single source of truth;
// validation and any future UI must read from them.
var (
	Categories = []string{
		CategoryWallet, CategoryExchange, CategoryMerchant, CategoryMining,
		CategoryVPN, CategoryService, CategoryTool, CategoryEducational, CategoryNode,
	}
	Statuses = []string{
		StatusVerified, StatusAdmitted, StatusQuestionable, StatusScam,
	}
	AccessTypes = []string{
		AccessClearnet, AccessTor, AccessI2P,
	}
)

// validCategory, validStatus, validAccess are membership sets derived from the
// canonical lists above.
var (
	validCategory = sliceToSet(Categories)
	validStatus   = sliceToSet(Statuses)
	validAccess   = sliceToSet(AccessTypes)
)

func sliceToSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
