//go:build ffmpeg

// This test runs only with -tags ffmpeg AND a real ffmpeg reachable (bundled,
// GIFLY_FFMPEG_DIR, or PATH). It proves the argv actually produce a GIF - the
// one thing the unit tests, which assert argv against expected argv, cannot.
package gifjob

import (
	"context"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

func TestImagesProduceARealGIF(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg available: %v", err)
	}
	dir := t.TempDir()
	// Two solid-color PNGs via ffmpeg's lavfi color source, so the test needs no
	// fixtures checked in.
	for i, col := range []string{"red", "blue"} {
		p := filepath.Join(dir, col+".png")
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", "color=c=" + col + ":s=64x48:d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("making test image %d: %v", i, err)
		}
	}
	out := filepath.Join(dir, "out.gif")
	c := ImagesConfig{
		Inputs:  []string{filepath.Join(dir, "red.png"), filepath.Join(dir, "blue.png")},
		FrameMS: 200, Width: 64, Loop: LoopForever, Quality: DefaultQuality(),
	}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil)
	if err != nil {
		t.Fatalf("RunImages = %v", err)
	}
	f, err := os.Open(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("output is not a valid GIF: %v", err)
	}
	if len(g.Image) < 2 {
		t.Errorf("GIF has %d frames, want at least 2", len(g.Image))
	}
	if res.Width != 64 {
		t.Errorf("GIF width = %d, want 64", res.Width)
	}
}
