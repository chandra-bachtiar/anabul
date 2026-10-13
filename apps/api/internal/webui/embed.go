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

// Handler menyajikan hasil build frontend. Semua path yang bukan file akan
// dijawab index.html agar routing SPA tetap jalan.
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

		info, statErr := fs.Stat(sub, p)
		if statErr == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}

		if indexErr != nil {
			http.Error(w, "frontend belum dibuild, jalankan `make dev-web` atau `make build`", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
