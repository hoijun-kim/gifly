package gifjob

import "testing"

func TestVideoConfigValidate(t *testing.T) {
	ok := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 5000, FPS: 15, Width: 480, Loop: LoopForever, Quality: DefaultQuality()}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	bad := []struct {
		name string
		c    VideoConfig
	}{
		{"empty input", VideoConfig{Input: "", EndMS: 5000, FPS: 15, Width: 480, Quality: DefaultQuality()}},
		{"end before start", VideoConfig{Input: "in.mp4", StartMS: 5000, EndMS: 5000, FPS: 15, Width: 480, Quality: DefaultQuality()}},
		{"zero fps", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 0, Width: 480, Quality: DefaultQuality()}},
		{"zero width", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 15, Width: 0, Quality: DefaultQuality()}},
		{"colors too low", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 15, Width: 480, Quality: Quality{MaxColors: 1}}},
		{"colors too high", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 15, Width: 480, Quality: Quality{MaxColors: 300}}},
		{"loop below once", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 15, Width: 480, Loop: -2, Quality: DefaultQuality()}},
	}
	for _, b := range bad {
		if err := b.c.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", b.name)
		}
	}
}

func TestImagesConfigValidate(t *testing.T) {
	ok := ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Loop: LoopForever, Quality: DefaultQuality()}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	bad := []struct {
		name string
		c    ImagesConfig
	}{
		{"empty image list", ImagesConfig{Inputs: nil, FrameMS: 100, Width: 400, Quality: DefaultQuality()}},
		{"zero frame duration", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 0, Width: 400, Quality: DefaultQuality()}},
		{"zero width", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 0, Quality: DefaultQuality()}},
		{"loop below once", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Loop: -2, Quality: DefaultQuality()}},
		{"colors too low", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Quality: Quality{MaxColors: 1}}},
		{"colors too high", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Quality: Quality{MaxColors: 300}}},
	}
	for _, b := range bad {
		if err := b.c.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", b.name)
		}
	}
}
