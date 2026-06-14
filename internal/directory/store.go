package directory

import "sort"

// Store is an immutable, read-only view of the directory, built once by Load.
// All accessors return defensive copies of their slices, so callers cannot
// mutate the store's internal ordering. The pointed-to Resource values are
// likewise meant to be treated as read-only.
type Store struct {
	all        []*Resource          // sorted by (name, slug)
	bySlug     map[string]*Resource // slug → resource
	byCategory map[string][]*Resource
	byStatus   map[string][]*Resource
}

// newStore builds a Store from validated resources, establishing the canonical
// (name, slug) ordering used by every accessor.
func newStore(resources []*Resource) *Store {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Name != resources[j].Name {
			return resources[i].Name < resources[j].Name
		}
		return resources[i].Slug < resources[j].Slug
	})

	s := &Store{
		all:        resources,
		bySlug:     make(map[string]*Resource, len(resources)),
		byCategory: make(map[string][]*Resource),
		byStatus:   make(map[string][]*Resource),
	}
	for _, r := range resources {
		s.bySlug[r.Slug] = r
		s.byCategory[r.Category] = append(s.byCategory[r.Category], r)
		s.byStatus[r.Status] = append(s.byStatus[r.Status], r)
	}
	return s
}

// Len reports the number of resources in the store.
func (s *Store) Len() int { return len(s.all) }

// All returns every resource, sorted by name (ties broken by slug).
func (s *Store) All() []*Resource {
	return clone(s.all)
}

// BySlug returns the resource with the given slug, if present.
func (s *Store) BySlug(slug string) (*Resource, bool) {
	r, ok := s.bySlug[slug]
	return r, ok
}

// ByCategory returns the resources in the given category, in canonical order.
func (s *Store) ByCategory(cat string) []*Resource {
	return clone(s.byCategory[cat])
}

// ByStatus returns the resources with the given status, in canonical order.
func (s *Store) ByStatus(st string) []*Resource {
	return clone(s.byStatus[st])
}

// Categories returns the distinct categories present in the store, in the
// canonical Categories order. Empty categories are omitted.
func (s *Store) Categories() []string {
	present := make([]string, 0, len(Categories))
	for _, c := range Categories {
		if len(s.byCategory[c]) > 0 {
			present = append(present, c)
		}
	}
	return present
}

// clone returns a shallow copy of a resource-pointer slice so callers cannot
// reorder or otherwise mutate the store's internal slices.
func clone(xs []*Resource) []*Resource {
	if len(xs) == 0 {
		return nil
	}
	out := make([]*Resource, len(xs))
	copy(out, xs)
	return out
}
