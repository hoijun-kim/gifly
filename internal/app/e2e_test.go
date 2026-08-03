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

func TestConvertFitsToTargetSize(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "in.mp4")
	if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
		[]string{"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15", "-c:v", "mpeg4", "-q:v", "3", in}, nil); err != nil {
		t.Fatalf("making test video: %v", err)
	}
	a := NewApp()
	a.ctx = context.Background()
	base := VideoRequest{Input: in, StartMS: 0, EndMS: 2000, FPS: 15, Width: 320, Loop: "forever", Colors: 256, Dither: true, Out: filepath.Join(dir, "base.gif")}
	res0, err := a.ConvertVideo(base)
	if err != nil {
		t.Fatalf("baseline convert: %v", err)
	}
	// Target half the untargeted size, so the fit loop must shrink to reach it.
	targetKB := int(res0.Bytes/1024/2) + 1
	fit := base
	fit.Out = filepath.Join(dir, "fit.gif")
	fit.TargetKB = targetKB
	res1, err := a.ConvertVideo(fit)
	if err != nil {
		t.Fatalf("targeted convert: %v", err)
	}
	if res1.Width >= 320 {
		t.Errorf("fit did not shrink the width: %d (baseline 320)", res1.Width)
	}
	// Either it reached the target, or it bottomed out at the 120 floor.
	if res1.Bytes > int64(targetKB)*1024 && res1.Width != 120 {
		t.Errorf("fit output %d bytes over target %d KB but width %d is above the 120 floor",
			res1.Bytes, targetKB, res1.Width)
	}
}
