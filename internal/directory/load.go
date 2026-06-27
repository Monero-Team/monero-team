package directory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	countryRe = regexp.MustCompile(`^[A-Z]{2}$`)
)

// Load parses and validates every directory/*.json file in fsys, building a
// read-only Store. It validates each record independently and also checks
// cross-record invariants (unique slug, unique clearnet URL), aggregating every
// problem — each tagged with its file name and field — into a single returned
// error. If any record is invalid, no Store is returned.
//
// An empty dataset (no matching files) is not an error: it yields an empty
// Store.
func Load(fsys fs.FS) (*Store, error) {
	paths, err := fs.Glob(fsys, "directory/*.json")
	if err != nil {
		return nil, fmt.Errorf("directory: globbing dataset: %w", err)
	}
	sort.Strings(paths) // deterministic processing and error order

	var (
		errs          []error
		resources     []*Resource
		slugToFile    = make(map[string]string)
		clearnetToErr = make(map[string]string) // clearnet URL → first file seen
	)

	for _, p := range paths {
		file := path.Base(p)
		base := strings.TrimSuffix(file, ".json")

		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: reading file: %w", file, err))
			continue
		}

		var r Resource
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&r); err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid JSON: %v", file, err))
			continue
		}
		if dec.More() {
			errs = append(errs, fmt.Errorf("%s: invalid JSON: unexpected trailing data after object", file))
			continue
		}

		fieldErrs := validate(&r, base)

		// Cross-record: globally unique slug.
		if r.Slug != "" {
			if prev, ok := slugToFile[r.Slug]; ok {
				fieldErrs = append(fieldErrs, FieldError{"slug", fmt.Sprintf("duplicate slug %q (also defined in %s)", r.Slug, prev)})
			} else {
				slugToFile[r.Slug] = file
			}
		}

		// Cross-record: no two resources may share a clearnet URL.
		if c := normalizedClearnet(&r); c != "" {
			if prev, ok := clearnetToErr[c]; ok {
				fieldErrs = append(fieldErrs, FieldError{"links.clearnet", fmt.Sprintf("duplicate clearnet URL %q (also used by %s)", c, prev)})
			} else {
				clearnetToErr[c] = file
			}
		}

		for _, fe := range fieldErrs {
			errs = append(errs, fmt.Errorf("%s: %w", file, fe))
		}
		if len(fieldErrs) == 0 {
			resources = append(resources, &r)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("directory: %d validation error(s):\n%w", len(errs), errors.Join(errs...))
	}
	return newStore(resources), nil
}

// FieldError is a validation problem tied to a specific field. Field uses the
// resource's path (e.g. "name", "links.clearnet", "access"); the loader
// prefixes it with the file name, and the /submit form maps it to its inputs.
type FieldError struct {
	Field string
	Msg   string
}

func (e FieldError) Error() string { return e.Field + ": " + e.Msg }

// validate checks a single resource against every per-record rule, including
// the loader-only rules (slug format/file-name equality and the status enum).
// Returned errors carry no file prefix; the caller adds it.
func validate(r *Resource, fileBase string) []FieldError {
	errs := checkCommon(r)

	// slug: format and equality with the file name (loader-only — a submission
	// has no file yet).
	switch {
	case r.Slug == "":
		errs = append(errs, FieldError{"slug", "must not be empty"})
	default:
		if !slugRe.MatchString(r.Slug) {
			errs = append(errs, FieldError{"slug", fmt.Sprintf("%q must match ^[a-z0-9]+(-[a-z0-9]+)*$", r.Slug)})
		}
		if r.Slug != fileBase {
			errs = append(errs, FieldError{"slug", fmt.Sprintf("%q must equal the file name (%q)", r.Slug, fileBase)})
		}
	}

	// status enum (loader-only — a submission's status is assigned by a
	// moderator, so it is not part of the shared rules).
	if !validStatus[r.Status] {
		errs = append(errs, FieldError{"status", fmt.Sprintf("%q is not a valid status", r.Status)})
	}

	return errs
}

// ValidateSubmission validates a user-submitted resource using the same rules
// as the dataset loader, minus the two loader-only rules: the slug/file-name
// equality (a submission has no file; its slug is generated) and the status
// enum (status is assigned later by a moderator). It returns field-keyed
// errors; an empty result means the entry is valid.
func ValidateSubmission(r *Resource) []FieldError {
	return checkCommon(r)
}

