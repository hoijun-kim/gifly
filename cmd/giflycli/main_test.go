package main

import (
	"testing"

	"github.com/hoijun-kim/gifly/internal/gifjob"
)

func TestParseLoop(t *testing.T) {
	cases := []struct {
		in   string
		want gifjob.LoopMode
		bad  bool
	}{
		{"forever", gifjob.LoopForever, false},
		{"once", gifjob.LoopOnce, false},
		{"3", gifjob.LoopMode(3), false},
		{"-2", 0, true},
		{"nope", 0, true},
		{"0", gifjob.LoopMode(0), false},
		{"-1", 0, true},
	}
	for _, c := range cases {
		got, err := parseLoop(c.in)
		if c.bad {
			if err == nil {
				t.Errorf("parseLoop(%q) = %v, want an error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseLoop(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
}

func TestImagesHeight(t *testing.T) {
	// signature: imagesHeight(override, width, firstW, firstH int) int
	// override <= 0 -> derive from the first frame via CanvasHeight
	if got := imagesHeight(0, 800, 1600, 900); got != gifjob.CanvasHeight(1600, 900, 800) {
		t.Errorf("imagesHeight(derive) = %d, want %d", got, gifjob.CanvasHeight(1600, 900, 800))
	}
	// positive override is used verbatim
	if got := imagesHeight(500, 800, 1600, 900); got != 500 {
		t.Errorf("imagesHeight(override) = %d, want 500", got)
	}
}

func TestParseFormatAspectDither(t *testing.T) {
	if f, err := parseFormat("webp"); err != nil || f != gifjob.FormatWebP {
		t.Errorf("parseFormat(webp) = %v %v", f, err)
	}
	if _, err := parseFormat("mp4"); err == nil {
		t.Error("parseFormat(mp4) should error")
	}
	if a, err := parseAspect("16:9"); err != nil || a != gifjob.AspectWide {
		t.Errorf("parseAspect(16:9) = %v %v", a, err)
	}
	if a, err := parseAspect("free"); err != nil || a != gifjob.AspectFree {
		t.Errorf("parseAspect(free) = %v %v", a, err)
	}
	if d, err := parseDither("floyd"); err != nil || d != gifjob.DitherFloyd {
		t.Errorf("parseDither(floyd) = %v %v", d, err)
	}
}

func TestDefaultOut(t *testing.T) {
	// A caller-set output keeps its name; the default name follows the format.
	if got := defaultOut("out.gif", gifjob.FormatWebP); got != "out.webp" {
		t.Errorf("defaultOut(out.gif, webp) = %q, want out.webp", got)
	}
	if got := defaultOut("mine.gif", gifjob.FormatGIF); got != "mine.gif" {
		t.Errorf("defaultOut(mine.gif, gif) = %q, want mine.gif", got)
	}
	if got := defaultOut("out.gif", gifjob.FormatAPNG); got != "out.png" {
		t.Errorf("defaultOut(out.gif, apng) = %q, want out.png", got)
	}
}
