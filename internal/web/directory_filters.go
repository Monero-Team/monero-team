package web

import (
	"net/url"
	"sort"

	"github.com/Monero-Team/monero-team/internal/directory"
)

// FilterOption is one selectable checkbox in a filter group. Field names match
// the filter-sidebar partial exactly.
type FilterOption struct {
	Value   string
	Label   string
	Count   int // global count in the whole dataset (independent of selection)
	Checked bool
}

// Filters groups the four filter dimensions in render order:
// category → status → kyc → access.
type Filters struct {
	Categories []FilterOption
	Statuses   []FilterOption
	KYC        []FilterOption
	Access     []FilterOption
}

// ActiveFilter is one removable pill above the list. Field names match the
// active-filters partial exactly.
type ActiveFilter struct {
	Label     string // e.g. "Category: wallet"
	RemoveURL string // current selection minus this value
}

// Canonical orders for the non-category dimensions. Categories come from the
// store (only those present).
var (
	canonicalStatuses = []string{
		directory.StatusVerified,
		directory.StatusAdmitted,
		directory.StatusQuestionable,
		directory.StatusScam,
	}
	canonicalAccess = []string{
		directory.AccessClearnet,
		directory.AccessTor,
		directory.AccessI2P,
	}
	canonicalKYC = []string{"no", "yes"}
)

// kycLabels maps the kyc query values to their display labels.
var kycLabels = map[string]string{"no": "No KYC", "yes": "KYC required"}

// selection holds the sanitized, enum-validated filter choices from the query.
type selection struct {
	category map[string]bool
	status   map[string]bool
	access   map[string]bool
	kyc      map[string]bool // "no" / "yes"
}

// parseSelection reads repeatable category/status/access/kyc params and keeps
// only values that belong to their enum; unknown values are silently dropped.
func parseSelection(q url.Values, validCategory map[string]bool) selection {
	return selection{
		category: keep(q["category"], validCategory),
		status:   keep(q["status"], setOf(canonicalStatuses)),
		access:   keep(q["access"], setOf(canonicalAccess)),
		kyc:      keep(q["kyc"], setOf(canonicalKYC)),
	}
}

func keep(values []string, valid map[string]bool) map[string]bool {
	out := make(map[string]bool)
	for _, v := range values {
		if valid[v] {
			out[v] = true
		}
	}
	return out
}

// active reports whether any dimension has a selected value.
func (s selection) active() bool {
	return len(s.category)+len(s.status)+len(s.access)+len(s.kyc) > 0
}

// match applies the filter: OR within a dimension, AND across dimensions. An
// empty dimension does not constrain.
func (s selection) match(r *directory.Resource) bool {
	if len(s.category) > 0 && !s.category[r.Category] {
		return false
	}
	if len(s.status) > 0 && !s.status[r.Status] {
		return false
	}
	if len(s.access) > 0 && !intersects(r.Access, s.access) {
		return false
	}
	if len(s.kyc) > 0 {
		ok := (s.kyc["no"] && !r.KYC) || (s.kyc["yes"] && r.KYC)
		if !ok {
			return false
		}
	}
	return true
}

// values rebuilds a canonical url.Values from the sanitized selection, so URLs
// the server emits never carry garbage params. url.Values.Encode sorts keys, so
// the output is deterministic regardless of insertion order.
func (s selection) values(categoriesInOrder []string) url.Values {
	q := url.Values{}
	for _, c := range categoriesInOrder {
		if s.category[c] {
			q.Add("category", c)
		}
	}
	for _, st := range canonicalStatuses {
		if s.status[st] {
			q.Add("status", st)
		}
	}
	for _, k := range canonicalKYC {
		if s.kyc[k] {
			q.Add("kyc", k)
		}
	}
	for _, a := range canonicalAccess {
		if s.access[a] {
			q.Add("access", a)
		}
	}
	return q
}

func intersects(have []string, want map[string]bool) bool {
	for _, v := range have {
		if want[v] {
			return true
		}
	}
	return false
}

