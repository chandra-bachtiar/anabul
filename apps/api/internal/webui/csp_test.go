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
		t.Fatalf("hashes = %d, want 1 (inline scripts only)", len(hashes))
	}
	if !strings.HasPrefix(hashes[0], "'sha256-") || !strings.HasSuffix(hashes[0], "'") {
		t.Fatalf("hash does not match the CSP format: %q", hashes[0])
	}

	if again := InlineScriptHashes(index); again[0] != hashes[0] {
		t.Fatal("the hash must be stable for identical content")
	}
	if different := InlineScriptHashes([]byte(`<script>window.x = 2</script>`)); different[0] == hashes[0] {
		t.Fatal("different content must produce a different hash")
	}
}

func TestInlineScriptHashesWithoutInlineScripts(t *testing.T) {
	index := []byte(`<html><head><script src="/app.js"></script></head></html>`)
	if hashes := InlineScriptHashes(index); len(hashes) != 0 {
		t.Fatalf("hashes = %d, want 0", len(hashes))
	}
}
