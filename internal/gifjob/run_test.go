package gifjob

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// fakeRunner records every invocation and, on the second call (the encode pass),
// writes a stub GIF to the output path so RunVideo/RunImages can stat a result.
type fakeRunner struct {
	calls [][]string
	out   string
}

func (r *fakeRunner) Run(_ context.Context, _ string, args []string, _ func(ffmpeg.Progress)) error {
	r.calls = append(r.calls, args)
	if len(r.calls) == 2 { // the encode pass writes the output (last arg)
		return os.WriteFile(r.out, []byte("GIF89a-stub"), 0o644)
	}
	return nil
}

func TestRunVideoRunsBothPasses(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	r := &fakeRunner{out: out}
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 2000, FPS: 12, Width: 320, Loop: LoopForever, Quality: DefaultQuality()}

	res, err := RunVideo(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunVideo = %v", err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("expected 2 ffmpeg passes, got %d", len(r.calls))
	}
	// Pass 1 writes a palette; pass 2's last arg is the output GIF.
	if r.calls[1][len(r.calls[1])-1] != out {
		t.Errorf("encode pass output = %q, want %q", r.calls[1][len(r.calls[1])-1], out)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Errorf("result = %+v, want a non-empty file at %q", res, out)
	}
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
