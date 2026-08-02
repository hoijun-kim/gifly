package app

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCancelFiresAndClears(t *testing.T) {
	a := NewApp()

	// Set a cancel function that fires a flag
	fired := false
	a.mu.Lock()
	a.cancelFn = func() {
		fired = true
	}
	a.mu.Unlock()

	// Call Cancel() and verify it fired
	a.Cancel()
	if !fired {
		t.Error("Cancel did not call cancelFn")
	}

	// Verify cancelFn is now nil under the lock
	a.mu.Lock()
	if a.cancelFn != nil {
		t.Error("cancelFn should be cleared after Cancel, but it is not")
	}
	a.mu.Unlock()

	// Call Cancel() again on the now-cleared App; should not panic
	a.Cancel()
}

func TestPreviewHandlerServesAndGuards(t *testing.T) {
	a := NewApp()
	h := a.PreviewHandler()

	// Test 1: No preview set, GET should return 404
	req := httptest.NewRequest(http.MethodGet, "/preview.gif", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("no preview: got status %d, want 404", w.Code)
	}

	// Test 2: Write a valid GIF to a temp file
	dir := t.TempDir()
	gifPath := filepath.Join(dir, "test.gif")

	// Create a small 2x2 GIF with one frame using image/gif
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 255, 0, 255},
	})
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 1, color.RGBA{0, 255, 0, 255})

	f, err := os.Create(gifPath)
	if err != nil {
		t.Fatalf("failed to create temp GIF: %v", err)
	}
	defer f.Close()

	if err := gif.EncodeAll(f, &gif.GIF{
		Image: []*image.Paletted{img},
		Delay: []int{10},
	}); err != nil {
		t.Fatalf("failed to encode GIF: %v", err)
	}
	f.Close()

	// Set the preview
	a.SetPreview(gifPath)

	// Test 3: GET with preview set should return 200 with correct headers
	req = httptest.NewRequest(http.MethodGet, "/preview.gif", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with preview: got status %d, want 200", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "image/gif" {
		t.Errorf("Content-Type = %q, want image/gif", ct)
	}

	cc := w.Header().Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}

	// Verify response body is a valid GIF
	if body := w.Body.Bytes(); len(body) < 6 {
		t.Errorf("response body too short to be a GIF: %d bytes", len(body))
	}
	bodyReader := bytes.NewReader(w.Body.Bytes())
	_, err = gif.DecodeAll(bodyReader)
	if err != nil {
		t.Errorf("response body is not a valid GIF: %v", err)
	}

	// Test 4: POST should return 405
	req = httptest.NewRequest(http.MethodPost, "/preview.gif", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST method: got status %d, want 405", w.Code)
	}
}
