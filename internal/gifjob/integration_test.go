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
		FrameMS: 200, Width: 64, Height: 48, Loop: LoopForever, Quality: DefaultQuality(),
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

func TestImagesOfDifferentSizesProduceARealGIF(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg available: %v", err)
	}
	dir := t.TempDir()
	// Two deliberately different-sized frames - the case the concat demuxer
	// rejects without normalization.
	sizes := []struct {
		name string
		spec string
	}{
		{"wide.png", "color=c=red:s=300x120"},
		{"tall.png", "color=c=blue:s=120x300"},
	}
	for _, s := range sizes {
		p := filepath.Join(dir, s.name)
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", s.spec + ":d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("making %s: %v", s.name, err)
		}
	}
	out := filepath.Join(dir, "out.gif")
	c := ImagesConfig{
		Inputs:  []string{filepath.Join(dir, "wide.png"), filepath.Join(dir, "tall.png")},
		FrameMS: 200, Width: 240, Height: 160, Loop: LoopForever, Quality: DefaultQuality(),
	}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil)
	if err != nil {
		t.Fatalf("RunImages on mixed sizes = %v", err)
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
	if res.Width != 240 || res.Height != 160 {
		t.Errorf("GIF is %dx%d, want the 240x160 canvas", res.Width, res.Height)
	}
}

// makeTestVideo writes a short lavfi testsrc clip and returns its path.
func makeTestVideo(t *testing.T, tools ffmpeg.Paths, dir string) string {
	t.Helper()
	in := filepath.Join(dir, "in.mp4")
	if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
		[]string{"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15", "-c:v", "mpeg4", "-q:v", "3", in}, nil); err != nil {
		t.Fatalf("making test video: %v", err)
	}
	return in
}

func TestVideoWebPAndAPNGAreReal(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	for _, f := range []struct {
		format Format
		out    string
	}{
		{FormatWebP, "out.webp"},
		{FormatAPNG, "out.png"},
	} {
		c := VideoConfig{Input: in, StartMS: 0, EndMS: 1500, FPS: 10, Width: 160, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: f.format, Quality: DefaultQuality()}
		res, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, f.out), nil)
		if err != nil {
			t.Fatalf("%s encode: %v", f.format, err)
		}
		if res.Bytes < 100 {
			t.Errorf("%s output suspiciously small: %d bytes", f.format, res.Bytes)
		}
		if res.Width != 160 {
			t.Errorf("%s width = %d, want 160", f.format, res.Width)
		}
	}
}

func TestVideoBoomerangDoublesFrames(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	base := VideoConfig{Input: in, StartMS: 0, EndMS: 1000, FPS: 10, Width: 120, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: FormatGIF, Quality: DefaultQuality()}
	plain := base
	if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), plain, filepath.Join(dir, "p.gif"), nil); err != nil {
		t.Fatalf("plain: %v", err)
	}
	boom := base
	boom.Boomerang = true
	if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), boom, filepath.Join(dir, "b.gif"), nil); err != nil {
		t.Fatalf("boomerang: %v", err)
	}
	np := frameCount(t, filepath.Join(dir, "p.gif"))
	nb := frameCount(t, filepath.Join(dir, "b.gif"))
	if nb <= np {
		t.Errorf("boomerang produced %d frames, want more than the plain %d", nb, np)
	}
}

func TestVideoAspectSquareIsSquare(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	c := VideoConfig{Input: in, StartMS: 0, EndMS: 1000, FPS: 10, Width: 200, SrcWidth: 320, SrcHeight: 240, Aspect: AspectSquare, Loop: LoopForever, Format: FormatGIF, Quality: DefaultQuality()}
	res, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, "sq.gif"), nil)
	if err != nil {
		t.Fatalf("aspect square: %v", err)
	}
	if res.Width != res.Height {
		t.Errorf("1:1 output is %dx%d, want a square", res.Width, res.Height)
	}
}

func TestVideoDitherMethodsAllEncode(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	for _, d := range []DitherMethod{DitherNone, DitherBayer, DitherSierra, DitherFloyd} {
		c := VideoConfig{Input: in, StartMS: 0, EndMS: 800, FPS: 8, Width: 120, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: FormatGIF, Quality: Quality{MaxColors: 128, Dither: d}}
		out := filepath.Join(dir, "d_"+string(d)+".gif")
		if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil); err != nil {
			t.Fatalf("dither %s: %v", d, err)
		}
	}
}

func TestImagesReverseAndSpeedEncode(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	var imgs []string
	for i, col := range []string{"red", "green", "blue"} {
		p := filepath.Join(dir, col+".png")
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", "color=c=" + col + ":s=64x48:d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("image %d: %v", i, err)
		}
		imgs = append(imgs, p)
	}
	c := ImagesConfig{Inputs: imgs, FrameMS: 200, Width: 64, Height: 48, Reverse: true, Boomerang: true, Speed: 2.0, Loop: LoopForever, Format: FormatWebP, Quality: DefaultQuality()}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, "out.webp"), nil)
	if err != nil {
		t.Fatalf("images webp with reverse+boomerang+speed: %v", err)
	}
	if res.Bytes < 100 {
		t.Errorf("webp output suspiciously small: %d bytes", res.Bytes)
	}
}

// frameCount decodes a GIF and returns its frame count.
func frameCount(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("%s is not a valid GIF: %v", path, err)
	}
	return len(g.Image)
}
