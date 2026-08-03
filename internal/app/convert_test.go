package app

import (
	"testing"

	"github.com/hoijun-kim/gifly/internal/gifjob"
)

func TestPercentClamped(t *testing.T) {
	if p := percent(500, 2000); p != 25 {
		t.Errorf("percent(500,2000) = %d, want 25", p)
	}
	if p := percent(5000, 2000); p != 100 {
		t.Errorf("percent past total should clamp to 100, got %d", p)
	}
	if p := percent(100, 0); p != 0 {
		t.Errorf("percent with zero total = %d, want 0 (no divide-by-zero)", p)
	}
}

func TestVideoConfigMapsRequest(t *testing.T) {
	req := VideoRequest{Input: "in.mp4", StartMS: 1000, EndMS: 3000, FPS: 15, Width: 480, Loop: "forever", Colors: 200, Dither: true}
	c := videoConfig(req)
	if c.Input != "in.mp4" || c.StartMS != 1000 || c.EndMS != 3000 || c.FPS != 15 || c.Width != 480 {
		t.Errorf("videoConfig basic fields wrong: %+v", c)
	}
	if int(c.Loop) != 0 { // "forever" -> 0
		t.Errorf("loop forever should map to 0, got %d", int(c.Loop))
	}
	if c.Quality.MaxColors != 200 || !c.Quality.Dither {
		t.Errorf("quality mapping wrong: %+v", c.Quality)
	}
}

func TestImagesConfigMapsRequest(t *testing.T) {
	req := ImagesRequest{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 320, Loop: "once", Colors: 128, Dither: false}
	c := imagesConfig(req, 240)
	if len(c.Inputs) != 2 || c.Inputs[0] != "a.png" || c.Inputs[1] != "b.png" {
		t.Errorf("imagesConfig inputs wrong: %+v", c.Inputs)
	}
	if c.FrameMS != 100 {
		t.Errorf("imagesConfig FrameMS = %d, want 100", c.FrameMS)
	}
	if c.Width != 320 {
		t.Errorf("imagesConfig Width = %d, want 320", c.Width)
	}
	if c.Height != 240 {
		t.Errorf("imagesConfig Height = %d, want 240", c.Height)
	}
	if int(c.Loop) != -1 { // "once" -> -1
		t.Errorf("loop once should map to -1, got %d", int(c.Loop))
	}
	if c.Quality.MaxColors != 128 || c.Quality.Dither {
		t.Errorf("quality mapping wrong: %+v", c.Quality)
	}
}

func TestNextFitWidth(t *testing.T) {
	cases := []struct {
		name string
		in   int
	}{
		{"large width shrinks and stays even", 600},
		{"odd width shrinks and stays even", 601},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := nextFitWidth(c.in)
			if got >= c.in {
				t.Errorf("nextFitWidth(%d) = %d, want strictly less than %d", c.in, got, c.in)
			}
			if got%2 != 0 {
				t.Errorf("nextFitWidth(%d) = %d, want even", c.in, got)
			}
		})
	}

	if got := nextFitWidth(120); got != 120 {
		t.Errorf("nextFitWidth(120) = %d, want 120 (already at the floor)", got)
	}
	if got := nextFitWidth(122); got != 120 {
		t.Errorf("nextFitWidth(122) = %d, want 120 (floors)", got)
	}

	// Repeated application must strictly decrease until it reaches the floor,
	// then hold there.
	w := 2000
	for i := 0; i < 50 && w > 120; i++ {
		next := nextFitWidth(w)
		if next >= w {
			t.Fatalf("nextFitWidth(%d) = %d, want strictly less than %d", w, next, w)
		}
		w = next
	}
	if w != 120 {
		t.Errorf("repeated application from 2000 should reach the floor 120, got stuck at %d", w)
	}
}

func TestParseLoopMode(t *testing.T) {
	tests := []struct {
		input    string
		expected gifjob.LoopMode
	}{
		{"forever", gifjob.LoopForever},
		{"once", gifjob.LoopOnce},
		{"5", gifjob.LoopMode(5)},
		{"nope", gifjob.LoopForever}, // invalid -> default to forever
		{"0", gifjob.LoopForever},    // 0 is not positive, falls back to forever
	}
	for _, tt := range tests {
		got := parseLoopMode(tt.input)
		if got != tt.expected {
			t.Errorf("parseLoopMode(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
