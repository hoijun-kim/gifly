//go:build ffmpeg

package app

import (
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

func TestConvertImagesEndToEnd(t *testing.T) {
	if _, err := ffmpeg.Tools(); err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	mk := func(name string, w, h int, c color.RGBA) string {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, c)
			}
		}
		p := filepath.Join(dir, name)
		f, _ := os.Create(p)
		_ = png.Encode(f, img)
		f.Close()
		return p
	}
	a := NewApp()
	a.ctx = context.Background() // no Wails runtime; events are best-effort
	out := filepath.Join(dir, "out.gif")
	req := ImagesRequest{
		Inputs:  []string{mk("a.png", 200, 120, color.RGBA{255, 0, 0, 255}), mk("b.png", 80, 200, color.RGBA{0, 0, 255, 255})},
		FrameMS: 200, Width: 240, Loop: "forever", Colors: 256, Dither: true, Out: out,
	}
	res, err := a.ConvertImages(req)
	if err != nil {
		t.Fatalf("ConvertImages = %v", err)
	}
	f, err := os.Open(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil || len(g.Image) < 2 {
		t.Fatalf("output not a multi-frame GIF: %v (%d frames)", err, len(g.Image))
	}
}

func TestConvertVideoEndToEnd(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()

	// Generate a short test video using lavfi testsrc
	inMP4 := filepath.Join(dir, "in.mp4")
	args := []string{"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=160x120:rate=10", "-c:v", "mpeg4", "-q:v", "3", inMP4}
	if err := ffmpeg.Run(context.Background(), tools.FFmpeg, args, nil); err != nil {
		t.Fatalf("failed to generate test video: %v", err)
	}

	a := NewApp()
	a.ctx = context.Background() // no Wails runtime; events are best-effort
	outGIF := filepath.Join(dir, "out.gif")

	req := VideoRequest{
		Input:   inMP4,
		StartMS: 200,
		EndMS:   1200,
		FPS:     8,
		Width:   120,
		Loop:    "forever",
		Colors:  128,
		Dither:  true,
		Out:     outGIF,
	}

	res, err := a.ConvertVideo(req)
	if err != nil {
		t.Fatalf("ConvertVideo = %v", err)
	}

	f, err := os.Open(res.Path)
	if err != nil {
		t.Fatalf("failed to open output GIF: %v", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("output not a valid GIF: %v", err)
	}

	if len(g.Image) < 2 {
		t.Errorf("output has only %d frame(s), want >= 2", len(g.Image))
	}

	if res.Width != 120 {
		t.Errorf("result width = %d, want 120", res.Width)
	}
}
