package app

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestProbeImageInfo(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 120, 90))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := probeImageInfo(p)
	if err != nil {
		t.Fatalf("probeImageInfo = %v", err)
	}
	if got.Width != 120 || got.Height != 90 || got.Path != p {
		t.Errorf("probeImageInfo = %+v, want 120x90 at %q", got, p)
	}
}
