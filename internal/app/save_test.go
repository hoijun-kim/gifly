package app

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"clip":              "clip",
		"a/b:c*d?":          "abcd",
		`bad\\/:*?"<>|name`: "badname",
		"  spaced  ":        "spaced",
		"trailing. ":        "trailing",
		"":                  "gifly",
		"   ":               "gifly",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveOutPath(t *testing.T) {
	now := time.Date(2026, 8, 4, 2, 15, 30, 0, time.UTC)
	dir := filepath.Join("out", "dir")

	// overwrite: always the plain name, ignores existence.
	taken := map[string]bool{filepath.Join(dir, "clip.gif"): true}
	exists := func(p string) bool { return taken[p] }
	if got := resolveOutPath(dir, "clip", ".gif", "overwrite", exists, now); got != filepath.Join(dir, "clip.gif") {
		t.Errorf("overwrite = %q", got)
	}

	// number: free name when nothing exists.
	empty := func(string) bool { return false }
	if got := resolveOutPath(dir, "clip", ".gif", "number", empty, now); got != filepath.Join(dir, "clip.gif") {
		t.Errorf("number (free) = %q, want clip.gif", got)
	}
	// number: first collision -> " (1)".
	if got := resolveOutPath(dir, "clip", ".gif", "number", exists, now); got != filepath.Join(dir, "clip (1).gif") {
		t.Errorf("number (1 taken) = %q, want clip (1).gif", got)
	}
	// number: clip.gif and clip (1).gif taken -> " (2)".
	taken2 := map[string]bool{filepath.Join(dir, "clip.gif"): true, filepath.Join(dir, "clip (1).gif"): true}
	if got := resolveOutPath(dir, "clip", ".gif", "number", func(p string) bool { return taken2[p] }, now); got != filepath.Join(dir, "clip (2).gif") {
		t.Errorf("number (2 taken) = %q, want clip (2).gif", got)
	}

	// timestamp: name-YYYYMMDD-HHMMSS.
	if got := resolveOutPath(dir, "clip", ".webp", "timestamp", exists, now); got != filepath.Join(dir, "clip-20260804-021530.webp") {
		t.Errorf("timestamp = %q, want clip-20260804-021530.webp", got)
	}

	// unknown policy defaults to number behavior.
	if got := resolveOutPath(dir, "clip", ".gif", "", exists, now); got != filepath.Join(dir, "clip (1).gif") {
		t.Errorf("default policy = %q, want number behavior", got)
	}

	// bad name gets sanitized before use.
	if got := resolveOutPath(dir, "a/b", ".gif", "overwrite", empty, now); got != filepath.Join(dir, "ab.gif") {
		t.Errorf("sanitized = %q, want ab.gif", got)
	}
}
