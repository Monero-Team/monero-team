package directory_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Monero-Team/monero-team/data"
	"github.com/Monero-Team/monero-team/internal/directory"
)

// TestValidDatasetLoads checks that a well-formed fixture dataset loads and that
// the store's count, ordering, and accessors behave as specified.
func TestValidDatasetLoads(t *testing.T) {
	s, err := directory.Load(os.DirFS("testdata/valid"))
	if err != nil {
		t.Fatalf("Load(valid): unexpected error: %v", err)
	}

	if got, want := s.Len(), 3; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}

	// BySlug: present and absent.
	if r, ok := s.BySlug("alpha-wallet"); !ok {
		t.Error(`BySlug("alpha-wallet"): not found`)
	} else if r.Name != "Alpha wallet" {
		t.Errorf(`BySlug("alpha-wallet").Name = %q, want "Alpha wallet"`, r.Name)
	}
	if _, ok := s.BySlug("does-not-exist"); ok {
		t.Error(`BySlug("does-not-exist"): unexpectedly found`)
	}

	// All() is sorted by name (ties by slug).
	all := s.All()
	gotOrder := make([]string, len(all))
	for i, r := range all {
		gotOrder[i] = r.Slug
	}
	wantOrder := []string{"alpha-wallet", "beta-service", "gamma-node"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Errorf("All() order = %v, want %v", gotOrder, wantOrder)
	}

	// Category / status indexes.
	if got := len(s.ByCategory(directory.CategoryWallet)); got != 1 {
		t.Errorf("ByCategory(wallet) = %d, want 1", got)
	}
	if got := len(s.ByStatus(directory.StatusVerified)); got != 1 {
		t.Errorf("ByStatus(verified) = %d, want 1", got)
	}

	// Categories(): distinct present categories in canonical order.
	wantCats := []string{directory.CategoryWallet, directory.CategoryService, directory.CategoryNode}
	if got := strings.Join(s.Categories(), ","); got != strings.Join(wantCats, ",") {
		t.Errorf("Categories() = %v, want %v", s.Categories(), wantCats)
	}
}

// TestEmptyDatasetIsNotError checks that zero files is a valid, empty store.
func TestEmptyDatasetIsNotError(t *testing.T) {
	s, err := directory.Load(os.DirFS("testdata/empty"))
	if err != nil {
		t.Fatalf("Load(empty): unexpected error: %v", err)
	}
	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
	if s.All() != nil {
		t.Errorf("All() = %v, want nil", s.All())
	}
}

// TestInvalidFixtures checks that each malformed fixture fails, and that the
// aggregated error names both the offending file and the relevant field/rule.
func TestInvalidFixtures(t *testing.T) {
	cases := []struct {
		dir     string // under testdata/invalid/
		wantSub []string
	}{
		{"missing-name", []string{"missing-name.json", "name"}},
		{"bad-enum", []string{"bad-enum.json", "category"}},
		{"dup-slug", []string{"dup-copy.json", "duplicate slug"}},
		{"onion-no-suffix", []string{"onion-no-suffix.json", "links.onion", "must end with .onion"}},
		{"clearnet-no-link", []string{"clearnet-no-link.json", `links.clearnet is missing`}},
	}

	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			_, err := directory.Load(os.DirFS("testdata/invalid/" + c.dir))
			if err == nil {
				t.Fatalf("Load(invalid/%s): expected error, got nil", c.dir)
			}
			msg := err.Error()
			for _, sub := range c.wantSub {
				if !strings.Contains(msg, sub) {
					t.Errorf("error missing %q\nfull error:\n%s", sub, msg)
				}
			}
		})
	}
}

// TestRealDataValid loads the real embedded dataset. It runs in CI, so a bad
// data PR fails the build rather than reaching production. It also enforces the
// moderation guardrail: no real (non-fictional) entity may exceed "admitted".
func TestRealDataValid(t *testing.T) {
	s, err := directory.Load(data.Files)
	if err != nil {
		t.Fatalf("Load(real data): %v", err)
	}
	if s.Len() == 0 {
		t.Fatal("real dataset is empty")
	}

	if r, ok := s.BySlug("getmonero-org"); !ok {
		t.Error(`expected "getmonero-org" in real dataset`)
	} else if r.Status != directory.StatusAdmitted {
		t.Errorf("getmonero-org status = %q, want %q", r.Status, directory.StatusAdmitted)
	}

	// Guardrail: only explicitly fictional placeholders (slug "example-*") may
	// carry a status other than "admitted".
	for _, r := range s.All() {
		if strings.HasPrefix(r.Slug, "example-") {
			continue
		}
		if r.Status != directory.StatusAdmitted {
			t.Errorf("real entity %q has status %q; real entities must be %q without verification",
				r.Slug, r.Status, directory.StatusAdmitted)
		}
	}
}
