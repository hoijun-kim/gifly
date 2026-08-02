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

func TestCanvasHeight(t *testing.T) {
	// 1600x900 source at output width 800 -> 450 (even).
	if got := CanvasHeight(1600, 900, 800); got != 450 {
		t.Errorf("CanvasHeight(1600,900,800) = %d, want 450", got)
	}
	// Odd result rounds up to even: 100x101 at width 100 -> 101 -> 102.
	if got := CanvasHeight(100, 101, 100); got != 102 {
		t.Errorf("CanvasHeight(100,101,100) = %d, want 102 (even)", got)
	}
	// Degenerate inputs never produce less than 2.
	if got := CanvasHeight(0, 0, 0); got != 2 {
		t.Errorf("CanvasHeight(0,0,0) = %d, want 2", got)
	}
	// All-positive inputs that round below 2 exercise the floor: 100000x1 at width 10 -> round(10*1/100000)=0 -> floored to 2.
	if got := CanvasHeight(100000, 1, 10); got != 2 {
		t.Errorf("CanvasHeight(100000,1,10) = %d, want 2 (floor)", got)
	}
}

func TestImagesConfigValidate(t *testing.T) {
	ok := ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 300, Loop: LoopForever, Quality: DefaultQuality()}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	bad := []struct {
		name string
		c    ImagesConfig
	}{
		{"empty image list", ImagesConfig{Inputs: nil, FrameMS: 100, Width: 400, Height: 300, Quality: DefaultQuality()}},
		{"zero frame duration", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 0, Width: 400, Height: 300, Quality: DefaultQuality()}},
		{"zero width", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 0, Height: 300, Quality: DefaultQuality()}},
		{"zero height", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 0, Quality: DefaultQuality()}},
		{"loop below once", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 300, Loop: -2, Quality: DefaultQuality()}},
		{"colors too low", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 300, Quality: Quality{MaxColors: 1}}},
		{"colors too high", ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 300, Quality: Quality{MaxColors: 300}}},
	}
	for _, b := range bad {
		if err := b.c.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", b.name)
		}
	}
}

func TestImagesConfigValidateRequiresHeight(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Height: 300, Loop: LoopForever, Quality: DefaultQuality()}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid config with Height rejected: %v", err)
	}
	c.Height = 0
	if err := c.Validate(); err == nil {
		t.Error("a zero canvas Height must be rejected")
	}
}
