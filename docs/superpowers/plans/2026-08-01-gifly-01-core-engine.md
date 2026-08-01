# gifly Plan 1 - Core Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A headless, cgo-free Go engine that turns a video (with trim) or an ordered set of images into an optimized GIF by driving a bundled ffmpeg through a two-pass palettegen/paletteuse pipeline, exercised end to end by a small CLI harness.

**Architecture:** Four small packages. `internal/ffmpeg` is the only code that shells out (locate the binaries, run them, parse progress). `internal/probe` reads a source's duration/dimensions via ffprobe. `internal/gifjob` turns a typed config into the exact ffmpeg argument lists and orchestrates the two passes; its argument builders are pure and are the load-bearing tests. `cmd/giflycli` wires it together for real end-to-end use before any GUI exists.

**Tech Stack:** Go 1.26, standard library only (os/exec, encoding/json, image), external ffmpeg/ffprobe processes. No GUI in this plan.

## Global Constraints

- Windows only. Go 1.26. Module path `github.com/hoijun-kim/gifly`.
- cgo-free: `CGO_ENABLED=0 go build ./...` and `go test ./...` must pass. No `golang.org/x/image`. No cgo.
- Dependencies: standard library only in this plan (Wails arrives in Plan 2). No new module `require` entries.
- ffmpeg is an external process, bundled later; never assumed on PATH. Located next to the exe, else `GIFLY_FFMPEG_DIR`, else PATH.
- Copy/comments: plain ASCII `-` only, never a unicode dash.
- Commits: author is the repo's configured user; NO AI co-author trailer on any commit.
- `.gitattributes` already pins `* text=auto eol=lf`; keep Go sources LF so `gofmt -l` stays clean.
- ffmpeg `-loop` GIF semantics used verbatim as the `LoopMode` value: `0` = loop forever, `-1` = play once, `n > 0` = loop n times.

---

### Task 1: Project scaffold and ffmpeg binary location

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/ffmpeg/locate.go`
- Test: `internal/ffmpeg/locate_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces:
  - `type Paths struct { FFmpeg, FFprobe string }`
  - `func Tools() (Paths, error)` - resolves the two binaries.
  - `func resolveDir() (string, error)` - the directory the binaries live in, used by Tools; exported-to-package for the test.

- [ ] **Step 1: Initialize the module and ignore file**

Run:
```bash
cd /c/Users/hoijun/Projects/gifly
go mod init github.com/hoijun-kim/gifly
printf 'gifly.exe\ngiflycli.exe\n/build/\n*.gif\n' > .gitignore
```
Expected: `go.mod` exists naming module `github.com/hoijun-kim/gifly` with `go 1.26`.

- [ ] **Step 2: Write the failing test**

```go
package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
)

// writeExe creates a fake executable file so location logic (which only checks
// existence, never runs the file) has something to find.
func writeExe(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestToolsFindsBinariesInAGivenDir(t *testing.T) {
	dir := t.TempDir()
	writeExe(t, dir, "ffmpeg.exe")
	writeExe(t, dir, "ffprobe.exe")
	t.Setenv("GIFLY_FFMPEG_DIR", dir)

	got, err := Tools()
	if err != nil {
		t.Fatalf("Tools() = %v, want the binaries in %s", err, dir)
	}
	if got.FFmpeg != filepath.Join(dir, "ffmpeg.exe") {
		t.Errorf("FFmpeg = %q, want it under %s", got.FFmpeg, dir)
	}
	if got.FFprobe != filepath.Join(dir, "ffprobe.exe") {
		t.Errorf("FFprobe = %q, want it under %s", got.FFprobe, dir)
	}
}

func TestToolsErrorsWhenMissing(t *testing.T) {
	// An empty dir with the env pointed at it, and an empty PATH, so neither the
	// bundle nor a system ffmpeg can be found - the test does not depend on
	// whether the machine running it happens to have ffmpeg installed.
	t.Setenv("GIFLY_FFMPEG_DIR", t.TempDir())
	t.Setenv("PATH", "")
	if _, err := Tools(); err == nil {
		t.Fatal("Tools() with no binaries present returned nil error; a missing bundle must be reported")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/ffmpeg/ -run TestTools -v`
Expected: FAIL - `undefined: Tools`.

- [ ] **Step 4: Implement the location logic**

