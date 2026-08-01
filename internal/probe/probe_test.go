package probe

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// A trimmed but real-shaped ffprobe JSON dump for one video stream.
const probeJSON = `{
  "streams": [
    {"codec_type": "video", "width": 1920, "height": 1080, "avg_frame_rate": "30000/1001"}
  ],
  "format": {"duration": "12.500000"}
}`

func TestParseProbeJSON(t *testing.T) {
	m, err := parseProbeJSON([]byte(probeJSON))
	if err != nil {
		t.Fatalf("parseProbeJSON = %v", err)
	}
	if m.DurationMS != 12500 {
		t.Errorf("DurationMS = %d, want 12500", m.DurationMS)
	}
	if m.Width != 1920 || m.Height != 1080 {
		t.Errorf("size = %dx%d, want 1920x1080", m.Width, m.Height)
	}
	// 30000/1001 = 29.97
	if m.FPS < 29.9 || m.FPS > 30.0 {
		t.Errorf("FPS = %v, want ~29.97", m.FPS)
	}
}

func TestImageReadsDimensions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 64, 48))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Image(p)
	if err != nil {
		t.Fatalf("Image = %v", err)
	}
	if f.Width != 64 || f.Height != 48 {
		t.Errorf("frame = %dx%d, want 64x48", f.Width, f.Height)
	}
}
