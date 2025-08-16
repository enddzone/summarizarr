package api

import (
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// getEmbeddedStaticFS attempts to load the built static assets directory that
// is normally embedded at build time. If it's missing (e.g. developer forgot to
// run `make build-frontend`) the test is skipped to avoid false negatives.
func getEmbeddedStaticFS(t *testing.T) fs.FS {
	t.Helper()

	// Find repository root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	repoRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Fatalf("could not find repository root (go.mod not found)")
		}
		repoRoot = parent
	}

	root := filepath.Join(repoRoot, "internal", "frontend", "static")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("static assets directory %s not present (run make build-frontend first): %v", root, err)
	}
	return os.DirFS(root)
}

func TestFrontendIndexServed(t *testing.T) {
	ffs := getEmbeddedStaticFS(t)
	srv := NewServer(":0", nil, ffs) // DB not required for static file serving

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for /, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(strings.ToLower(body), "<html") || !strings.Contains(body, "</html>") {
		t.Fatalf("response for / does not appear to be full HTML document; length=%d", len(body))
	}
	if !strings.Contains(body, "Summarizarr") {
		t.Errorf("expected HTML to contain 'Summarizarr'")
	}
}

func TestFrontendSpaFallback(t *testing.T) {
	ffs := getEmbeddedStaticFS(t)
	srv := NewServer(":0", nil, ffs)

	req := httptest.NewRequest("GET", "/non-existent-deep/route", nil)
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)

	if w.Code != 200 { // SPA fallback should still serve index.html
		t.Fatalf("expected 200 for SPA fallback route, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Summarizarr") {
		t.Errorf("expected fallback HTML to contain 'Summarizarr'")
	}
}