```go
// Package ffmpeg is the only part of gifly that shells out to an external
// process. It locates the bundled ffmpeg and ffprobe binaries and runs them;
// nothing else in the codebase spawns a process.
package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Paths holds the resolved locations of the two binaries gifly drives.
type Paths struct {
	FFmpeg  string
	FFprobe string
}

// resolveDir returns the directory the bundled binaries live in. Preference,
// highest first: the GIFLY_FFMPEG_DIR override (used in development and tests),
// then an "ffmpeg" folder beside the running executable (how a release ships).
// It returns the first directory that actually contains ffmpeg.exe.
func resolveDir() (string, error) {
	var tried []string
	consider := func(dir string) (string, bool) {
		if dir == "" {
			return "", false
		}
		tried = append(tried, dir)
		if _, err := os.Stat(filepath.Join(dir, "ffmpeg.exe")); err == nil {
			return dir, true
		}
		return "", false
	}

	if dir, ok := consider(os.Getenv("GIFLY_FFMPEG_DIR")); ok {
		return dir, nil
	}
	if exe, err := os.Executable(); err == nil {
		if dir, ok := consider(filepath.Join(filepath.Dir(exe), "ffmpeg")); ok {
			return dir, nil
		}
	}
	return "", fmt.Errorf("ffmpeg not found; looked in %v", tried)
}

// Tools resolves ffmpeg and ffprobe. It first tries the bundled directory
// (resolveDir); if that fails it falls back to PATH, so a developer with ffmpeg
// installed can work without a bundle. A clear error names where it looked.
func Tools() (Paths, error) {
	if dir, err := resolveDir(); err == nil {
		return Paths{
			FFmpeg:  filepath.Join(dir, "ffmpeg.exe"),
			FFprobe: filepath.Join(dir, "ffprobe.exe"),
		}, nil
	}
	ff, errFF := exec.LookPath("ffmpeg")
	fp, errFP := exec.LookPath("ffprobe")
	if errFF == nil && errFP == nil {
		return Paths{FFmpeg: ff, FFprobe: fp}, nil
	}
	return Paths{}, fmt.Errorf("ffmpeg/ffprobe not found: not beside the exe, not in GIFLY_FFMPEG_DIR, not on PATH")
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/ffmpeg/ -run TestTools -v`
Expected: PASS (both cases).

- [ ] **Step 6: Commit**

```bash
git add go.mod .gitignore internal/ffmpeg/locate.go internal/ffmpeg/locate_test.go
git commit -m "feat(ffmpeg): locate bundled ffmpeg/ffprobe with env and PATH fallback"
```

---

### Task 2: Run ffmpeg and parse its progress stream

**Files:**
- Create: `internal/ffmpeg/run.go`
- Test: `internal/ffmpeg/run_test.go`

**Interfaces:**
- Consumes: `Paths` (Task 1).
- Produces:
  - `type Progress struct { OutTimeMS int64; Frame int; Done bool }`
  - `func parseProgressLine(line string, p *Progress) (emit bool)` - updates p from one `key=value` line; returns true when a progress block is complete (`progress=continue|end`).
  - `func Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error` - runs `bin` with global flags + args, streams progress, returns an error whose message includes ffmpeg's stderr tail on non-zero exit.

- [ ] **Step 1: Write the failing test**

```go
package ffmpeg

import (
	"strings"
	"testing"
)

func TestParseProgressStream(t *testing.T) {
	// ffmpeg -progress writes flat key=value lines, one block per update,
	// terminated by a progress=continue (or progress=end for the last).
	stream := strings.Join([]string{
		"frame=12",
		"out_time_ms=500000",
		"progress=continue",
		"frame=48",
		"out_time_ms=2000000",
		"progress=end",
	}, "\n")

	var got []Progress
	var p Progress
	for _, line := range strings.Split(stream, "\n") {
		if parseProgressLine(line, &p) {
			got = append(got, p)
		}
	}

	if len(got) != 2 {
		t.Fatalf("emitted %d progress blocks, want 2: %+v", len(got), got)
	}
	if got[0].Frame != 12 || got[0].OutTimeMS != 500 || got[0].Done {
		t.Errorf("block 0 = %+v, want frame 12, 500ms, not done", got[0])
	}
	if got[1].Frame != 48 || got[1].OutTimeMS != 2000 || !got[1].Done {
		t.Errorf("block 1 = %+v, want frame 48, 2000ms, done", got[1])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ffmpeg/ -run TestParseProgressStream -v`
Expected: FAIL - `undefined: parseProgressLine`.

- [ ] **Step 3: Implement run and progress parsing**

```go
package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Progress is one update from ffmpeg's -progress stream. OutTimeMS is the output
// timestamp in milliseconds (ffmpeg reports microseconds as out_time_ms - a
// long-standing misnomer - so it is divided by 1000 here). Done is set on the
// terminal block.
type Progress struct {
	OutTimeMS int64
	Frame     int
	Done      bool
}

// parseProgressLine folds one "key=value" line into p. It returns true when the
// line closes a progress block ("progress=continue" or "progress=end"), at which
// point p holds a complete update; Done reflects whether it was the end block.
func parseProgressLine(line string, p *Progress) bool {
	k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok {
		return false
	}
	switch k {
	case "frame":
		if n, err := strconv.Atoi(v); err == nil {
			p.Frame = n
		}
	case "out_time_ms": // ffmpeg reports microseconds under this key
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			p.OutTimeMS = n / 1000
		}
	case "progress":
		p.Done = v == "end"
		return true
	}
	return false
}

// Run executes bin with gifly's standard global flags followed by args, parsing
// the -progress stream on stdout and calling onProgress (if non-nil) for each
// block. On a non-zero exit it returns an error carrying the tail of stderr, so
// the caller can surface the real cause (unsupported codec, missing file, ...).
// A cancelled ctx kills the process.
func Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	full := append([]string{"-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats"}, args...)
	cmd := exec.CommandContext(ctx, bin, full...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ffmpeg: %w", err)
	}

	var p Progress
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if parseProgressLine(sc.Text(), &p) && onProgress != nil {
			onProgress(p)
		}
	}

	if err := cmd.Wait(); err != nil {
		tail := strings.TrimSpace(stderr.String())
		if len(tail) > 500 {
			tail = tail[len(tail)-500:]
		}
		if tail == "" {
			return fmt.Errorf("ffmpeg failed: %w", err)
		}
		return fmt.Errorf("ffmpeg failed: %w: %s", err, tail)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/ffmpeg/ -v`
