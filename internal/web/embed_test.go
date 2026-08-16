package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesEmbeddedAssetsWithoutSPAFallback(t *testing.T) {
	root, err := fs.Sub(files, "dist")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(root, "assets")
	if err != nil {
		t.Fatal(err)
	}
	var script string
	var underscoredScript string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			script = entry.Name()
			if strings.HasPrefix(entry.Name(), "_") {
				underscoredScript = entry.Name()
			}
			if underscoredScript != "" {
				break
			}
		}
	}
	if script == "" {
		t.Fatal("embedded assets do not contain a JavaScript chunk")
	}

	handler := Handler()
	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/"+script, nil))
	if asset.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", asset.Code, http.StatusOK)
	}
	if contentType := asset.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
		t.Fatalf("asset content type = %q, want JavaScript", contentType)
	}
	if underscoredScript == "" {
		t.Fatal("embedded assets do not contain an underscore-prefixed JavaScript chunk")
	}
	underscored := httptest.NewRecorder()
	handler.ServeHTTP(underscored, httptest.NewRequest(http.MethodGet, "/assets/"+underscoredScript, nil))
	if underscored.Code != http.StatusOK {
		t.Fatalf("underscore-prefixed asset status = %d, want %d", underscored.Code, http.StatusOK)
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want %d", missing.Code, http.StatusNotFound)
	}

	index := httptest.NewRecorder()
	handler.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if index.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("index cache control = %q, want no-cache", index.Header().Get("Cache-Control"))
	}
}
