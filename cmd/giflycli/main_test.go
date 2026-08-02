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