Expected: PASS (parse test plus Task 1's tests).

- [ ] **Step 5: Commit**

```bash
git add internal/ffmpeg/run.go internal/ffmpeg/run_test.go
git commit -m "feat(ffmpeg): run a command and parse its -progress stream"
```

---

### Task 3: Probe a source's duration and dimensions

**Files:**
- Create: `internal/probe/probe.go`
- Test: `internal/probe/probe_test.go`

**Interfaces:**
- Consumes: `ffmpeg.Run` is not used here; probe calls ffprobe directly via os/exec for a single JSON dump. Consumes nothing from earlier tasks except the module.
- Produces:
  - `type Media struct { DurationMS int64; Width, Height int; FPS float64 }`
  - `type Frame struct { Width, Height int }`
  - `func parseProbeJSON(b []byte) (Media, error)` - pure parse of ffprobe JSON.
  - `func Video(ctx context.Context, ffprobe, path string) (Media, error)` - run ffprobe then parse.
  - `func Image(path string) (Frame, error)` - stdlib decode of a still.

- [ ] **Step 1: Write the failing test**

```go
package probe

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// A trimmed but real-shaped ffprobe JSON dump for one video stream.
const probeJSON = `{
  "streams": [
    {"codec_type": "video", "width": 1920, "height": 1080, "avg_frame_rate": "30000/1001"}
  ],
  "format": {"duration": "12.500000"}
}`

func TestParseProbeJSON(t *testing.T) {
	m, err := parseProbeJSON([]byte(probeJSON))
	if err != nil {
		t.Fatalf("parseProbeJSON = %v", err)
	}
	if m.DurationMS != 12500 {
		t.Errorf("DurationMS = %d, want 12500", m.DurationMS)
	}
	if m.Width != 1920 || m.Height != 1080 {
		t.Errorf("size = %dx%d, want 1920x1080", m.Width, m.Height)
	}
	// 30000/1001 = 29.97
	if m.FPS < 29.9 || m.FPS > 30.0 {
		t.Errorf("FPS = %v, want ~29.97", m.FPS)
	}
}

func TestImageReadsDimensions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 64, 48))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Image(p)
	if err != nil {
		t.Fatalf("Image = %v", err)
	}
	if f.Width != 64 || f.Height != 48 {
		t.Errorf("frame = %dx%d, want 64x48", f.Width, f.Height)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/probe/ -v`
Expected: FAIL - `undefined: parseProbeJSON` / `undefined: Image`.

- [ ] **Step 3: Implement probe**

```go
// Package probe reads a source's shape - a video's duration, dimensions and
// frame rate via ffprobe, or a still image's dimensions via the standard
// library - to populate the conversion controls (trim range, default width).
package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// Media is what gifly needs to know about a video before converting it.
type Media struct {
	DurationMS int64
	Width      int
	Height     int
	FPS        float64
}

// Frame is a still image's pixel size.
type Frame struct {
	Width  int
	Height int
}

type probeDoc struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		AvgFrame  string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// parseProbeJSON turns ffprobe's -print_format json output into a Media. It uses
// the first video stream for size and frame rate and format.duration for length.
func parseProbeJSON(b []byte) (Media, error) {
	var d probeDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return Media{}, fmt.Errorf("probe: bad ffprobe json: %w", err)
	}
	var m Media
	for _, s := range d.Streams {
		if s.CodecType == "video" {
			m.Width, m.Height = s.Width, s.Height
			m.FPS = parseRate(s.AvgFrame)
			break
		}
	}
	if m.Width == 0 {
		return Media{}, fmt.Errorf("probe: no video stream found")
	}
	if secs, err := strconv.ParseFloat(d.Format.Duration, 64); err == nil {
		m.DurationMS = int64(secs*1000 + 0.5)
	}
	return m, nil
}

// parseRate turns ffprobe's "num/den" frame-rate string into fps, tolerating a
// zero denominator (returns 0).
func parseRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	n, err1 := strconv.ParseFloat(num, 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d == 0 {
		return 0
	}
	return n / d
}

// Video runs ffprobe on path and parses the result.
func Video(ctx context.Context, ffprobe, path string) (Media, error) {
	out, err := exec.CommandContext(ctx, ffprobe,
		"-v", "error",
		"-print_format", "json",
		"-show_format", "-show_streams",
		path,
	).Output()
	if err != nil {
		return Media{}, fmt.Errorf("probe: ffprobe on %q failed: %w", path, err)
	}
	return parseProbeJSON(out)
}

// Image decodes just the header of a still to read its dimensions, which also
// validates that the file is a real PNG, JPEG or GIF.
func Image(path string) (Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return Frame{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Frame{}, fmt.Errorf("probe: %q is not a readable image: %w", path, err)
	}
	return Frame{Width: cfg.Width, Height: cfg.Height}, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/probe/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/probe/probe.go internal/probe/probe_test.go
git commit -m "feat(probe): read video shape via ffprobe and image dimensions via stdlib"
```

---

### Task 4: gifjob config types and validation

**Files:**
- Create: `internal/gifjob/config.go`
- Test: `internal/gifjob/config_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `type LoopMode int` with `const ( LoopForever LoopMode = 0; LoopOnce LoopMode = -1 )` (positive values = loop that many times; the int is the literal ffmpeg `-loop` value).
  - `type Quality struct { MaxColors int; Dither bool }`
  - `type VideoConfig struct { Input string; StartMS, EndMS int64; FPS int; Width int; Loop LoopMode; Quality Quality }`
  - `type ImagesConfig struct { Inputs []string; FrameMS int; Width int; Loop LoopMode; Quality Quality }`
  - `type Result struct { Path string; Bytes int64; Width, Height int }`
  - `func DefaultQuality() Quality` - `{MaxColors: 256, Dither: true}`.
  - `func (c VideoConfig) Validate() error`
  - `func (c ImagesConfig) Validate() error`

- [ ] **Step 1: Write the failing test**

```go
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
	if err := (ImagesConfig{Inputs: nil, FrameMS: 100, Width: 400, Quality: DefaultQuality()}).Validate(); err == nil {
		t.Error("empty image list: want an error")
	}
	if err := (ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 0, Width: 400, Quality: DefaultQuality()}).Validate(); err == nil {
		t.Error("zero frame duration: want an error")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/gifjob/ -run Validate -v`
Expected: FAIL - undefined types.

- [ ] **Step 3: Implement the config**

```go
// Package gifjob turns a typed GIF conversion request into the exact ffmpeg
// argument lists and runs the two-pass palettegen/paletteuse pipeline. The
// argument builders (args.go) are pure and are the load-bearing tests: a wrong
// flag is the likeliest defect and the hardest to see by eye.
package gifjob

import "fmt"

// LoopMode is the literal ffmpeg -loop value for a GIF: 0 loops forever, -1
// plays once, and any positive n loops n times.
type LoopMode int

const (
	LoopForever LoopMode = 0
	LoopOnce    LoopMode = -1
)

// Quality controls the palette. MaxColors is 2..256; Dither on uses ffmpeg's
// sierra2_4a, off uses none (smaller, banding on gradients).
type Quality struct {
	MaxColors int
	Dither    bool
}

// DefaultQuality is a full 256-color dithered palette.
func DefaultQuality() Quality { return Quality{MaxColors: 256, Dither: true} }

// VideoConfig is a video-to-GIF request. StartMS/EndMS is the trim window;
// Width is the output width in pixels (height follows, keeping aspect).
type VideoConfig struct {
	Input   string
	StartMS int64
	EndMS   int64
	FPS     int
	Width   int
	Loop    LoopMode
	Quality Quality
}

// ImagesConfig is an images-to-GIF request. FrameMS is how long each frame
// shows; Inputs is the ordered list of image paths.
type ImagesConfig struct {
	Inputs  []string
	FrameMS int
	Width   int
	Loop    LoopMode
	Quality Quality
}

// Result describes a finished GIF.
type Result struct {
	Path   string
	Bytes  int64
	Width  int
	Height int
}

func validateShared(width int, loop LoopMode, q Quality) error {
	if width <= 0 {
		return fmt.Errorf("output width must be positive, got %d", width)
	}
	if loop < LoopOnce {
		return fmt.Errorf("loop must be -1 (once), 0 (forever) or positive, got %d", int(loop))
	}
	if q.MaxColors < 2 || q.MaxColors > 256 {
		return fmt.Errorf("palette colors must be 2..256, got %d", q.MaxColors)
	}
	return nil
}

// Validate refuses a video config that cannot produce a GIF.
func (c VideoConfig) Validate() error {
	if c.Input == "" {
		return fmt.Errorf("no input video")
	}
	if c.EndMS <= c.StartMS {
		return fmt.Errorf("trim end (%d ms) must be after start (%d ms)", c.EndMS, c.StartMS)
	}
	if c.FPS <= 0 {
		return fmt.Errorf("fps must be positive, got %d", c.FPS)
	}
	return validateShared(c.Width, c.Loop, c.Quality)
}

// Validate refuses an images config that cannot produce a GIF.
func (c ImagesConfig) Validate() error {
	if len(c.Inputs) == 0 {
		return fmt.Errorf("no input images")
	}
	if c.FrameMS <= 0 {
		return fmt.Errorf("frame duration must be positive, got %d ms", c.FrameMS)
	}
	return validateShared(c.Width, c.Loop, c.Quality)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -run Validate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/config.go internal/gifjob/config_test.go
git commit -m "feat(gifjob): config types and validation for video/images jobs"
```

---

### Task 5: gifjob argument builders and the images concat list

**Files:**
- Create: `internal/gifjob/args.go`
- Test: `internal/gifjob/args_test.go`

**Interfaces:**
- Consumes: `VideoConfig`, `ImagesConfig`, `Quality`, `LoopMode` (Task 4).
- Produces:
  - `func VideoArgs(c VideoConfig, palettePath, outPath string) (pass1, pass2 []string)` (config assumed already Validate-d).
  - `func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (pass1, pass2 []string)`.
  - `func WriteConcatList(w io.Writer, inputs []string, frameMS int) error`.

- [ ] **Step 1: Write the failing test**

```go
package gifjob

import (
	"reflect"
	"strings"
	"testing"
)

func TestVideoArgsTwoPass(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 1000, EndMS: 3500, FPS: 15, Width: 480, Loop: LoopForever, Quality: Quality{MaxColors: 128, Dither: true}}
	p1, p2 := VideoArgs(c, "pal.png", "out.gif")

	want1 := []string{
		"-y", "-ss", "1.000", "-i", "in.mp4", "-t", "2.500",
		"-vf", "fps=15,scale=480:-2:flags=lanczos,palettegen=max_colors=128:stats_mode=diff",
		"pal.png",
	}
	if !reflect.DeepEqual(p1, want1) {
		t.Errorf("pass1 =\n%v\nwant\n%v", p1, want1)
	}
	want2 := []string{
		"-y", "-ss", "1.000", "-i", "in.mp4", "-t", "2.500", "-i", "pal.png",
		"-lavfi", "fps=15,scale=480:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a",
		"-loop", "0", "out.gif",
	}
	if !reflect.DeepEqual(p2, want2) {
		t.Errorf("pass2 =\n%v\nwant\n%v", p2, want2)
	}
}

func TestVideoArgsDitherOffAndLoopOnce(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 320, Loop: LoopOnce, Quality: Quality{MaxColors: 256, Dither: false}}
	_, p2 := VideoArgs(c, "pal.png", "out.gif")
	joined := strings.Join(p2, " ")
	if !strings.Contains(joined, "paletteuse=dither=none") {
		t.Errorf("dither off should produce dither=none: %s", joined)
	}
	if !strings.Contains(joined, "-loop -1") {
		t.Errorf("LoopOnce should produce -loop -1: %s", joined)
	}
}

