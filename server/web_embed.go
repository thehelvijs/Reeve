package main

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"strings"
)

// The PWA manifest must be served as application/manifest+json; Go's mime table
// does not know this extension by default.
func init() {
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// webFS holds the built React UI. `make web-build` copies web/dist into
// server/webdist before release builds; a placeholder index.html keeps `go
// build` working in dev when the bundle is absent.
//
//go:embed all:webdist
var webFS embed.FS

// assetCacheControl caches the bundle forever and the shell never: Vite stamps
// a content hash into every name under assets/, so those URLs never change
// meaning, while index.html must be re-fetched to learn the new hashes.
func assetCacheControl(path string) string {
	if strings.HasPrefix(path, "assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

// uiHandler serves the embedded SPA: real files when present, else index.html so
// client-side routes resolve.
func (a *app) uiHandler() http.Handler {
	sub, err := fs.Sub(webFS, "webdist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, statErr := fs.Stat(sub, p); statErr != nil {
			r.URL.Path = "/"
			p = "index.html"
		}
		w.Header().Set("Cache-Control", assetCacheControl(p))
		fileServer.ServeHTTP(w, r)
	})
}
