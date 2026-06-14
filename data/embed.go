// Package data embeds the version-controlled directory dataset into the binary.
//
// The dataset is community-curated JSON, one file per resource under
// directory/. It is released under CC0 1.0 (see LICENSE) so it can be reused
// freely. Nothing here is fetched at build or run time — the files are compiled
// into the binary, consistent with the project's single-auditable-binary goal.
package data

import "embed"

// Files holds the embedded directory dataset. Only directory/*.json is
// embedded; the directory's README.md and the top-level LICENSE are not, so the
// loader can glob directory/*.json without filtering non-resource files.
//
//go:embed directory/*.json
var Files embed.FS
