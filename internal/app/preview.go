package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
)

// SetPreview stores the path to the current preview GIF, guarded by the lock.
func (a *App) SetPreview(path string) {
	a.mu.Lock()
	a.previewPath = path
	a.mu.Unlock()
}

// PreviewHandler returns an http.Handler that serves the current preview GIF.
// It responds to any GET request (since it is the AssetServer fallback) by
// serving the file at previewPath with cache-disabling headers, or 404 if
// no preview is available.
func (a *App) PreviewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		a.mu.Lock()
		path := a.previewPath
		a.mu.Unlock()

		if path == "" {
			http.NotFound(w, r)
			return
		}

		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "image/gif")
		http.ServeContent(w, r, path, fi.ModTime(), f)
	})
}

// RevealOutput opens the folder containing the output GIF and selects the file.
// On Windows, it uses "explorer /select".
func (a *App) RevealOutput(path string) error {
	// Verify the file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("output file not found: %w", err)
	}

	// On Windows, use explorer /select to highlight the file
	return exec.Command("explorer", "/select,", path).Start()
}
