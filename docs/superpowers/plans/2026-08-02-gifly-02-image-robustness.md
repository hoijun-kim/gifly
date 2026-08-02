# gifly Plan 2 - Image Robustness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the images-to-GIF path accept an arbitrary set of differently-sized (or differently-shaped) images by normalizing every frame onto one shared canvas before encoding, clean up a partially-written output on a failed encode, and close the one untested error path in the ffmpeg runner.

**Architecture:** Keep the concat-demuxer pipeline (its per-frame `duration` gives clean GIF frame delays), but pre-normalize each input image to a common width x height (scale-to-fit inside the canvas, then pad) into a temp directory, so the demuxer - which requires identical frame dimensions - accepts a mixed-size set. The canvas height is derived from the first frame's aspect. Partial-output removal and a TestHelperProcess-based runner test round out the engine before the GUI (Plan 3) feeds it arbitrary user-picked files.

**Tech Stack:** Go 1.26, standard library, the external ffmpeg the engine already drives. No GUI in this plan.

## Global Constraints

- Windows only. Go 1.26. Module `github.com/hoijun-kim/gifly`.
- cgo-free: `CGO_ENABLED=0 go build ./...` and `go test ./...` must pass. Standard library only - no new module `require` entries in this plan (Wails arrives in Plan 3). No `golang.org/x/image`.
- Plain ASCII `-` only in code and comments; never a unicode dash. NOTE: a doc comment must never contain two ADJACENT single quotes - Go's doc-comment formatter rewrites `''` to a unicode quote and then `gofmt` fails or injects unicode. Describe escape sequences in words.
- `gofmt -l .` must print nothing; keep Go sources LF (`.gitattributes` pins `eol=lf`).
- Commits carry NO AI co-author trailer.
- Real ffmpeg for the gated tests / manual checks is at `C:\Users\hoijun\ffmpeg-dev\ffmpeg-master-latest-win64-lgpl\bin` - set `GIFLY_FFMPEG_DIR` to it. The plain `go test ./...` run (no `-tags ffmpeg`) must stay green WITHOUT ffmpeg.

## Existing interfaces this plan changes (from Plan 1, on master)

- `internal/gifjob/config.go`: `type ImagesConfig struct { Inputs []string; FrameMS int; Width int; Loop LoopMode; Quality Quality }`, its `Validate()`, `validateShared(width int, loop LoopMode, q Quality) error`, `DefaultQuality()`, `LoopForever`, `LoopOnce`, `Quality{MaxColors,Dither}`, `Result`.
- `internal/gifjob/args.go`: `ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (pass1, pass2 []string)`, `WriteConcatList(w io.Writer, inputs []string, frameMS int) error`, `secs`, `scaleChain`, `palettegen`, `paletteuse`.
- `internal/gifjob/run.go`: `RunVideo(...)`, `RunImages(ctx, tools ffmpeg.Paths, r Runner, c ImagesConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error)`, `Runner` interface, `statResult`.
- `internal/ffmpeg/run.go`: `Run(ctx, bin string, args []string, onProgress func(Progress)) error`, `RunnerFunc`, `Progress`.
- `cmd/giflycli/main.go`: the `images` subcommand builds `ImagesConfig` from positional args.

---

### Task 1: Canvas sizing and the per-frame normalize command

**Files:**
- Modify: `internal/gifjob/config.go` (add `Height` to `ImagesConfig`, add `CanvasHeight`, extend `ImagesConfig.Validate`)
- Modify: `internal/gifjob/args.go` (add `NormalizeArgs`)
- Test: `internal/gifjob/config_test.go` (extend), `internal/gifjob/args_test.go` (extend)

**Interfaces:**
- Consumes: `ImagesConfig`, `validateShared` (Plan 1).
- Produces:
  - `ImagesConfig` gains a field `Height int` (the shared canvas height in pixels).
  - `func CanvasHeight(srcW, srcH, outW int) int` - even output height from the first frame's aspect.
  - `func NormalizeArgs(input, output string, w, h int) []string` - ffmpeg args to fit+pad one image to w x h.

- [ ] **Step 1: Write the failing tests**

Add to `internal/gifjob/config_test.go`:
```go
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
```

