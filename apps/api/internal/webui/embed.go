package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

func init() {
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// Handler serves the frontend build, falling back to index.html for non-file
// paths so SPA routing works; asset requests are excluded.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}

	index, indexErr := fs.ReadFile(sub, "index.html")
	files := http.FileServerFS(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "metode tidak didukung", http.StatusMethodNotAllowed)
			return
		}

		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" || p == "." {
			p = "index.html"
		}

		if info, statErr := fs.Stat(sub, p); statErr == nil && !info.IsDir() {
			// index.html and the service worker must not be cached for long:
			// after a new build the old copy points at assets that are gone.
			if p == "index.html" || p == "sw.js" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		// A missing asset must be 404, never index.html: an HTML body for a .js
		// or .css request is rejected by the browser as a MIME error.
		if isAssetRequest(p) {
			http.NotFound(w, r)
			return
		}

		if indexErr != nil {
			http.Error(w, "frontend belum dibuild, jalankan `make dev-web` atau `make build`", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

// isAssetRequest reports static file requests: the Vite build folder, or any
// path with an extension other than .html.
func isAssetRequest(p string) bool {
	if strings.HasPrefix(p, "_app/") {
		return true
	}
	ext := strings.ToLower(path.Ext(p))
	return ext != "" && ext != ".html"
}
