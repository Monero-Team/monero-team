package web

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"strings"
)

// assetHashLen is how many hex characters of the sha256 we keep. 10 hex chars
// (40 bits) is ample to detect content changes for a handful of static files.
const assetHashLen = 10

// assetVersions maps an asset path relative to the assets root (e.g.
// "app.css", "fonts/inter-tight-400.woff2") to a short hash of its bytes.
// It is computed once at package init from the embedded FS; the hash changes
// whenever the file's content changes, which is exactly what makes the
// aggressive immutable cache header correct — a new build with new CSS yields a
// new URL, so stale caches are bypassed automatically.
var assetVersions = mustHashAssets()

func mustHashAssets() map[string]string {
	versions, err := hashAssets(assetsFS, "assets")
	if err != nil {
		// assetsFS is embedded at build time; a failure here is a programmer
		// error, not a runtime condition.
		panic(err)
	}
	return versions
}

// hashAssets walks fsys under root and returns a content hash per file, keyed by
// the path relative to root.
func hashAssets(fsys fs.FS, root string) (map[string]string, error) {
	versions := make(map[string]string)
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		versions[rel] = shortHash(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// shortHash returns the first assetHashLen hex characters of the sha256 of b.
func shortHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:assetHashLen]
}

// assetURL returns the cache-busting URL for an embedded asset: the stable
// "/static/<name>" path with a "?v=<hash>" query whose value tracks the file's
// content. Templates call this via the "asset" function. Unknown names fall
// back to the unversioned path rather than failing the render.
//
// Fonts are intentionally referenced unversioned (from @font-face in app.css):
// their bytes never change, so the immutable cache is already correct for them.
func assetURL(name string) string {
	if v, ok := assetVersions[name]; ok {
		return "/static/" + name + "?v=" + v
	}
	return "/static/" + name
}
