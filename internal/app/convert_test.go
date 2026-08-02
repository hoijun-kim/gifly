package app

import "testing"

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