Add to `internal/gifjob/args_test.go`:
```go
func TestNormalizeArgs(t *testing.T) {
	got := NormalizeArgs("in.png", "out.png", 400, 300)
	want := []string{
		"-y", "-i", "in.png",
		"-vf", "scale=400:300:force_original_aspect_ratio=decrease,pad=400:300:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1",
		"out.png",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NormalizeArgs =\n%v\nwant\n%v", got, want)
	}
}
```
(`reflect` is already imported in args_test.go.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/gifjob/ -run "CanvasHeight|RequiresHeight|NormalizeArgs" -v`
Expected: FAIL - undefined `CanvasHeight` / `NormalizeArgs`, and the `Height` field does not exist.

- [ ] **Step 3: Implement**

In `internal/gifjob/config.go` add `"math"` to the import block, add the `Height` field to `ImagesConfig` (right after `Width int`):
```go
	Width   int
	Height  int
```
Extend `ImagesConfig.Validate` - add this check before the `return validateShared(...)` line:
```go
	if c.Height <= 0 {
		return fmt.Errorf("canvas height must be positive, got %d", c.Height)
	}
```
Add:
```go
// CanvasHeight returns the even output height for an images GIF of width outW,
// derived from the first frame's aspect ratio (srcW by srcH). Every frame is
// later scaled to fit inside outW by this height and padded, so a set of
// differently-sized images shares one canvas. It never returns less than 2, and
// rounds up to an even number because the GIF scaler needs even dimensions.
func CanvasHeight(srcW, srcH, outW int) int {
	if srcW <= 0 || srcH <= 0 || outW <= 0 {
		return 2
	}
	h := int(math.Round(float64(outW) * float64(srcH) / float64(srcW)))
	if h < 2 {
		h = 2
	}
	if h%2 != 0 {
		h++
	}
	return h
}
```

In `internal/gifjob/args.go` add:
```go
// NormalizeArgs builds the ffmpeg call that fits one source image inside a
// w by h canvas (scaled down to preserve aspect, never up-cropped) and pads it
// with black to exactly w by h. A set of differently-sized frames becomes
// uniform this way, which the concat demuxer requires before it will read them.
func NormalizeArgs(input, output string, w, h int) []string {
	vf := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1",
		w, h, w, h)
	return []string{"-y", "-i", input, "-vf", vf, output}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -v`
Expected: PASS (new tests plus all Plan-1 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/config.go internal/gifjob/args.go internal/gifjob/config_test.go internal/gifjob/args_test.go
git commit -m "feat(gifjob): canvas height and a per-frame normalize command for mixed-size images"
```

---

### Task 2: Normalize each frame in RunImages before the demuxer

**Files:**
- Modify: `internal/gifjob/run.go` (`RunImages`)
- Test: `internal/gifjob/run_test.go` (update the RunImages fake-runner tests for the new call sequence)
- Test: `internal/gifjob/integration_test.go` (add a gated mixed-size test)

**Interfaces:**
- Consumes: `NormalizeArgs`, `CanvasHeight` (Task 1), `ImagesArgs`, `WriteConcatList`, `statResult`, `Runner` (Plan 1).
- Produces: a `RunImages` that runs one normalize pass per input, then the palette and encode passes, over a single temp directory.

- [ ] **Step 1: Update the fake-runner test to the new call sequence**

The fake runner in `run_test.go` writes the stub output on the encode call. With N inputs, RunImages now calls the runner N (normalize) + 2 (palette, encode) times, and the encode is the LAST call. Change the fake so it writes the stub on the FINAL expected call rather than hard-coding call index 2. Replace the existing `fakeRunner` definition with:
```go
type fakeRunner struct {
	calls    [][]string
	out      string
	writeOn  int // 1-based call index that writes the stub output; 0 = never
}

func (r *fakeRunner) Run(_ context.Context, _ string, args []string, _ func(ffmpeg.Progress)) error {
	r.calls = append(r.calls, args)
	if r.writeOn != 0 && len(r.calls) == r.writeOn {
		return os.WriteFile(r.out, []byte("GIF89a-stub"), 0o644)
	}
	return nil
}
```
Update the existing `TestRunVideoRunsBothPasses` fake construction to set `writeOn: 2` (video: palette + encode, encode is call 2). Update `TestRunImagesRunsBothPasses` to:
```go
func TestRunImagesRunsBothPasses(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	before := giflyTempFiles(t)
	// 2 images -> 2 normalize calls + palette + encode = 4 calls; encode is last.
	r := &fakeRunner{out: out, writeOn: 4}
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 320, Height: 240, Loop: LoopForever, Quality: DefaultQuality()}

	res, err := RunImages(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunImages = %v", err)
	}
	if len(r.calls) != 4 {
		t.Fatalf("expected 2 normalize + palette + encode = 4 calls, got %d", len(r.calls))
	}
	// Calls 1 and 2 are the per-image normalize passes (last arg is a temp png,
	// not the output); the final call's last arg is the output GIF.
	if r.calls[0][len(r.calls[0])-1] == out || r.calls[1][len(r.calls[1])-1] == out {
		t.Error("a normalize call wrote the output GIF; expected a temp frame")
	}
	if r.calls[3][len(r.calls[3])-1] != out {
		t.Errorf("final (encode) call output = %q, want %q", r.calls[3][len(r.calls[3])-1], out)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Errorf("result = %+v, want a non-empty file at %q", res, out)
	}
	assertNoNewGiflyTempFiles(t, before)
}
```
`TestRunImagesRejectsInvalidConfig` stays as is except it must now supply a valid `Height` to reach the runner in the valid case (it uses an empty config, which is invalid, so no change needed) - leave it, but confirm it still asserts `len(r.calls) == 0`.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/gifjob/ -run RunImages -v`
Expected: FAIL - RunImages still uses the old 2-call demuxer path and the `Height` field / new call count do not line up.

- [ ] **Step 3: Rewrite RunImages**

Replace `RunImages` in `internal/gifjob/run.go` with:
```go
// RunImages validates the config, normalizes every input frame onto the shared
// c.Width by c.Height canvas (so a mixed-size set becomes uniform for the concat
// demuxer), then runs the palette and encode passes. All temp files live in one
// directory removed on every exit path; a failed encode also removes the partial
// output.
func RunImages(ctx context.Context, tools ffmpeg.Paths, r Runner, c ImagesConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error) {
	if err := c.Validate(); err != nil {
		return Result{}, err
	}
	tmp, err := os.MkdirTemp("", "gifly-imgs-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tmp)

	norm := make([]string, len(c.Inputs))
	for i, in := range c.Inputs {
		abs := in
		if a, err := filepath.Abs(in); err == nil {
			abs = a
		}
		out := filepath.Join(tmp, fmt.Sprintf("f%04d.png", i))
		if err := r.Run(ctx, tools.FFmpeg, NormalizeArgs(abs, out, c.Width, c.Height), nil); err != nil {
			return Result{}, fmt.Errorf("normalizing frame %d: %w", i+1, err)
		}
		norm[i] = out
	}

	listPath := filepath.Join(tmp, "list.txt")
	list, err := os.Create(listPath)
	if err != nil {
		return Result{}, err
	}
	if err := WriteConcatList(list, norm, c.FrameMS); err != nil {
		list.Close()
		return Result{}, err
	}
	list.Close()

	palettePath := filepath.Join(tmp, "palette.png")
	p1, p2 := ImagesArgs(c, listPath, palettePath, outPath)
	if err := r.Run(ctx, tools.FFmpeg, p1, nil); err != nil {
		return Result{}, fmt.Errorf("palette pass: %w", err)
	}
	if err := r.Run(ctx, tools.FFmpeg, p2, onProgress); err != nil {
		os.Remove(outPath)
		return Result{}, fmt.Errorf("encode pass: %w", err)
	}
	return statResult(outPath)
}
```
(The `filepath` import is already present. This drops the old `os.CreateTemp` list/palette handling in favor of the single temp dir.)

- [ ] **Step 4: Run the unit tests**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -v`
Expected: PASS.

- [ ] **Step 5: Add the gated mixed-size integration test**

Append to `internal/gifjob/integration_test.go` (same `//go:build ffmpeg` file):
```go
func TestImagesOfDifferentSizesProduceARealGIF(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg available: %v", err)
	}
	dir := t.TempDir()
	// Two deliberately different-sized frames - the case the concat demuxer
	// rejects without normalization.
	sizes := []struct {
		name string
		spec string
	}{
		{"wide.png", "color=c=red:s=300x120"},
		{"tall.png", "color=c=blue:s=120x300"},
	}
	for _, s := range sizes {
		p := filepath.Join(dir, s.name)
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", s.spec + ":d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("making %s: %v", s.name, err)
		}
	}
	out := filepath.Join(dir, "out.gif")
	c := ImagesConfig{
		Inputs:  []string{filepath.Join(dir, "wide.png"), filepath.Join(dir, "tall.png")},
		FrameMS: 200, Width: 240, Height: 160, Loop: LoopForever, Quality: DefaultQuality(),
	}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil)
	if err != nil {
		t.Fatalf("RunImages on mixed sizes = %v", err)
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
	if res.Width != 240 || res.Height != 160 {
		t.Errorf("GIF is %dx%d, want the 240x160 canvas", res.Width, res.Height)
	}
}
```

- [ ] **Step 6: Run the plain suite (gated test excluded) and commit**

Run: `CGO_ENABLED=0 go test ./... -count=1` (expect PASS; the ffmpeg-tagged test does not compile in).
Then, if ffmpeg is available, optionally: `set GIFLY_FFMPEG_DIR=C:\Users\hoijun\ffmpeg-dev\ffmpeg-master-latest-win64-lgpl\bin` and `go test -tags ffmpeg ./internal/gifjob/ -run DifferentSizes -v` (expect PASS).
```bash
git add internal/gifjob/run.go internal/gifjob/run_test.go internal/gifjob/integration_test.go
git commit -m "feat(gifjob): normalize frames onto a shared canvas so mixed-size images encode"
```

---

### Task 3: Remove a partial output on a failed video encode

**Files:**
- Modify: `internal/gifjob/run.go` (`RunVideo` - RunImages already got this in Task 2)
- Test: `internal/gifjob/run_test.go`

**Interfaces:**
- Consumes: `RunVideo`, `fakeRunner` (updated in Task 2).
- Produces: `RunVideo` removes `outPath` when the encode pass fails.

- [ ] **Step 1: Write the failing test**

Add to `internal/gifjob/run_test.go`:
```go
// failOnEncode is a runner that writes a partial output on the encode pass and
// then reports failure, to prove RunVideo removes the half-written file.
type failOnEncode struct {
	out       string
	encodeIdx int
	calls     int
}

func (r *failOnEncode) Run(_ context.Context, _ string, _ []string, _ func(ffmpeg.Progress)) error {
	r.calls++
	if r.calls == r.encodeIdx {
		_ = os.WriteFile(r.out, []byte("partial"), 0o644)
		return errors.New("ffmpeg failed mid-encode")
	}
	return nil
}

func TestRunVideoRemovesPartialOutputOnFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.gif")
	r := &failOnEncode{out: out, encodeIdx: 2} // video: palette(1), encode(2)
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 320, Loop: LoopForever, Quality: DefaultQuality()}

	if _, err := RunVideo(context.Background(), tools, r, c, out, nil); err == nil {
		t.Fatal("RunVideo should return the encode error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("partial output %q was left behind after a failed encode", out)
	}
}
```
Add `"errors"` to the `run_test.go` imports if not present.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/gifjob/ -run RemovesPartial -v`
Expected: FAIL - the partial file still exists.

- [ ] **Step 3: Implement**

In `RunVideo`, change the encode-pass error branch to remove the output first:
```go
	if err := r.Run(ctx, tools.FFmpeg, p2, onProgress); err != nil {
		os.Remove(outPath)
		return Result{}, fmt.Errorf("encode pass: %w", err)
	}
```

- [ ] **Step 4: Run the tests**

Run: `CGO_ENABLED=0 go test ./internal/gifjob/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/run.go internal/gifjob/run_test.go
git commit -m "fix(gifjob): remove a partial output GIF when the video encode fails"
```

---

### Task 4: Unit-test the ffmpeg runner's error and cancel paths

**Files:**
- Test: `internal/ffmpeg/run_test.go` (add a TestHelperProcess-based test; no production change)

**Interfaces:**
- Consumes: `Run`, `Progress` (Plan 1). Uses the standard os/exec `TestHelperProcess` re-exec pattern so a real ffmpeg is not needed.
- Produces:
  - a small refactor of `Run` in run.go: `Run` prepends the global flags and delegates to a new unexported `runArgs(ctx, bin string, args []string, onProgress func(Progress)) error` that does the exec + progress parsing. This lets the tests exercise the exec/parse body WITHOUT the global ffmpeg flags (which a Go test binary re-exec would reject).
  - coverage for the non-zero-exit stderr-tail message, the progress callback, and ctx-cancel-kills-process.

- [ ] **Step 1: Split Run so the exec body is callable without the global flags**

In `internal/ffmpeg/run.go`, keep `Run`'s signature and behavior but move its body into `runArgs`:
```go
// Run executes bin with gifly's standard global flags followed by args (see
// runArgs for the streaming/error behavior).
func Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	full := append([]string{"-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats"}, args...)
	return runArgs(ctx, bin, full, onProgress)
}

// runArgs runs bin with exactly args (no flags prepended), parses the -progress
// stream on stdout into onProgress, kills the process on ctx cancel, and on a
// non-zero exit returns an error carrying the tail of stderr. Run adds the
// global flags; tests call runArgs directly with a test-binary re-exec, which
// must not receive those flags.
func runArgs(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	// ... the exact body that Run had before (cmd := exec.CommandContext(ctx, bin, args...); stdout pipe;
	//     stderr builder; Start; scan+parseProgressLine+onProgress; Wait; stderr-tail error) ...
}
```
Move the existing body verbatim into `runArgs` (it already uses the local variable `args`, so only the two lines above are new). Confirm `Run` still compiles and behaves identically for real callers.

- [ ] **Step 2: Write the failing tests**

Ensure `internal/ffmpeg/run_test.go`'s import block contains `context`, `os`, `strings`, `testing`, `time` (Plan 1 already imports `strings` and `testing`; add the other three). Add:
```go
// TestHelperProcess is not a real test; the tests below re-exec this binary as a
// stand-in for ffmpeg, choosing behavior via GIFLY_HELPER_MODE.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GIFLY_HELPER_WANTED") != "1" {
		return
	}
	switch os.Getenv("GIFLY_HELPER_MODE") {
	case "progress-then-ok":
		os.Stdout.WriteString("frame=5\nout_time_ms=1000000\nprogress=continue\nprogress=end\n")
	case "fail-with-stderr":
		os.Stderr.WriteString("Error: unsupported codec")
		os.Exit(3)
	case "hang":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

// helperRun re-execs this test binary as TestHelperProcess and drives it through
// runArgs (NOT Run - the global ffmpeg flags would be rejected as test flags).
func helperRun(ctx context.Context, mode string, onProgress func(Progress)) error {
	os.Setenv("GIFLY_HELPER_WANTED", "1")
	os.Setenv("GIFLY_HELPER_MODE", mode)
	return runArgs(ctx, os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, onProgress)
}

func TestRunReportsProgressAndSucceeds(t *testing.T) {
	var last Progress
	if err := helperRun(context.Background(), "progress-then-ok", func(p Progress) { last = p }); err != nil {
		t.Fatalf("runArgs = %v, want success", err)
	}
	if last.OutTimeMS != 1000 || !last.Done {
		t.Errorf("last progress = %+v, want 1000ms and Done", last)
	}
}

func TestRunSurfacesStderrTailOnFailure(t *testing.T) {
	err := helperRun(context.Background(), "fail-with-stderr", nil)
	if err == nil {
		t.Fatal("runArgs should fail on a non-zero exit")
	}
	if !strings.Contains(err.Error(), "unsupported codec") {
		t.Errorf("error %q does not carry ffmpeg's stderr tail", err.Error())
	}
}

func TestRunCancelKillsProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- helperRun(ctx, "hang", nil) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a cancelled run should return a non-nil error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after ctx cancel; the process was not killed")
	}
}
```

- [ ] **Step 3: Run to verify they fail, then pass**

Run: `go test ./internal/ffmpeg/ -run "RunReportsProgress|StderrTail|CancelKills" -v`
Before the Step 1 split is in place: FAIL - `undefined: runArgs`.
After Step 1: run again, expect PASS. (`TestHelperProcess` prints the progress stream on stdout, which `runArgs` parses; the fail mode exits 3 with stderr; the hang mode is killed by ctx cancel.)

- [ ] **Step 4: Run the whole ffmpeg suite**

Run: `CGO_ENABLED=0 go test ./internal/ffmpeg/ -v`
Expected: PASS (the three helper tests plus the Plan-1 tests). `gofmt -l internal/ffmpeg/` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/ffmpeg/run.go internal/ffmpeg/run_test.go
git commit -m "test(ffmpeg): cover Run's progress, stderr-tail and ctx-cancel paths without ffmpeg"
```

