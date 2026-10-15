package webui

import (
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"regexp"
)

var (
	// scriptPattern splits each script tag's attributes from its body.
	scriptPattern = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	srcAttribute  = regexp.MustCompile(`(?i)\bsrc\s*=`)
)

// Hashes returns CSP hashes for the inline scripts in the embedded index.html.
func Hashes() []string {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil
	}
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil
	}
	return InlineScriptHashes(index)
}

// InlineScriptHashes returns the SHA-256 hash (CSP format) of every inline
// script. SvelteKit injects one at hydration, so 'self' is not enough.
func InlineScriptHashes(index []byte) []string {
	hashes := make([]string, 0, 1)
	for _, match := range scriptPattern.FindAllSubmatch(index, -1) {
		if srcAttribute.Match(match[1]) {
			continue
		}
		body := match[2]
		if len(body) == 0 {
			continue
		}
		digest := sha256.Sum256(body)
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(digest[:])+"'")
	}
	return hashes
}