func TestImagesArgsAndConcatList(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 400, Loop: 3, Quality: Quality{MaxColors: 256, Dither: true}}
	p1, p2 := ImagesArgs(c, "list.txt", "pal.png", "out.gif")

	want1 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt",
		"-vf", "scale=400:-2:flags=lanczos,palettegen=max_colors=256:stats_mode=diff",
		"pal.png",
	}
	if !reflect.DeepEqual(p1, want1) {
		t.Errorf("images pass1 =\n%v\nwant\n%v", p1, want1)
	}
	want2 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt", "-i", "pal.png",
		"-lavfi", "scale=400:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a",
		"-loop", "3", "out.gif",
	}
	if !reflect.DeepEqual(p2, want2) {
		t.Errorf("images pass2 =\n%v\nwant\n%v", p2, want2)
	}

	var b strings.Builder
	if err := WriteConcatList(&b, c.Inputs, c.FrameMS); err != nil {
		t.Fatal(err)
	}
	want := "file 'a.png'\nduration 0.100\nfile 'b.png'\nduration 0.100\nfile 'b.png'\n"
	if b.String() != want {
		t.Errorf("concat list =\n%q\nwant\n%q", b.String(), want)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/gifjob/ -run "Args|ConcatList" -v`
Expected: FAIL - `undefined: VideoArgs`.

- [ ] **Step 3: Implement the builders**

```go
package gifjob

import (
	"fmt"
	"io"
	"strconv"
)

// secs formats a millisecond count as fixed-point seconds with millisecond
// precision, e.g. 2500 -> "2.500". ffmpeg accepts this for -ss/-t.
func secs(ms int64) string {
	return fmt.Sprintf("%.3f", float64(ms)/1000)
}

func ditherArg(q Quality) string {
	if q.Dither {
		return "sierra2_4a"
	}
	return "none"
}

// scaleChain is the shared scale filter: force width, keep aspect, even height,
// lanczos resampling.
func scaleChain(width int) string {
	return fmt.Sprintf("scale=%d:-2:flags=lanczos", width)
}

func palettegen(width int, q Quality) string {
	return scaleChain(width) + fmt.Sprintf(",palettegen=max_colors=%d:stats_mode=diff", q.MaxColors)
}

func paletteuse(width int, q Quality) string {
	return scaleChain(width) + fmt.Sprintf("[x];[x][1:v]paletteuse=dither=%s", ditherArg(q))
}

// VideoArgs builds the two ffmpeg passes for a video job. Pass 1 writes the
// optimized palette; pass 2 encodes the GIF against it. -ss before -i is a fast
// seek; -t (duration) avoids the -ss/-to origin ambiguity. The config is assumed
// already validated.
func VideoArgs(c VideoConfig, palettePath, outPath string) (pass1, pass2 []string) {
	start := secs(c.StartMS)
	dur := secs(c.EndMS - c.StartMS)
	fps := strconv.Itoa(c.FPS)

	pass1 = []string{
		"-y", "-ss", start, "-i", c.Input, "-t", dur,
		"-vf", "fps=" + fps + "," + palettegen(c.Width, c.Quality),
		palettePath,
	}
	pass2 = []string{
		"-y", "-ss", start, "-i", c.Input, "-t", dur, "-i", palettePath,
		"-lavfi", "fps=" + fps + "," + paletteuse(c.Width, c.Quality),
		"-loop", strconv.Itoa(int(c.Loop)), outPath,
	}
	return pass1, pass2
}

// ImagesArgs builds the two ffmpeg passes for an images job, reading the ordered
// frames through the concat demuxer (frame durations come from listPath, so no
// fps filter is applied).
func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (pass1, pass2 []string) {
	pass1 = []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath,
		"-vf", palettegen(c.Width, c.Quality),
		palettePath,
	}
	pass2 = []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath, "-i", palettePath,
		"-lavfi", paletteuse(c.Width, c.Quality),
		"-loop", strconv.Itoa(int(c.Loop)), outPath,
	}
	return pass1, pass2
}

