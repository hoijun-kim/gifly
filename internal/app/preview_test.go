package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewContentTypeByExt(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"out.gif":  "image/gif",
		"out.webp": "image/webp",
		"out.png":  "image/apng",
	}
	for name, wantCT := range cases {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("stub-bytes"), 0o644); err != nil {
			t.Fatal(err)
		}
		a := NewApp()
		a.SetPreview(p)
		rec := httptest.NewRecorder()
		a.PreviewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/preview.gif", nil))
		if got := rec.Header().Get("Content-Type"); got != wantCT {
			t.Errorf("%s -> Content-Type %q, want %q", name, got, wantCT)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s -> Cache-Control %q, want no-store", name, got)
		}
	}
}
