package main

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"strings"
)

// agentFiles holds the cross-compiled agent binaries. Populated by
// `python3 scripts/release.py --embed`; the committed PLACEHOLDER keeps
// go:embed and `go build` working when no binaries are present.
//
//go:embed all:agentdist
var agentFiles embed.FS

// installScripts holds the host install/uninstall scripts, staged into
// server/scripts by `make server-assets` (source of truth: deploy/*.sh).
//
//go:embed all:scripts
var installScripts embed.FS

// agentDistFS returns the agentdist subtree.
func agentDistFS() fs.FS {
	sub, err := fs.Sub(agentFiles, "agentdist")
	if err != nil {
		panic(err)
	}
	return sub
}

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
		// An unrouted /api/ path is a caller's mistake, not a client-side
		// route. Falling through would answer it with the SPA and a 200, which
		// a non-browser caller reads as success.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
			return
		}
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