// WriteConcatList writes the concat-demuxer script: each frame with its duration
// in seconds, and the final frame repeated because the concat demuxer ignores
// the last entry's duration (without the repeat, the last frame flashes for one
// output tick instead of frameMS).
func WriteConcatList(w io.Writer, inputs []string, frameMS int) error {
	dur := secs(int64(frameMS))
	for _, in := range inputs {
		if _, err := fmt.Fprintf(w, "file '%s'\nduration %s\n", in, dur); err != nil {
			return err
		}
	}
	if len(inputs) > 0 {
		if _, err := fmt.Fprintf(w, "file '%s'\n", inputs[len(inputs)-1]); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -v`
Expected: PASS (args, concat list, plus Task 4 validation).

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/args.go internal/gifjob/args_test.go
git commit -m "feat(gifjob): exact ffmpeg argv builders and the images concat list"
```

---

### Task 6: gifjob orchestration (run both passes) with a fake runner and a gated real test

**Files:**
- Create: `internal/gifjob/run.go`
- Test: `internal/gifjob/run_test.go`
- Test: `internal/gifjob/integration_test.go` (build-tagged `//go:build ffmpeg`)

**Interfaces:**
- Consumes: `ffmpeg.Paths`, `ffmpeg.Progress` (Tasks 1-2); all gifjob types and builders (Tasks 4-5).
- Produces:
  - `type Runner interface { Run(ctx context.Context, bin string, args []string, onProgress func(ffmpeg.Progress)) error }`
  - `func RunVideo(ctx context.Context, tools ffmpeg.Paths, r Runner, c VideoConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error)`
  - `func RunImages(ctx context.Context, tools ffmpeg.Paths, r Runner, c ImagesConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error)`
  - `ffmpeg.Run` satisfies `Runner` via `ffmpeg.RunnerFunc` - add `type RunnerFunc func(...) error` and its `Run` method to `internal/ffmpeg/run.go` in this task (small addition).

- [ ] **Step 1: Add the RunnerFunc adapter to ffmpeg**

Append to `internal/ffmpeg/run.go`:
```go
// RunnerFunc adapts the package-level Run into an interface value the gifjob
// orchestrator can accept, so tests can substitute a fake that records argv and
// never launches a process.
type RunnerFunc func(ctx context.Context, bin string, args []string, onProgress func(Progress)) error

func (f RunnerFunc) Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	return f(ctx, bin, args, onProgress)
}
```

- [ ] **Step 2: Write the failing test (fake runner)**

```go
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
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/gifjob/ -run RunVideo -v`
Expected: FAIL - `undefined: RunVideo`.

- [ ] **Step 4: Implement the orchestrator**

```go
package gifjob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// Runner runs one ffmpeg invocation. ffmpeg.RunnerFunc(ffmpeg.Run) is the real
// implementation; tests pass a fake.
type Runner interface {
	Run(ctx context.Context, bin string, args []string, onProgress func(ffmpeg.Progress)) error
}

// statResult stats a finished GIF and reads its dimensions back with the
// standard library, so a Result reports what was actually written.
func statResult(path string) (Result, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("output not produced: %w", err)
	}
	res := Result{Path: path, Bytes: fi.Size()}
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		if cfg, _, err := decodeConfig(f); err == nil {
			res.Width, res.Height = cfg.Width, cfg.Height
		}
	}
	return res, nil
}

// RunVideo validates the config, runs the palette pass then the encode pass
// against a temp palette file, cleans the palette up on every path, and stats
// the result.
func RunVideo(ctx context.Context, tools ffmpeg.Paths, r Runner, c VideoConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error) {
	if err := c.Validate(); err != nil {
		return Result{}, err
	}
	palette, err := os.CreateTemp("", "gifly-*.png")
	if err != nil {
		return Result{}, err
	}
	palettePath := palette.Name()
	palette.Close()
	defer os.Remove(palettePath)

	p1, p2 := VideoArgs(c, palettePath, outPath)
	if err := r.Run(ctx, tools.FFmpeg, p1, nil); err != nil {
		return Result{}, fmt.Errorf("palette pass: %w", err)
	}
	if err := r.Run(ctx, tools.FFmpeg, p2, onProgress); err != nil {
		return Result{}, fmt.Errorf("encode pass: %w", err)
	}
	return statResult(outPath)
}

// RunImages is RunVideo's sibling for an ordered image set: it also writes a
// temp concat list, and cleans both temp files up.
func RunImages(ctx context.Context, tools ffmpeg.Paths, r Runner, c ImagesConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error) {
	if err := c.Validate(); err != nil {
		return Result{}, err
	}
	list, err := os.CreateTemp("", "gifly-*.txt")
	if err != nil {
		return Result{}, err
	}
	listPath := list.Name()
	defer os.Remove(listPath)
	// Absolute paths so the list works regardless of ffmpeg's working dir.
	abs := make([]string, len(c.Inputs))
	for i, in := range c.Inputs {
		if a, err := filepath.Abs(in); err == nil {
			abs[i] = a
		} else {
			abs[i] = in
		}
	}
	if err := WriteConcatList(list, abs, c.FrameMS); err != nil {
		list.Close()
		return Result{}, err
	}
	list.Close()

	palette, err := os.CreateTemp("", "gifly-*.png")
	if err != nil {
		return Result{}, err
	}
	palettePath := palette.Name()
	palette.Close()
	defer os.Remove(palettePath)

	p1, p2 := ImagesArgs(c, listPath, palettePath, outPath)
	if err := r.Run(ctx, tools.FFmpeg, p1, nil); err != nil {
		return Result{}, fmt.Errorf("palette pass: %w", err)
	}
	if err := r.Run(ctx, tools.FFmpeg, p2, onProgress); err != nil {
		return Result{}, fmt.Errorf("encode pass: %w", err)
	}
	return statResult(outPath)
}
```

Also add, at the top of `run.go`, the stdlib GIF decoder wiring used by statResult:
```go
import (
	"image"
	_ "image/gif"
)

// decodeConfig is image.DecodeConfig, named locally so statResult reads clearly.
var decodeConfig = image.DecodeConfig
```
(Place these with the other imports; `var decodeConfig = image.DecodeConfig` sits at package scope.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -v`
Expected: PASS.

- [ ] **Step 6: Write the gated integration test**

```go
//go:build ffmpeg

// This test runs only with -tags ffmpeg AND a real ffmpeg reachable (bundled,
// GIFLY_FFMPEG_DIR, or PATH). It proves the argv actually produce a GIF - the
// one thing the unit tests, which assert argv against expected argv, cannot.
package gifjob

import (
	"context"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

func TestImagesProduceARealGIF(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg available: %v", err)
	}
	dir := t.TempDir()
	// Two solid-color PNGs via ffmpeg's lavfi color source, so the test needs no
	// fixtures checked in.
	for i, col := range []string{"red", "blue"} {
		p := filepath.Join(dir, col+".png")
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", "color=c=" + col + ":s=64x48:d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("making test image %d: %v", i, err)
		}
	}
	out := filepath.Join(dir, "out.gif")
	c := ImagesConfig{
		Inputs:  []string{filepath.Join(dir, "red.png"), filepath.Join(dir, "blue.png")},
		FrameMS: 200, Width: 64, Loop: LoopForever, Quality: DefaultQuality(),
	}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil)
	if err != nil {
		t.Fatalf("RunImages = %v", err)
	}
	f, err := os.Open(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("output is not a valid GIF: %v", err)
	}
	if len(g.Image) < 2 {
		t.Errorf("GIF has %d frames, want at least 2", len(g.Image))
	}
	if res.Width != 64 {
		t.Errorf("GIF width = %d, want 64", res.Width)
	}
}
```

- [ ] **Step 7: Run the unit tests (the gated test is skipped without the tag)**

Run: `CGO_ENABLED=0 go test ./... -v`
Expected: PASS. The integration test does not compile into this run (no `-tags ffmpeg`), so it neither runs nor fails.

- [ ] **Step 8: Commit**

```bash
git add internal/ffmpeg/run.go internal/gifjob/run.go internal/gifjob/run_test.go internal/gifjob/integration_test.go
git commit -m "feat(gifjob): orchestrate the two-pass run; fake-runner unit test and a gated real-ffmpeg test"
```

---

### Task 7: giflycli end-to-end harness

**Files:**
- Create: `cmd/giflycli/main.go`
- Test: `cmd/giflycli/main_test.go`

**Interfaces:**
- Consumes: everything above (`ffmpeg.Tools`, `ffmpeg.Run`, `probe.Video`, `gifjob.RunVideo`, `gifjob.RunImages`).
- Produces: a command-line entry point; `func parseLoop(s string) (gifjob.LoopMode, error)` is the one piece of pure flag logic worth a unit test.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cmd/giflycli/ -v`
Expected: FAIL - `undefined: parseLoop`.

- [ ] **Step 3: Implement the CLI**

```go
// Command giflycli drives a GIF conversion from the command line - the engine's
// end-to-end harness before any GUI exists, and the way the manual gate below is
// run against a real ffmpeg.
//
//	giflycli video -i in.mp4 -o out.gif -ss 1000 -to 3500 -fps 15 -w 480 [-loop forever] [-colors 256] [-nodither]
//	giflycli images -o out.gif -ms 100 -w 400 [-loop forever] a.png b.png c.png
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
	"github.com/hoijun-kim/gifly/internal/gifjob"
	"github.com/hoijun-kim/gifly/internal/probe"
)

// parseLoop turns the -loop flag into a LoopMode: "forever", "once", or a
// non-negative integer count.
func parseLoop(s string) (gifjob.LoopMode, error) {
	switch s {
	case "forever":
		return gifjob.LoopForever, nil
	case "once":
		return gifjob.LoopOnce, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("loop must be forever, once, or a non-negative count, got %q", s)
	}
	return gifjob.LoopMode(n), nil
}

func quality(colors int, noDither bool) gifjob.Quality {
	return gifjob.Quality{MaxColors: colors, Dither: !noDither}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: giflycli <video|images> ...")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "giflycli:", err)
		os.Exit(1)
	}
}

func run(sub string, argv []string) error {
	tools, err := ffmpeg.Tools()
	if err != nil {
		return err
	}
	onProg := func(p ffmpeg.Progress) { fmt.Printf("\r  %d ms encoded", p.OutTimeMS) }

	switch sub {
	case "video":
		fs := flag.NewFlagSet("video", flag.ContinueOnError)
		in := fs.String("i", "", "input video")
		out := fs.String("o", "out.gif", "output gif")
		ss := fs.Int64("ss", 0, "trim start (ms)")
		to := fs.Int64("to", 0, "trim end (ms); 0 = end of video")
		fps := fs.Int("fps", 15, "output fps")
		w := fs.Int("w", 0, "output width; 0 = source width")
		loopS := fs.String("loop", "forever", "forever | once | N")
		colors := fs.Int("colors", 256, "palette colors 2..256")
		noDither := fs.Bool("nodither", false, "disable dithering")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
		if err != nil {
			return err
		}
		m, err := probe.Video(context.Background(), tools.FFprobe, *in)
		if err != nil {
			return err
		}
		end := *to
		if end == 0 {
			end = m.DurationMS
		}
		width := *w
		if width == 0 {
			width = m.Width
		}
		c := gifjob.VideoConfig{Input: *in, StartMS: *ss, EndMS: end, FPS: *fps, Width: width, Loop: loop, Quality: quality(*colors, *noDither)}
		res, err := gifjob.RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, *out, onProg)
		if err != nil {
			return err
		}
		fmt.Printf("\rwrote %s  %dx%d  %d bytes\n", res.Path, res.Width, res.Height, res.Bytes)
		return nil

	case "images":
		fs := flag.NewFlagSet("images", flag.ContinueOnError)
		out := fs.String("o", "out.gif", "output gif")
		ms := fs.Int("ms", 100, "per-frame duration (ms)")
		w := fs.Int("w", 480, "output width")
		loopS := fs.String("loop", "forever", "forever | once | N")
		colors := fs.Int("colors", 256, "palette colors 2..256")
		noDither := fs.Bool("nodither", false, "disable dithering")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
		if err != nil {
			return err
		}
		c := gifjob.ImagesConfig{Inputs: fs.Args(), FrameMS: *ms, Width: *w, Loop: loop, Quality: quality(*colors, *noDither)}
		res, err := gifjob.RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, *out, onProg)
		if err != nil {
			return err
		}
		fmt.Printf("\rwrote %s  %dx%d  %d bytes\n", res.Path, res.Width, res.Height, res.Bytes)
		return nil

	default:
		return fmt.Errorf("unknown subcommand %q (want video or images)", sub)
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `CGO_ENABLED=0 go test ./cmd/giflycli/ -v`
Expected: PASS.

- [ ] **Step 5: Build the whole module and vet it**

Run: `CGO_ENABLED=0 go build ./... && go vet ./... && gofmt -l .`
Expected: builds clean, vet silent, `gofmt -l` prints nothing.

- [ ] **Step 6: Commit**

```bash
git add cmd/giflycli/main.go cmd/giflycli/main_test.go
git commit -m "feat(cli): giflycli end-to-end harness for video and images jobs"
```

---

## Manual Gate (user-owned, after Task 7)

The unit tests assert argv against expected argv; they cannot prove ffmpeg
accepts those args or that the GIF looks right. This gate does, and it also
front-loads obtaining the ffmpeg that Plan 3 will bundle.

1. Obtain a Windows **LGPL** ffmpeg build (ffmpeg.exe + ffprobe.exe) and put the
   two files in a folder, e.g. `C:\ffmpeg-dev\`.
2. `set GIFLY_FFMPEG_DIR=C:\ffmpeg-dev` (or PowerShell `$env:GIFLY_FFMPEG_DIR="C:\ffmpeg-dev"`).
3. Run the gated integration test:
   `go test -tags ffmpeg ./internal/gifjob/ -run RealGIF -v` (expect PASS).
4. Real images: `go run ./cmd/giflycli images -o a.gif -ms 120 -w 480 <a.png> <b.png> <c.png>` and open `a.gif` - it should loop the frames at the set pace.
5. Real video: `go run ./cmd/giflycli video -i <some.mp4> -o v.gif -ss 1000 -to 4000 -fps 15 -w 480` and open `v.gif` - it should be the trimmed segment, smooth, well-colored.

Record the ffmpeg build used (source + version) - Plan 3 bundles that same build.

---

## Subsequent plans (not this one)

- **Plan 2 - GUI:** the Wails app and Svelte front end over this engine (mode
  switch, pickers, controls, trim handles, reorderable image list, preview,
  progress, results), plus `internal/app` bindings.
- **Plan 3 - Brand and distribution:** icon and logo, the bundled LGPL ffmpeg
  under `ffmpeg/` with its license, landing site, blog row, and a checksummed
  release zip.
