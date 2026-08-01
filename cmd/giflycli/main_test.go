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
