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
		{"speed too high", VideoConfig{Input: "in.mp4", EndMS: 5000, FPS: 15, Width: 480, Speed: 9, Quality: DefaultQuality()}},
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

func TestFormatMethods(t *testing.T) {
	cases := []struct {
		f     Format
		valid bool
		ext   string
	}{
		{FormatGIF, true, ".gif"},
		{FormatWebP, true, ".webp"},
		{FormatAPNG, true, ".png"},
		{Format("xyz"), false, ".gif"},
	}
	for _, c := range cases {
		if c.f.Valid() != c.valid {
			t.Errorf("%q.Valid() = %v, want %v", c.f, c.f.Valid(), c.valid)
		}
		if c.f.Ext() != c.ext {
			t.Errorf("%q.Ext() = %q, want %q", c.f, c.f.Ext(), c.ext)
		}
	}
}

func TestDitherFFmpegNames(t *testing.T) {
	cases := map[DitherMethod]string{
		DitherNone:       "none",
		DitherBayer:      "bayer",
		DitherSierra:     "sierra2_4a",
		DitherFloyd:      "floyd_steinberg",
		DitherMethod(""): "sierra2_4a", // empty defaults to sierra2_4a
	}
	for d, want := range cases {
		if got := d.ffmpeg(); got != want {
			t.Errorf("%q.ffmpeg() = %q, want %q", string(d), got, want)
		}
	}
}

func TestOutputHeightAspect(t *testing.T) {
	// free: same as old CanvasHeight (round, even-up).
	if got := OutputHeight(1600, 900, AspectFree, 800); got != 450 {
		t.Errorf("free 1600x900@800 = %d, want 450", got)
	}
	// square: height equals width.
	if got := OutputHeight(1600, 900, AspectSquare, 200); got != 200 {
		t.Errorf("square @200 = %d, want 200", got)
	}
	// 16:9 at width 200 -> 112 or 114 (even); assert it is even and near 112.
	if got := OutputHeight(1600, 900, AspectWide, 200); got%2 != 0 || got < 112 || got > 114 {
		t.Errorf("16:9 @200 = %d, want an even value 112..114", got)
	}
	// 9:16 at width 200 -> 356 (even).
	if got := OutputHeight(1600, 900, AspectTall, 200); got != 356 {
		t.Errorf("9:16 @200 = %d, want 356", got)
	}
}

func TestDefaultQualityIsDithered(t *testing.T) {
	q := DefaultQuality()
	if q.MaxColors != 256 || q.Dither != DitherSierra || q.WebPQuality != 75 {
		t.Errorf("DefaultQuality() = %+v, want {256 sierra2 75}", q)
	}
}