---

### Task 5: Compute the canvas height in the images CLI

**Files:**
- Modify: `cmd/giflycli/main.go` (the `images` subcommand)
- Test: `cmd/giflycli/main_test.go` (a pure helper test)

**Interfaces:**
- Consumes: `probe.Image` (returns `Frame{Width,Height}`), `gifjob.CanvasHeight`, `gifjob.ImagesConfig.Height`.
- Produces: the `images` subcommand probes the first input, sets `Height` via `CanvasHeight`, and a `-h` flag overrides it.

- [ ] **Step 1: Write the failing test**

Add to `cmd/giflycli/main_test.go`. It already has `import "testing"` and imports `gifjob`; ensure `github.com/hoijun-kim/gifly/internal/gifjob` is in the import block (the existing TestParseLoop already references `gifjob.LoopMode`, so it is). Add:
```go
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
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/giflycli/ -run ImagesHeight -v`
Expected: FAIL - `undefined: imagesHeight`.

- [ ] **Step 3: Implement**

In `cmd/giflycli/main.go` add the helper:
```go
// imagesHeight picks the images canvas height: a positive -h override, else the
// even height derived from the first frame's aspect at the output width.
func imagesHeight(override, width, firstW, firstH int) int {
	if override > 0 {
		return override
	}
	return gifjob.CanvasHeight(firstW, firstH, width)
}
```
Add `"github.com/hoijun-kim/gifly/internal/probe"` to imports if not present (the video subcommand already imports it). In the `images` subcommand, add a `-h` flag (`h := fs.Int("h", 0, "output height; 0 = from the first image")`) and, after parsing and before building the config, probe the first input and set the height:
```go
		if len(fs.Args()) == 0 {
			return fmt.Errorf("no input images")
		}
		first, err := probe.Image(fs.Args()[0])
		if err != nil {
			return err
		}
		height := imagesHeight(*h, *w, first.Width, first.Height)
```
Then add `Height: height` to the `gifjob.ImagesConfig{...}` literal.

