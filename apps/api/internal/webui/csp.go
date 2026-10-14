package webui

import (
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"regexp"
)

var (
	// scriptPattern memisahkan atribut tag dan isi setiap skrip.
	scriptPattern = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	srcAttribute  = regexp.MustCompile(`(?i)\bsrc\s*=`)
)

// Hashes menghitung hash CSP untuk skrip inline di index.html yang di-embed.
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

// InlineScriptHashes menghitung hash SHA-256 (format CSP) untuk setiap skrip
// inline. SvelteKit menyisipkan satu skrip inline saat hydration, jadi CSP tidak
// cukup memakai 'self' dan tidak perlu turun ke 'unsafe-inline'.
func InlineScriptHashes(index []byte) []string {
	hasil := make([]string, 0, 1)
	for _, match := range scriptPattern.FindAllSubmatch(index, -1) {
		if srcAttribute.Match(match[1]) {
			continue
		}
		isi := match[2]
		if len(isi) == 0 {
			continue
		}
		ringkas := sha256.Sum256(isi)
		hasil = append(hasil, "'sha256-"+base64.StdEncoding.EncodeToString(ringkas[:])+"'")
	}
	return hasil
}