// checkCommon runs every per-record rule shared by the loader and the
// submission path: name, description, category, access, country, links (and
// access↔links consistency), and tags.
func checkCommon(r *Resource) []FieldError {
	var errs []FieldError
	add := func(field, format string, args ...any) {
		errs = append(errs, FieldError{field, fmt.Sprintf(format, args...)})
	}

	// name
	switch n := utf8.RuneCountInString(r.Name); {
	case r.Name == "":
		add("name", "must not be empty")
	case n > maxNameLen:
		add("name", "must be at most %d characters (got %d)", maxNameLen, n)
	}

	// description
	switch n := utf8.RuneCountInString(r.Description); {
	case r.Description == "":
		add("description", "must not be empty")
	case n > maxDescriptionLen:
		add("description", "must be at most %d characters (got %d)", maxDescriptionLen, n)
	}

	// category enum
	if !validCategory[r.Category] {
		add("category", "%q is not a valid category", r.Category)
	}

	// access: non-empty, valid enum, no duplicates
	if len(r.Access) == 0 {
		add("access", "must list at least one access type")
	} else {
		seen := make(map[string]bool, len(r.Access))
		for _, a := range r.Access {
			if !validAccess[a] {
				add("access", "%q is not a valid access type", a)
			}
			if seen[a] {
				add("access", "duplicate access type %q", a)
			}
			seen[a] = true
		}
	}

	// country: null or ISO-3166-1 alpha-2
	if r.Country != nil && !countryRe.MatchString(*r.Country) {
		add("country", "%q must be a 2-letter ISO-3166-1 alpha-2 code", *r.Country)
	}

	// links: at least one present
	lnClear := nonEmpty(r.Links.Clearnet)
	lnOnion := nonEmpty(r.Links.Onion)
	lnI2P := nonEmpty(r.Links.I2P)
	if !lnClear && !lnOnion && !lnI2P {
		add("links", "at least one non-empty link is required")
	}

	// link formats
	if lnClear && !validHTTPURL(*r.Links.Clearnet) {
		add("links.clearnet", "%q must be a valid http(s) URL", *r.Links.Clearnet)
	}
	if lnOnion && !strings.HasSuffix(strings.TrimSpace(*r.Links.Onion), ".onion") {
		add("links.onion", "%q must end with .onion", *r.Links.Onion)
	}
	if lnI2P && !strings.HasSuffix(strings.TrimSpace(*r.Links.I2P), ".i2p") {
		add("links.i2p", "%q must end with .i2p", *r.Links.I2P)
	}

	// access ↔ links consistency (each access type requires its link, and each
	// present link requires its access type).
	accClear := hasAccess(r.Access, AccessClearnet)
	accTor := hasAccess(r.Access, AccessTor)
	accI2P := hasAccess(r.Access, AccessI2P)
	if accClear && !lnClear {
		add("access", `lists "clearnet" but links.clearnet is missing`)
	}
	if lnClear && !accClear {
		add("links.clearnet", `present but access does not list "clearnet"`)
	}
	if accTor && !lnOnion {
		add("access", `lists "tor" but links.onion is missing`)
	}
	if lnOnion && !accTor {
		add("links.onion", `present but access does not list "tor"`)
	}
	if accI2P && !lnI2P {
		add("access", `lists "i2p" but links.i2p is missing`)
	}
	if lnI2P && !accI2P {
		add("links.i2p", `present but access does not list "i2p"`)
	}

	// tags: lowercase, non-empty, de-duplicated
	seenTag := make(map[string]bool, len(r.Tags))
	for i, t := range r.Tags {
		switch {
		case t == "":
			add("tags", "entry %d must not be empty", i)
		case t != strings.ToLower(t):
			add("tags", "%q must be lowercase", t)
		}
		if seenTag[t] {
			add("tags", "duplicate tag %q", t)
		}
		seenTag[t] = true
	}

	return errs
}

// normalizedClearnet returns the trimmed clearnet URL used for cross-record
// duplicate detection, or "" if there is none.
func normalizedClearnet(r *Resource) string {
	if !nonEmpty(r.Links.Clearnet) {
		return ""
	}
	return strings.TrimSpace(*r.Links.Clearnet)
}

func nonEmpty(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}

func hasAccess(access []string, want string) bool {
	for _, a := range access {
		if a == want {
			return true
		}
	}
	return false
}

// validHTTPURL reports whether s is a syntactically valid absolute http(s) URL.
func validHTTPURL(s string) bool {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
