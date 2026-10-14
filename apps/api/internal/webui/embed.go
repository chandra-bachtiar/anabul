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

// Handler menyajikan hasil build frontend. Path yang bukan file dijawab
// index.html supaya routing SPA tetap jalan, kecuali permintaan aset.
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
			// index.html dan service worker tidak boleh di-cache lama: setelah
			// build baru, berkas lama menunjuk aset yang sudah tidak ada.
			if p == "index.html" || p == "sw.js" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		// Aset yang hilang dijawab 404, bukan index.html. Kalau HTML dikirim
		// sebagai jawaban permintaan .js atau .css, browser menolaknya dengan
		// error MIME dan halaman jadi kosong.
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

// isAssetRequest menandai permintaan berkas statis: folder build Vite, atau
// path apa pun yang punya ekstensi selain .html.
func isAssetRequest(p string) bool {
	if strings.HasPrefix(p, "_app/") {
		return true
	}
	ext := strings.ToLower(path.Ext(p))
	return ext != "" && ext != ".html"
}