func setOf(values []string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

// dirURL renders a /dir URL from a query value set; "/dir" when empty.
func dirURL(q url.Values) string {
	if enc := q.Encode(); enc != "" {
		return directoryPath + "?" + enc
	}
	return directoryPath
}

// removeURL returns the /dir URL for the current selection with one (key,value)
// pair removed.
func removeURL(current url.Values, key, value string) string {
	next := url.Values{}
	for k, vs := range current {
		for _, v := range vs {
			if k == key && v == value {
				continue
			}
			next.Add(k, v)
		}
	}
	return dirURL(next)
}

// buildDirectoryView assembles the full directory page model: the filtered
// rows, the sidebar options (with global counts), and the active-filter pills
// with their remove URLs. All URL building lives here, not in templates.
func buildDirectoryView(store *directory.Store, q url.Values) directoryView {
	all := store.All()
	categories := store.Categories() // present categories, sorted
	sel := parseSelection(q, setOf(categories))

	// Filter (store order is already canonical: name, slug).
	filtered := make([]*directory.Resource, 0, len(all))
	for _, r := range all {
		if sel.match(r) {
			filtered = append(filtered, r)
		}
	}
	rows := buildDirectoryRows(filtered)

	// Global counts across the whole dataset (independent of selection).
	catCount := map[string]int{}
	statusCount := map[string]int{}
	accessCount := map[string]int{}
	kycCount := map[string]int{"no": 0, "yes": 0}
	for _, r := range all {
		catCount[r.Category]++
		statusCount[r.Status]++
		seen := map[string]bool{}
		for _, a := range r.Access {
			if !seen[a] {
				accessCount[a]++
				seen[a] = true
			}
		}
		if r.KYC {
			kycCount["yes"]++
		} else {
			kycCount["no"]++
		}
	}

	// Sidebar options.
	filters := Filters{}
	for _, c := range categories {
		filters.Categories = append(filters.Categories, FilterOption{
			Value: c, Label: c, Count: catCount[c], Checked: sel.category[c],
		})
	}
	for _, st := range canonicalStatuses {
		if statusCount[st] == 0 {
			continue // only statuses present in the dataset
		}
		filters.Statuses = append(filters.Statuses, FilterOption{
			Value: st, Label: statusLabel(st), Count: statusCount[st], Checked: sel.status[st],
		})
	}
	for _, k := range canonicalKYC {
		filters.KYC = append(filters.KYC, FilterOption{
			Value: k, Label: kycLabels[k], Count: kycCount[k], Checked: sel.kyc[k],
		})
	}
	for _, a := range canonicalAccess {
		filters.Access = append(filters.Access, FilterOption{
			Value: a, Label: a, Count: accessCount[a], Checked: sel.access[a],
		})
	}

	// Active-filter pills, in dimension order, each with a remove URL built
	// from the sanitized selection.
	canonical := sel.values(categories)
	var active []ActiveFilter
	addActive := func(dim, key string, vals []string, selected map[string]bool) {
		for _, v := range vals {
			if selected[v] {
				active = append(active, ActiveFilter{
					Label:     dim + ": " + v,
					RemoveURL: removeURL(canonical, key, v),
				})
			}
		}
	}
	sortedCats := append([]string(nil), categories...)
	sort.Strings(sortedCats)
	addActive("Category", "category", sortedCats, sel.category)
	addActive("Status", "status", canonicalStatuses, sel.status)
	addActive("KYC", "kyc", canonicalKYC, sel.kyc)
	addActive("Access", "access", canonicalAccess, sel.access)

	return directoryView{
		Resources:     rows,
		Count:         len(rows),
		Active:        "directory",
		Filters:       filters,
		ActiveFilters: active,
		ClearURL:      directoryPath,
		ApplyAction:   directoryPath,
		ResultCount:   len(rows),
		Filtered:      sel.active(),
	}
}
