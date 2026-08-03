package gifjob

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// fakeRunner records every invocation and, on a specified call index (writeOn),
// writes a stub GIF to the output path so RunVideo/RunImages can stat a result.
type fakeRunner struct {
	calls   [][]string
	out     string
	writeOn int // 1-based call index that writes the stub output; 0 = never
}

func (r *fakeRunner) Run(_ context.Context, _ string, args []string, _ func(ffmpeg.Progress)) error {
	r.calls = append(r.calls, args)
	if r.writeOn != 0 && len(r.calls) == r.writeOn {
		return os.WriteFile(r.out, []byte("GIF89a-stub"), 0o644)
	}
	return nil
}

// giflyTempFiles snapshots the set of gifly-* temp files currently in
// os.TempDir(), so a test can diff before/after a call and catch a temp file
// left behind by a missing or broken defer, without false-failing on
// unrelated pre-existing files.
func giflyTempFiles(t *testing.T) map[string]bool {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "gifly-*"))
	if err != nil {
		t.Fatalf("glob temp dir: %v", err)
	}
	set := make(map[string]bool, len(matches))
	for _, m := range matches {
		set[m] = true
	}
	return set
}

// assertNoNewGiflyTempFiles fails the test if any gifly-* temp file exists
// after that did not exist before.
func assertNoNewGiflyTempFiles(t *testing.T, before map[string]bool) {
	t.Helper()
	after := giflyTempFiles(t)
	for p := range after {
		if !before[p] {
			t.Errorf("temp file left behind: %s", p)
		}
	}
}

func TestRunVideoRunsBothPasses(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	r := &fakeRunner{out: out, writeOn: 2}
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 2000, FPS: 12, Width: 320, Loop: LoopForever, Quality: DefaultQuality()}

	before := giflyTempFiles(t)
	res, err := RunVideo(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunVideo = %v", err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("expected 2 ffmpeg passes, got %d", len(r.calls))
	}
	// Pass 1 writes a palette; pass 2's last arg is the output GIF. Assert
	// pass 1's last arg is NOT the output path, so a mutation that ran the
	// encode pass twice (instead of palette-then-encode) is caught.
	if r.calls[0][len(r.calls[0])-1] == out {
		t.Errorf("palette pass output = %q, want the palette path, not %q", r.calls[0][len(r.calls[0])-1], out)
	}
	if r.calls[1][len(r.calls[1])-1] != out {
		t.Errorf("encode pass output = %q, want %q", r.calls[1][len(r.calls[1])-1], out)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Errorf("result = %+v, want a non-empty file at %q", res, out)
	}
	assertNoNewGiflyTempFiles(t, before)
}

func TestRunVideoRejectsInvalidConfig(t *testing.T) {
	r := &fakeRunner{}
	_, err := RunVideo(context.Background(), ffmpeg.Paths{}, r, VideoConfig{}, "out.gif", nil)
	if err == nil {
		t.Fatal("RunVideo with an empty config should refuse before running ffmpeg")
	}
	if len(r.calls) != 0 {
		t.Errorf("no ffmpeg call should happen for an invalid config, got %d", len(r.calls))
	}
}

func TestRunImagesRunsBothPasses(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	before := giflyTempFiles(t)
	// 2 images -> 2 normalize calls + palette + encode = 4 calls; encode is last.
	r := &fakeRunner{out: out, writeOn: 4}
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 320, Height: 240, Loop: LoopForever, Quality: DefaultQuality()}

	res, err := RunImages(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunImages = %v", err)
	}
	if len(r.calls) != 4 {
		t.Fatalf("expected 2 normalize + palette + encode = 4 calls, got %d", len(r.calls))
	}
	// Calls 1 and 2 are the per-image normalize passes (last arg is a temp png,
	// not the output); the final call's last arg is the output GIF.
	if r.calls[0][len(r.calls[0])-1] == out || r.calls[1][len(r.calls[1])-1] == out {
		t.Error("a normalize call wrote the output GIF; expected a temp frame")
	}
	if r.calls[3][len(r.calls[3])-1] != out {
		t.Errorf("final (encode) call output = %q, want %q", r.calls[3][len(r.calls[3])-1], out)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Errorf("result = %+v, want a non-empty file at %q", res, out)
	}
	assertNoNewGiflyTempFiles(t, before)
}

func TestRunImagesRejectsInvalidConfig(t *testing.T) {
	r := &fakeRunner{}
	_, err := RunImages(context.Background(), ffmpeg.Paths{}, r, ImagesConfig{}, "out.gif", nil)
	if err == nil {
		t.Fatal("RunImages with an empty config should refuse before running ffmpeg")
	}
	if len(r.calls) != 0 {
		t.Errorf("no ffmpeg call should happen for an invalid config, got %d", len(r.calls))
	}
}

// failOnEncode is a runner that writes a partial output on the encode pass and
// then reports failure, to prove RunVideo and RunImages remove the half-written file.
type failOnEncode struct {
	out       string
	encodeIdx int
	calls     int
}

func (r *failOnEncode) Run(_ context.Context, _ string, _ []string, _ func(ffmpeg.Progress)) error {
	r.calls++
	if r.calls == r.encodeIdx {
		_ = os.WriteFile(r.out, []byte("partial"), 0o644)
		return errors.New("ffmpeg failed mid-encode")
	}
	return nil
}

func TestRunVideoRemovesPartialOutputOnFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	r := &failOnEncode{out: out, encodeIdx: 2} // video: palette(1), encode(2)
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 320, Loop: LoopForever, Quality: DefaultQuality()}

	if _, err := RunVideo(context.Background(), tools, r, c, out, nil); err == nil {
		t.Fatal("RunVideo should return the encode error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("partial output %q was left behind after a failed encode", out)
	}
}

func TestRunImagesRemovesPartialOutputOnFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	r := &failOnEncode{out: out, encodeIdx: 4} // 2 normalize + palette + encode(4)
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 320, Height: 240, Loop: LoopForever, Quality: DefaultQuality()}
	if _, err := RunImages(context.Background(), tools, r, c, out, nil); err == nil {
		t.Fatal("RunImages should return the encode error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("partial output %q was left behind after a failed encode", out)
	}
}

func TestRunVideoWebPIsSinglePass(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.webp")
	r := &fakeRunner{out: out, writeOn: 1} // single pass writes the output
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 2000, FPS: 12, Width: 320, Loop: LoopForever, Format: FormatWebP, SrcWidth: 640, SrcHeight: 360, Quality: DefaultQuality()}
	before := giflyTempFiles(t)
	res, err := RunVideo(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunVideo webp = %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("WebP must be a single ffmpeg pass, got %d", len(r.calls))
	}
	// WebP cannot be decoded by the stdlib, so dimensions fall back to the
	// config: width 320, height OutputHeight(640,360,free,320)=180.
	if res.Width != 320 || res.Height != 180 {
		t.Errorf("webp result dims = %dx%d, want 320x180 (fallback)", res.Width, res.Height)
	}
	assertNoNewGiflyTempFiles(t, before) // no palette temp for webp
}
