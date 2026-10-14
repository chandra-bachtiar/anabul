package webui

import (
	"strings"
	"testing"
)

func TestInlineScriptHashes(t *testing.T) {
	index := []byte(`<!doctype html><html><head>
		<script>window.__data = {"a":1}</script>
		<script src="/_app/immutable/entry/start.js"></script>
		<script type="module" src="/_app/immutable/entry/app.js"></script>
	</head><body></body></html>`)

	hashes := InlineScriptHashes(index)
	if len(hashes) != 1 {
		t.Fatalf("hash = %d, ingin 1 (hanya skrip inline)", len(hashes))
	}
	if !strings.HasPrefix(hashes[0], "'sha256-") || !strings.HasSuffix(hashes[0], "'") {
		t.Fatalf("format hash tidak sesuai CSP: %q", hashes[0])
	}

	if lagi := InlineScriptHashes(index); lagi[0] != hashes[0] {
		t.Fatal("hash harus stabil untuk isi yang sama")
	}
	if beda := InlineScriptHashes([]byte(`<script>window.x = 2</script>`)); beda[0] == hashes[0] {
		t.Fatal("isi berbeda harus menghasilkan hash berbeda")
	}
}

func TestInlineScriptHashesTidakAdaSkripInline(t *testing.T) {
	index := []byte(`<html><head><script src="/app.js"></script></head></html>`)
	if hashes := InlineScriptHashes(index); len(hashes) != 0 {
		t.Fatalf("hash = %d, ingin 0", len(hashes))
	}
}
