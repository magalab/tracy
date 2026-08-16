package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Include Vite chunks whose names start with '_' (for example _baseDifference).
// Plain "dist" patterns intentionally skip files and directories beginning with
// '.' or '_'; the all: prefix is required for a complete embedded asset tree.
//
//go:embed all:dist
var files embed.FS

func Handler() http.Handler {
	root, _ := fs.Sub(files, "dist")
	static := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(root, name); err != nil {
			if strings.HasPrefix(name, "assets/") {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/index.html"
		}
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		static.ServeHTTP(w, r)
	})
}