- [ ] **Step 4: Run the tests and build**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go build ./... && go vet ./... && gofmt -l .`
Expected: tests PASS, build clean, vet silent, gofmt prints nothing.

- [ ] **Step 5: Commit**

```bash
git add cmd/giflycli/main.go cmd/giflycli/main_test.go
git commit -m "feat(cli): derive the images canvas height from the first frame, with a -h override"
```

---

## Manual Gate (after Task 5)

With `GIFLY_FFMPEG_DIR` set to the LGPL build, prove the mixed-size path end to end:

1. `go test -tags ffmpeg ./internal/gifjob/ -v` - expect the two integration tests (uniform and mixed-size) to PASS.
2. Make two different-sized images and run: `go run ./cmd/giflycli images -o mix.gif -ms 300 -w 300 <big.png> <tall.png>` - it should succeed (no "Internal bug" error) and `mix.gif` should show both frames letterboxed onto one canvas.
3. Confirm a deliberately broken run (e.g. a non-image input) leaves no `mix.gif` behind.

## Next plan (not this one)

- **Plan 3 - Wails GUI:** the desktop app over this engine (mode switch, video/images pickers, trim handles, reorderable image list, controls, live preview, progress, results) plus `internal/app` bindings. This is where arbitrary user-picked images flow in, which is why the normalization above lands first.
