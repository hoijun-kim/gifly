# gifly Plan 4 - Engine v2 (multi-format + rich options) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the validated GIF-only engine into a multi-format (GIF / animated WebP / APNG), option-rich engine - crop/aspect, boomerang, reverse, playback speed, and selectable dither - driven end-to-end by `giflycli` and proven with real ffmpeg, before any GUI work.

**Architecture:** `gifjob` gains a `Format` and a shared video *filter graph* built once and reused across every encode path. GIF stays a two-pass palettegen/paletteuse pipeline; WebP and APNG are single-pass encodes. To carry a variable pass count, `VideoArgs`/`ImagesArgs` change from returning `(pass1, pass2)` to returning `passes [][]string`, and `RunVideo`/`RunImages` loop over them (last pass = the encode, gets progress + partial-output cleanup). Geometry/motion for video (crop, fps, scale, speed, reverse, boomerang) lives in one `filter_complex` producing a labeled `[v]`; for images the same effects are list-level (frame order + per-frame duration) because frames are pre-normalized.

**Tech Stack:** Go 1.26, cgo-free, ffmpeg via `os/exec` (bundled LGPL build has `libwebp_anim` + `apng`), Wails v2 only for the app layer (untouched this plan except a compile-compat mapping).

## Global Constraints

- Module path: `github.com/hoijun-kim/gifly`. Only direct Go dependency allowed: `github.com/wailsapp/wails/v2`. No `golang.org/x/image`, no other third-party imports.
- `CGO_ENABLED=0 go build ./...` MUST pass after every task. `go test ./...` (untagged) MUST pass after every task. `gofmt -l` MUST report no files; `go vet ./...` clean.
- `-race` is unavailable (no C compiler on this machine); do not add `-race` steps.
- Real-ffmpeg tests are tagged `//go:build ffmpeg` and are run by the CONTROLLER, not the implementer (implementers have no ffmpeg). ffmpeg for those runs: `C:\Users\hoijun\ffmpeg-dev\ffmpeg-master-latest-win64-lgpl\bin` (set env `GIFLY_FFMPEG_DIR` to that path).
- ASCII only in code and comments: plain hyphen `-`, never `—`/`–`/`·`. Never put two adjacent single quotes (`''`) in a Go doc comment - gofmt rewrites it to a unicode quote and breaks the gofmt gate.
- No AI/Claude co-author trailer on any commit. Commit messages are plain conventional-commit lines.
- Verified-good ffmpeg argv shapes (all confirmed against the bundled build - use verbatim):
  - GIF pass1: `-filter_complex "<GRAPH>;[v]palettegen=max_colors=N:stats_mode=diff[p]" -map "[p]" palette.png`
  - GIF pass2: `-i in -i palette.png -filter_complex "<GRAPH>;[v][1:v]paletteuse=dither=D[o]" -map "[o]" -loop L out.gif`
  - WebP: `-filter_complex "<GRAPH>" -map "[v]" -c:v libwebp_anim -loop L -q:v Q out.webp`
  - APNG: `-filter_complex "<GRAPH>" -map "[v]" -f apng -plays L out.png`
  - where `<GRAPH>` = `[0:v]<chain>[v]`, or with boomerang `[0:v]<chain>,split[a][b];[b]reverse[r];[a][r]concat=n=2:v=1[v]`
  - `<chain>` order = `crop=...,fps=N,scale=W:-2:flags=lanczos,setpts=F*PTS,reverse` (each piece omitted when not applicable; fps+scale always present)
  - aspect crop exprs (centered, ffmpeg centers when x/y omitted): 1:1 `crop='min(iw,ih)':'min(iw,ih)'`; 16:9 `crop='min(iw,ih*16/9)':'min(ih,iw*9/16)'`; 9:16 `crop='min(iw,ih*9/16)':'min(ih,iw*16/9)'`
  - loop values: GIF `-loop` forever=0 once=-1 N=N; WebP `-loop` forever=0 once=1 N=N; APNG `-plays` forever=0 once=1 N=N
  - dither ffmpeg names: none->`none`, bayer->`bayer`, sierra2->`sierra2_4a`, floyd->`floyd_steinberg`

## File Structure

- `internal/gifjob/config.go` - MODIFY: add `Format`, `Aspect`, `DitherMethod` types + methods; change `Quality` (`Dither bool` -> `Dither DitherMethod`, add `WebPQuality int`); add `SrcWidth/SrcHeight/Aspect/Speed/Reverse/Boomerang/Format` to `VideoConfig`, `Speed/Reverse/Boomerang/Format` to `ImagesConfig`; add `OutputHeight`; make `CanvasHeight` delegate; widen `Validate`.
- `internal/gifjob/args.go` - MODIFY: dither via `Quality.Dither.ffmpeg()`; add `loopArgs`, `palettegenTail`, `paletteuseTail`, `cropExpr`, `setptsExpr`, `filterChain`, `videoGraph`, `frameOrder`, `effectiveFrameMS`; rewrite `VideoArgs`/`ImagesArgs` to return `[][]string`.
- `internal/gifjob/run.go` - MODIFY: `statResult(path, wantW, wantH)`; `RunVideo`/`RunImages` loop passes, palette temp only for GIF, apply order/speed for images, compute `wantH`.
- `internal/gifjob/config_test.go`, `args_test.go`, `run_test.go` - MODIFY tests to new API.
- `internal/gifjob/integration_test.go` - MODIFY/ADD gated real-ffmpeg tests for every new format/option.
- `internal/app/convert.go` - MODIFY: compile-compat only - map `Quality`, default `Format=FormatGIF`, `Speed=1`. (Exposing options to the frontend is Plan 5.)
- `cmd/giflycli/main.go`, `main_test.go` - MODIFY: new flags `-format -aspect -speed -reverse -boomerang -dither -q`, output extension from format, fill `SrcWidth/SrcHeight`.

---

### Task 1: Config types + Quality migration (keep the whole module green)

Widen the type surface and migrate every existing reference so `go build ./...` and `go test ./...` stay green. New config fields are added but not yet consumed by the arg builders (later tasks).

**Files:**
- Modify: `internal/gifjob/config.go`
- Modify: `internal/gifjob/args.go` (only `ditherArg`)
- Modify: `internal/gifjob/config_test.go` (add tests)
- Modify: `internal/gifjob/args_test.go` (fix 2 `Dither:` literals so the package compiles)
- Modify: `internal/app/convert.go` (Quality mapping + Format/Speed defaults)
- Modify: `cmd/giflycli/main.go` (`quality()` builder)

**Interfaces:**
- Produces:
  - `type Format string` with `FormatGIF="gif"`, `FormatWebP="webp"`, `FormatAPNG="apng"`; `func (f Format) Valid() bool`; `func (f Format) Ext() string`.
  - `type Aspect string` with `AspectFree=""`, `AspectSquare="1:1"`, `AspectWide="16:9"`, `AspectTall="9:16"`; `func (a Aspect) Valid() bool`.
  - `type DitherMethod string` with `DitherNone="none"`, `DitherBayer="bayer"`, `DitherSierra="sierra2"`, `DitherFloyd="floyd"`; `func (d DitherMethod) ffmpeg() string`.
  - `type Quality struct { MaxColors int; Dither DitherMethod; WebPQuality int }`; `func DefaultQuality() Quality` = `{256, DitherSierra, 75}`.
  - `VideoConfig` new fields: `SrcWidth, SrcHeight int`, `Aspect Aspect`, `Speed float64`, `Reverse, Boomerang bool`, `Format Format`.
  - `ImagesConfig` new fields: `Speed float64`, `Reverse, Boomerang bool`, `Format Format`.
  - `func OutputHeight(srcW, srcH int, aspect Aspect, outW int) int`; `CanvasHeight` delegates to `OutputHeight(srcW, srcH, AspectFree, outW)`.

- [ ] **Step 1: Write failing tests** (append to `internal/gifjob/config_test.go`)

```go
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
		DitherNone:   "none",
		DitherBayer:  "bayer",
		DitherSierra: "sierra2_4a",
		DitherFloyd:  "floyd_steinberg",
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
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/gifjob/ -run 'TestFormatMethods|TestDitherFFmpegNames|TestOutputHeightAspect|TestDefaultQualityIsDithered'`
Expected: FAIL (undefined: Format, DitherMethod, OutputHeight, etc.). The package will not compile - that is the expected failure.

- [ ] **Step 3: Implement in `internal/gifjob/config.go`**

Replace the `Quality` block and add the new types. New/changed declarations:

```go
// Format is the output animation container.
type Format string

const (
	FormatGIF  Format = "gif"
	FormatWebP Format = "webp"
	FormatAPNG Format = "apng"
)

// Valid reports whether f is one of the three supported formats.
func (f Format) Valid() bool {
	return f == FormatGIF || f == FormatWebP || f == FormatAPNG
}

// Ext is the output file extension for f (APNG files use the .png extension).
// An empty or unknown format falls back to .gif.
func (f Format) Ext() string {
	switch f {
	case FormatWebP:
		return ".webp"
	case FormatAPNG:
		return ".png"
	default:
		return ".gif"
	}
}

// Aspect is an optional center-crop applied before scaling. AspectFree ("")
// keeps the source aspect ratio.
type Aspect string

const (
	AspectFree   Aspect = ""
	AspectSquare Aspect = "1:1"
	AspectWide   Aspect = "16:9"
	AspectTall   Aspect = "9:16"
)

// Valid reports whether a is free or one of the three preset ratios.
func (a Aspect) Valid() bool {
	return a == AspectFree || a == AspectSquare || a == AspectWide || a == AspectTall
}

// DitherMethod selects the GIF paletteuse dithering algorithm.
type DitherMethod string

const (
	DitherNone   DitherMethod = "none"
	DitherBayer  DitherMethod = "bayer"
	DitherSierra DitherMethod = "sierra2"
	DitherFloyd  DitherMethod = "floyd"
)

// ffmpeg maps a DitherMethod to ffmpeg's paletteuse dither name. Empty or
// unknown values default to sierra2_4a (the previous default).
func (d DitherMethod) ffmpeg() string {
	switch d {
	case DitherNone:
		return "none"
	case DitherBayer:
		return "bayer"
	case DitherFloyd:
		return "floyd_steinberg"
	default:
		return "sierra2_4a"
	}
}

// Quality controls the palette (GIF) and the WebP encoder. MaxColors is 2..256.
// Dither picks the GIF dithering algorithm. WebPQuality is 0..100 for libwebp
// (higher is better); it is ignored by GIF and APNG.
type Quality struct {
	MaxColors   int
	Dither      DitherMethod
	WebPQuality int
}

// DefaultQuality is a full 256-color sierra2 palette at WebP quality 75.
func DefaultQuality() Quality { return Quality{MaxColors: 256, Dither: DitherSierra, WebPQuality: 75} }
```

Add the new fields to `VideoConfig` and `ImagesConfig` (keep the existing fields; append these):

```go
// In VideoConfig, add:
//   SrcWidth  int   // source video width, used to compute WebP/APNG output height
//   SrcHeight int   // source video height
//   Aspect    Aspect
//   Speed     float64 // 1.0 = normal; <=0 is treated as 1.0
//   Reverse   bool
//   Boomerang bool
//   Format    Format  // "" is treated as FormatGIF

// In ImagesConfig, add:
//   Speed     float64
//   Reverse   bool
//   Boomerang bool
//   Format    Format
```

Replace `CanvasHeight` with a delegation and add `OutputHeight`:

```go
// OutputHeight returns the even output height for a source of srcW by srcH
// scaled to width outW, after an optional center-crop to aspect. For the
// presets the height follows the target ratio; for AspectFree it follows the
// source ratio. It never returns less than 2 and rounds up to an even number
// because the scalers need even dimensions.
func OutputHeight(srcW, srcH int, aspect Aspect, outW int) int {
	if outW <= 0 {
		return 2
	}
	var h int
	switch aspect {
	case AspectSquare:
		h = outW
	case AspectWide: // 16:9
		h = int(math.Round(float64(outW) * 9 / 16))
	case AspectTall: // 9:16
		h = int(math.Round(float64(outW) * 16 / 9))
	default: // free
		if srcW <= 0 || srcH <= 0 {
			return 2
		}
		h = int(math.Round(float64(outW) * float64(srcH) / float64(srcW)))
	}
	if h < 2 {
		h = 2
	}
	if h%2 != 0 {
		h++
	}
	return h
}

// CanvasHeight is OutputHeight with no crop (source aspect preserved).
func CanvasHeight(srcW, srcH, outW int) int {
	return OutputHeight(srcW, srcH, AspectFree, outW)
}
```

Widen `Validate` for both configs. In `VideoConfig.Validate` and `ImagesConfig.Validate`, before the final `return validateShared(...)`, add:

```go
	if c.Format != "" && !c.Format.Valid() {
		return fmt.Errorf("unknown format %q", string(c.Format))
	}
	if c.Speed < 0 || c.Speed > 4 {
		return fmt.Errorf("speed must be 0..4, got %v", c.Speed)
	}
```

(For `VideoConfig` also add `if c.Aspect != "" && !c.Aspect.Valid() { return fmt.Errorf("unknown aspect %q", string(c.Aspect)) }`.)

- [ ] **Step 4: Fix the two dither literals so the package compiles** (`internal/gifjob/args_test.go`)

`ditherArg` reads the new field via its method. In `args.go`, change the `ditherArg` helper body to:

```go
func ditherArg(q Quality) string {
	return q.Dither.ffmpeg()
}
```

In `args_test.go`, change the two `Quality{MaxColors: 128, Dither: true}` / `Dither: true` occurrences to `Dither: DitherSierra`, and any `Dither: false` to `Dither: DitherNone`. (These video-args tests are fully rewritten in Task 4; this is only to keep the package compiling now.)

- [ ] **Step 5: Fix app + CLI Quality construction** (compile-compat)

In `internal/app/convert.go`, `videoConfig` and `imagesConfig` build a `gifjob.Quality`. Change each to:

```go
	Quality: gifjob.Quality{
		MaxColors:   req.Colors,
		Dither:      ditherFromBool(req.Dither),
		WebPQuality: 75,
	},
```

and set `Format: gifjob.FormatGIF` and `Speed: 1` on the returned config structs. Add the helper to `convert.go`:

```go
// ditherFromBool maps the current boolean dither request onto the engine's
// dither method (Plan 5 will replace this with a real method selector).
func ditherFromBool(on bool) gifjob.DitherMethod {
	if on {
		return gifjob.DitherSierra
	}
	return gifjob.DitherNone
}
```

In `cmd/giflycli/main.go`, change `quality`:

```go
func quality(colors int, noDither bool) gifjob.Quality {
	d := gifjob.DitherSierra
	if noDither {
		d = gifjob.DitherNone
	}
	return gifjob.Quality{MaxColors: colors, Dither: d, WebPQuality: 75}
}
```

Ensure `config.go` still imports `math` (it does) and `fmt`.

- [ ] **Step 6: Run all module tests + build**

Run: `go build ./... && go test ./... && gofmt -l . && go vet ./...`
Expected: build OK, tests PASS, gofmt prints nothing, vet clean.

- [ ] **Step 7: Commit**

```bash
git add internal/gifjob/config.go internal/gifjob/config_test.go internal/gifjob/args.go internal/gifjob/args_test.go internal/app/convert.go cmd/giflycli/main.go
git commit -m "feat(engine): add Format/Aspect/DitherMethod types and widen configs"
```

---

### Task 2: Encode helpers - loop mapping, palette tails

Pure per-format helpers the arg builders share. The loop mapping is the subtle, bug-prone part (once = -1 for GIF but 1 for WebP/APNG) and gets its own test.

**Files:**
- Modify: `internal/gifjob/args.go`
- Modify: `internal/gifjob/args_test.go` (add tests)

**Interfaces:**
- Consumes: `Format`, `LoopMode`, `Quality` (Task 1).
- Produces:
  - `func webpApngLoop(loop LoopMode) int` - forever(0)->0, once(-1)->1, N->N.
  - `func loopArgs(f Format, loop LoopMode) (flag, val string)` - GIF `("-loop", n)`, WebP `("-loop", n)`, APNG `("-plays", n)`.
  - `func palettegenTail(q Quality) string` = `palettegen=max_colors=N:stats_mode=diff`.
  - `func paletteuseTail(q Quality) string` = `paletteuse=dither=<D>`.

- [ ] **Step 1: Write failing tests** (append to `internal/gifjob/args_test.go`)

```go
func TestLoopArgs(t *testing.T) {
	cases := []struct {
		f    Format
		loop LoopMode
		flag string
		val  string
	}{
		{FormatGIF, LoopForever, "-loop", "0"},
		{FormatGIF, LoopOnce, "-loop", "-1"},
		{FormatGIF, LoopMode(5), "-loop", "5"},
		{FormatWebP, LoopForever, "-loop", "0"},
		{FormatWebP, LoopOnce, "-loop", "1"}, // once is 1 for webp, not -1
		{FormatWebP, LoopMode(5), "-loop", "5"},
		{FormatAPNG, LoopForever, "-plays", "0"},
		{FormatAPNG, LoopOnce, "-plays", "1"},
		{FormatAPNG, LoopMode(3), "-plays", "3"},
	}
	for _, c := range cases {
		flag, val := loopArgs(c.f, c.loop)
		if flag != c.flag || val != c.val {
			t.Errorf("loopArgs(%q,%d) = %q %q, want %q %q", c.f, int(c.loop), flag, val, c.flag, c.val)
		}
	}
}

func TestPaletteTails(t *testing.T) {
	q := Quality{MaxColors: 128, Dither: DitherBayer}
	if got := palettegenTail(q); got != "palettegen=max_colors=128:stats_mode=diff" {
		t.Errorf("palettegenTail = %q", got)
	}
	if got := paletteuseTail(q); got != "paletteuse=dither=bayer" {
		t.Errorf("paletteuseTail = %q", got)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/gifjob/ -run 'TestLoopArgs|TestPaletteTails'`
Expected: FAIL (undefined: loopArgs, palettegenTail, paletteuseTail).

- [ ] **Step 3: Implement in `internal/gifjob/args.go`**

```go
// webpApngLoop maps the semantic loop mode to the integer libwebp/apng use:
// forever is 0, once is 1 (one playback), and a positive count is itself. GIF
// does not use this - GIF's -loop takes the LoopMode value directly (once = -1).
func webpApngLoop(loop LoopMode) int {
	if loop == LoopOnce {
		return 1
	}
	if loop < 0 {
		return 0
	}
	return int(loop)
}

// loopArgs returns the loop flag and value for a format. GIF and WebP use
// -loop; APNG uses -plays. The GIF value is the LoopMode literal; WebP and APNG
// translate once to 1.
func loopArgs(f Format, loop LoopMode) (flag, val string) {
	switch f {
	case FormatWebP:
		return "-loop", strconv.Itoa(webpApngLoop(loop))
	case FormatAPNG:
		return "-plays", strconv.Itoa(webpApngLoop(loop))
	default: // GIF
		return "-loop", strconv.Itoa(int(loop))
	}
}

// palettegenTail is the palettegen filter with the configured color count.
func palettegenTail(q Quality) string {
	return fmt.Sprintf("palettegen=max_colors=%d:stats_mode=diff", q.MaxColors)
}

// paletteuseTail is the paletteuse filter with the configured dither method.
func paletteuseTail(q Quality) string {
	return "paletteuse=dither=" + q.Dither.ffmpeg()
}
```

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/gifjob/ -run 'TestLoopArgs|TestPaletteTails'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/args.go internal/gifjob/args_test.go
git commit -m "feat(engine): per-format loop mapping and palette filter tails"
```

---

### Task 3: Video filter graph (crop, speed, reverse, boomerang)

Build the shared `filter_complex` value that every video format consumes.

**Files:**
- Modify: `internal/gifjob/args.go`
- Modify: `internal/gifjob/args_test.go` (add tests)

**Interfaces:**
- Consumes: `VideoConfig`, `Aspect`, `Quality` (Task 1).
- Produces:
  - `func cropExpr(a Aspect) string` - `""` for free, else `crop=...`.
  - `func setptsExpr(speed float64) string` - `""` when speed<=0 or ==1, else `setpts=F*PTS` (F = 1/speed, `%.4f`).
  - `func filterChain(c VideoConfig) string` - comma-joined `crop,fps,scale,setpts,reverse` (omitting empties; fps+scale always present).
  - `func videoGraph(c VideoConfig) string` - `[0:v]<chain>[v]`, or the boomerang split/reverse/concat form.

- [ ] **Step 1: Write failing tests** (append to `internal/gifjob/args_test.go`)

```go
func TestFilterChainAndGraph(t *testing.T) {
	// Plain: fps + scale only.
	c := VideoConfig{FPS: 15, Width: 480}
	if got := filterChain(c); got != "fps=15,scale=480:-2:flags=lanczos" {
		t.Errorf("plain chain = %q", got)
	}
	if got := videoGraph(c); got != "[0:v]fps=15,scale=480:-2:flags=lanczos[v]" {
		t.Errorf("plain graph = %q", got)
	}
	// Everything on: crop(1:1) + fps + scale + speed 2x + reverse, plus boomerang.
	c2 := VideoConfig{FPS: 12, Width: 200, Aspect: AspectSquare, Speed: 2.0, Reverse: true, Boomerang: true}
	wantChain := "crop='min(iw,ih)':'min(iw,ih)',fps=12,scale=200:-2:flags=lanczos,setpts=0.5000*PTS,reverse"
	if got := filterChain(c2); got != wantChain {
		t.Errorf("full chain =\n%q\nwant\n%q", got, wantChain)
	}
	wantGraph := "[0:v]" + wantChain + ",split[a][b];[b]reverse[r];[a][r]concat=n=2:v=1[v]"
	if got := videoGraph(c2); got != wantGraph {
		t.Errorf("boomerang graph =\n%q\nwant\n%q", got, wantGraph)
	}
	// Speed 1.0 and speed 0 both omit setpts.
	if setptsExpr(1.0) != "" || setptsExpr(0) != "" {
		t.Errorf("setpts for 1.0/0 should be empty")
	}
	if got := setptsExpr(0.5); got != "setpts=2.0000*PTS" {
		t.Errorf("setpts(0.5) = %q, want setpts=2.0000*PTS", got)
	}
	// Aspect crop expressions.
	if cropExpr(AspectFree) != "" {
		t.Error("free aspect must produce no crop")
	}
	if got := cropExpr(AspectWide); got != "crop='min(iw,ih*16/9)':'min(ih,iw*9/16)'" {
		t.Errorf("16:9 crop = %q", got)
	}
	if got := cropExpr(AspectTall); got != "crop='min(iw,ih*9/16)':'min(ih,iw*16/9)'" {
		t.Errorf("9:16 crop = %q", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/gifjob/ -run TestFilterChainAndGraph`
Expected: FAIL (undefined: filterChain, videoGraph, setptsExpr, cropExpr).

- [ ] **Step 3: Implement in `internal/gifjob/args.go`**

```go
// cropExpr returns the ffmpeg crop filter that center-crops the source to the
// aspect ratio, or "" for AspectFree. ffmpeg centers the crop when x/y are
// omitted.
func cropExpr(a Aspect) string {
	switch a {
	case AspectSquare:
		return "crop='min(iw,ih)':'min(iw,ih)'"
	case AspectWide: // 16:9
		return "crop='min(iw,ih*16/9)':'min(ih,iw*9/16)'"
	case AspectTall: // 9:16
		return "crop='min(iw,ih*9/16)':'min(ih,iw*16/9)'"
	default:
		return ""
	}
}

// setptsExpr returns the setpts filter that plays back at the given speed, or
// "" for normal speed (speed <= 0 or 1.0). A 2x speed halves PTS, so the factor
// is 1/speed.
func setptsExpr(speed float64) string {
	if speed <= 0 || speed == 1 {
		return ""
	}
	return fmt.Sprintf("setpts=%.4f*PTS", 1/speed)
}

// filterChain builds the ordered geometry/motion filters for a video source:
// crop, fps, scale, setpts (speed), reverse. Boomerang is NOT included here (it
// needs graph structure - see videoGraph). fps and scale are always present.
func filterChain(c VideoConfig) string {
	parts := make([]string, 0, 5)
	if s := cropExpr(c.Aspect); s != "" {
		parts = append(parts, s)
	}
	parts = append(parts, fmt.Sprintf("fps=%d", c.FPS))
	parts = append(parts, scaleChain(c.Width))
	if s := setptsExpr(c.Speed); s != "" {
		parts = append(parts, s)
	}
	if c.Reverse {
		parts = append(parts, "reverse")
	}
	return strings.Join(parts, ",")
}

// videoGraph builds the -filter_complex value that reads [0:v], applies the
// geometry/motion chain and optional boomerang (forward then reversed), and
// exposes the processed frames as [v].
func videoGraph(c VideoConfig) string {
	chain := filterChain(c)
	if c.Boomerang {
		return fmt.Sprintf("[0:v]%s,split[a][b];[b]reverse[r];[a][r]concat=n=2:v=1[v]", chain)
	}
	return fmt.Sprintf("[0:v]%s[v]", chain)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/gifjob/ -run TestFilterChainAndGraph && gofmt -l internal/gifjob/`
Expected: PASS, gofmt prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/args.go internal/gifjob/args_test.go
git commit -m "feat(engine): shared video filter graph with crop, speed, reverse, boomerang"
```

---

### Task 4: Format-aware VideoArgs (passes-slice)

Rewrite `VideoArgs` to return `passes [][]string` and branch per format. This replaces the old `(pass1, pass2)` signature - update the old video-args tests to the new form.

**Files:**
- Modify: `internal/gifjob/args.go`
- Modify: `internal/gifjob/args_test.go` (rewrite the video-args tests)

**Interfaces:**
- Consumes: `videoGraph`, `palettegenTail`, `paletteuseTail`, `loopArgs`, `secs` (existing), `Quality.WebPQuality`.
- Produces: `func VideoArgs(c VideoConfig, palettePath, outPath string) (passes [][]string)`. GIF -> 2 passes; WebP/APNG -> 1 pass. `palettePath` is used only for GIF.

- [ ] **Step 1: Rewrite the video-args tests** (replace `TestVideoArgsTwoPass`, `TestVideoArgsDitherOffAndLoopOnce`, `TestVideoArgsPositiveLoop` in `internal/gifjob/args_test.go`)

```go
func TestVideoArgsGIF(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 1000, EndMS: 3500, FPS: 15, Width: 480, Loop: LoopForever, Format: FormatGIF, Quality: Quality{MaxColors: 128, Dither: DitherSierra}}
	passes := VideoArgs(c, "pal.png", "out.gif")
	if len(passes) != 2 {
		t.Fatalf("GIF should be 2 passes, got %d", len(passes))
	}
	want1 := []string{
		"-y", "-ss", "1.000", "-t", "2.500", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=15,scale=480:-2:flags=lanczos[v];[v]palettegen=max_colors=128:stats_mode=diff[p]",
		"-map", "[p]", "pal.png",
	}
	if !reflect.DeepEqual(passes[0], want1) {
		t.Errorf("pass1 =\n%v\nwant\n%v", passes[0], want1)
	}
	want2 := []string{
		"-y", "-ss", "1.000", "-t", "2.500", "-i", "in.mp4", "-i", "pal.png",
		"-filter_complex", "[0:v]fps=15,scale=480:-2:flags=lanczos[v];[v][1:v]paletteuse=dither=sierra2_4a[o]",
		"-map", "[o]", "-loop", "0", "out.gif",
	}
	if !reflect.DeepEqual(passes[1], want2) {
		t.Errorf("pass2 =\n%v\nwant\n%v", passes[1], want2)
	}
}

func TestVideoArgsWebP(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 160, Loop: LoopForever, Format: FormatWebP, Quality: Quality{MaxColors: 256, Dither: DitherSierra, WebPQuality: 80}}
	passes := VideoArgs(c, "pal.png", "out.webp")
	if len(passes) != 1 {
		t.Fatalf("WebP should be 1 pass, got %d", len(passes))
	}
	want := []string{
		"-y", "-ss", "0.000", "-t", "1.000", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=10,scale=160:-2:flags=lanczos[v]", "-map", "[v]",
		"-c:v", "libwebp_anim", "-loop", "0", "-q:v", "80", "out.webp",
	}
	if !reflect.DeepEqual(passes[0], want) {
		t.Errorf("webp pass =\n%v\nwant\n%v", passes[0], want)
	}
}

func TestVideoArgsAPNGLoopOnce(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 160, Loop: LoopOnce, Format: FormatAPNG, Quality: DefaultQuality()}
	passes := VideoArgs(c, "pal.png", "out.png")
	if len(passes) != 1 {
		t.Fatalf("APNG should be 1 pass, got %d", len(passes))
	}
	want := []string{
		"-y", "-ss", "0.000", "-t", "1.000", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=10,scale=160:-2:flags=lanczos[v]", "-map", "[v]",
		"-f", "apng", "-plays", "1", "out.png", // once -> plays 1
	}
	if !reflect.DeepEqual(passes[0], want) {
		t.Errorf("apng pass =\n%v\nwant\n%v", passes[0], want)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/gifjob/ -run 'TestVideoArgs'`
Expected: FAIL (VideoArgs returns two values / wrong shape - compile error, then behavior).

- [ ] **Step 3: Implement in `internal/gifjob/args.go`** (replace the old `VideoArgs`, `palettegen`, `paletteuse` helpers - the tail helpers from Task 2 supersede `palettegen`/`paletteuse`)

```go
// VideoArgs builds the ffmpeg passes for a video job. GIF is two passes
// (palettegen then paletteuse); WebP and APNG are a single encode pass. -ss
// before -i is a fast seek; -t (duration) avoids the -ss/-to origin ambiguity.
// palettePath is used only by GIF. The config is assumed already validated.
func VideoArgs(c VideoConfig, palettePath, outPath string) (passes [][]string) {
	seek := []string{"-ss", secs(c.StartMS), "-t", secs(c.EndMS - c.StartMS)}
	graph := videoGraph(c)

	switch c.Format {
	case FormatWebP:
		flag, val := loopArgs(FormatWebP, c.Loop)
		p := append([]string{"-y"}, seek...)
		p = append(p, "-i", c.Input, "-filter_complex", graph, "-map", "[v]",
			"-c:v", "libwebp_anim", flag, val, "-q:v", strconv.Itoa(c.Quality.WebPQuality), outPath)
		return [][]string{p}
	case FormatAPNG:
		flag, val := loopArgs(FormatAPNG, c.Loop)
		p := append([]string{"-y"}, seek...)
		p = append(p, "-i", c.Input, "-filter_complex", graph, "-map", "[v]",
			"-f", "apng", flag, val, outPath)
		return [][]string{p}
	default: // GIF
		p1 := append([]string{"-y"}, seek...)
		p1 = append(p1, "-i", c.Input,
			"-filter_complex", graph+";[v]"+palettegenTail(c.Quality)+"[p]",
			"-map", "[p]", palettePath)
		flag, val := loopArgs(FormatGIF, c.Loop)
		p2 := append([]string{"-y"}, seek...)
		p2 = append(p2, "-i", c.Input, "-i", palettePath,
			"-filter_complex", graph+";[v][1:v]"+paletteuseTail(c.Quality)+"[o]",
			"-map", "[o]", flag, val, outPath)
		return [][]string{p1, p2}
	}
}
```

Delete the now-unused `palettegen` and `paletteuse` functions (their bodies are replaced by `palettegenTail`/`paletteuseTail`). Keep `scaleChain` (still used by `filterChain` and the images path). Keep `ditherArg` only if still referenced; if nothing references it after this task, delete it.

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/gifjob/ -run 'TestVideoArgs' && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/args.go internal/gifjob/args_test.go
git commit -m "feat(engine): format-aware VideoArgs returning a passes slice"
```

---

### Task 5: Images order/speed helpers + format-aware ImagesArgs

Images apply reverse/boomerang as frame *order* and speed as per-frame *duration* (frames are pre-normalized to a uniform canvas, so no video filter is needed). `ImagesArgs` also becomes a passes slice branching per format.

**Files:**
- Modify: `internal/gifjob/args.go`
- Modify: `internal/gifjob/args_test.go` (rewrite `TestImagesArgsAndConcatList`, add order/speed tests)

**Interfaces:**
- Consumes: `loopArgs`, `palettegenTail`, `paletteuseTail`, `scaleChain`.
- Produces:
  - `func frameOrder(frames []string, reverse, boomerang bool) []string`.
  - `func effectiveFrameMS(frameMS int, speed float64) int`.
  - `func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (passes [][]string)`.

- [ ] **Step 1: Rewrite/extend the images-args tests** (replace `TestImagesArgsAndConcatList` in `internal/gifjob/args_test.go`; keep `TestWriteConcatListEscapesQuotes` and `TestNormalizeArgs` unchanged)

```go
func TestImagesArgsGIF(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 400, Loop: 3, Format: FormatGIF, Quality: Quality{MaxColors: 256, Dither: DitherSierra}}
	passes := ImagesArgs(c, "list.txt", "pal.png", "out.gif")
	if len(passes) != 2 {
		t.Fatalf("GIF images should be 2 passes, got %d", len(passes))
	}
	want1 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt",
		"-vf", "scale=400:-2:flags=lanczos,palettegen=max_colors=256:stats_mode=diff", "pal.png",
	}
	if !reflect.DeepEqual(passes[0], want1) {
		t.Errorf("images pass1 =\n%v\nwant\n%v", passes[0], want1)
	}
	want2 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt", "-i", "pal.png",
		"-lavfi", "scale=400:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a", "-loop", "3", "out.gif",
	}
	if !reflect.DeepEqual(passes[1], want2) {
		t.Errorf("images pass2 =\n%v\nwant\n%v", passes[1], want2)
	}
}

func TestImagesArgsWebPAndAPNG(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png"}, FrameMS: 100, Width: 400, Loop: 3, Format: FormatWebP, Quality: Quality{WebPQuality: 70}}
	p := ImagesArgs(c, "list.txt", "pal.png", "out.webp")
	wantW := []string{"-y", "-f", "concat", "-safe", "0", "-i", "list.txt", "-c:v", "libwebp_anim", "-loop", "3", "-q:v", "70", "out.webp"}
	if len(p) != 1 || !reflect.DeepEqual(p[0], wantW) {
		t.Errorf("webp images =\n%v\nwant\n%v", p, wantW)
	}
	c.Format = FormatAPNG
	p = ImagesArgs(c, "list.txt", "pal.png", "out.png")
	wantA := []string{"-y", "-f", "concat", "-safe", "0", "-i", "list.txt", "-f", "apng", "-plays", "3", "out.png"}
	if len(p) != 1 || !reflect.DeepEqual(p[0], wantA) {
		t.Errorf("apng images =\n%v\nwant\n%v", p, wantA)
	}
}

func TestFrameOrder(t *testing.T) {
	in := []string{"a", "b", "c"}
	if got := frameOrder(in, false, false); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("plain = %v", got)
	}
	if got := frameOrder(in, true, false); !reflect.DeepEqual(got, []string{"c", "b", "a"}) {
		t.Errorf("reverse = %v", got)
	}
	if got := frameOrder(in, false, true); !reflect.DeepEqual(got, []string{"a", "b", "c", "b", "a"}) {
		t.Errorf("boomerang = %v", got)
	}
	if got := frameOrder(in, true, true); !reflect.DeepEqual(got, []string{"c", "b", "a", "b", "c"}) {
		t.Errorf("reverse+boomerang = %v", got)
	}
	// frameOrder must not mutate its input.
	if !reflect.DeepEqual(in, []string{"a", "b", "c"}) {
		t.Errorf("input was mutated: %v", in)
	}
}

func TestEffectiveFrameMS(t *testing.T) {
	cases := []struct {
		ms    int
		speed float64
		want  int
	}{
		{100, 1.0, 100},
		{100, 2.0, 50},
		{100, 0.5, 200},
		{100, 0, 100},   // speed 0 -> normal
		{100, -1, 100},  // negative -> normal
		{3, 2.0, 2},     // rounds; never below 1
		{1, 4.0, 1},     // floor at 1
	}
	for _, c := range cases {
		if got := effectiveFrameMS(c.ms, c.speed); got != c.want {
			t.Errorf("effectiveFrameMS(%d,%v) = %d, want %d", c.ms, c.speed, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/gifjob/ -run 'TestImagesArgs|TestFrameOrder|TestEffectiveFrameMS'`
Expected: FAIL (undefined: frameOrder, effectiveFrameMS; ImagesArgs shape).

- [ ] **Step 3: Implement in `internal/gifjob/args.go`** (replace the old `ImagesArgs`)

```go
// frameOrder returns the play order of the normalized frames after applying
// reverse (flip the sequence) then boomerang (append the sequence back minus
// its last frame, so the turn-around frame is not duplicated). It never mutates
// its input.
func frameOrder(frames []string, reverse, boomerang bool) []string {
	out := make([]string, len(frames))
	copy(out, frames)
	if reverse {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	if boomerang && len(out) > 1 {
		for i := len(out) - 2; i >= 0; i-- {
			out = append(out, out[i])
		}
	}
	return out
}

// effectiveFrameMS scales the per-frame duration by playback speed (2x speed
// halves each frame's on-screen time). speed <= 0 means normal. The result is
// never below 1 ms.
func effectiveFrameMS(frameMS int, speed float64) int {
	if speed <= 0 {
		return frameMS
	}
	ms := int(math.Round(float64(frameMS) / speed))
	if ms < 1 {
		ms = 1
	}
	return ms
}

// ImagesArgs builds the ffmpeg passes for an images job reading the ordered,
// pre-normalized frames through the concat demuxer. GIF is palettegen then
// paletteuse; WebP and APNG are a single encode pass. Frame durations come from
// listPath, so no fps filter is applied. palettePath is used only by GIF.
func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (passes [][]string) {
	in := []string{"-y", "-f", "concat", "-safe", "0", "-i", listPath}
	clone := func() []string { return append([]string(nil), in...) }

	switch c.Format {
	case FormatWebP:
		flag, val := loopArgs(FormatWebP, c.Loop)
		p := append(clone(), "-c:v", "libwebp_anim", flag, val, "-q:v", strconv.Itoa(c.Quality.WebPQuality), outPath)
		return [][]string{p}
	case FormatAPNG:
		flag, val := loopArgs(FormatAPNG, c.Loop)
		p := append(clone(), "-f", "apng", flag, val, outPath)
		return [][]string{p}
	default: // GIF
		p1 := append(clone(), "-vf", scaleChain(c.Width)+","+palettegenTail(c.Quality), palettePath)
		flag, val := loopArgs(FormatGIF, c.Loop)
		p2 := append(clone(), "-i", palettePath,
			"-lavfi", scaleChain(c.Width)+"[x];[x][1:v]"+paletteuseTail(c.Quality),
			flag, val, outPath)
		return [][]string{p2Head(p2)...} // placeholder - see note
	}
}
```

Note: the GIF `p2` must interleave `-i palettePath` right after the input list. Write it plainly instead of the placeholder above:

```go
	default: // GIF
		p1 := append(clone(), "-vf", scaleChain(c.Width)+","+palettegenTail(c.Quality), palettePath)
		flag, val := loopArgs(FormatGIF, c.Loop)
		p2 := clone()
		p2 = append(p2, "-i", palettePath,
			"-lavfi", scaleChain(c.Width)+"[x];[x][1:v]"+paletteuseTail(c.Quality),
			flag, val, outPath)
		return [][]string{p1, p2}
	}
```

(Delete the placeholder `p2Head` line - it is not real; the plain version is the implementation.)

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/gifjob/ -run 'TestImagesArgs|TestFrameOrder|TestEffectiveFrameMS' && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/args.go internal/gifjob/args_test.go
git commit -m "feat(engine): images frame-order/speed helpers and format-aware ImagesArgs"
```

---

### Task 6: run.go - passes-slice runner, statResult fallback, order/speed wiring

Wire the new arg builders into the runners: loop over passes (last = encode), create a palette temp only for GIF, apply frame order + speed for images, normalize Speed/Format defaults, and give `statResult` a width/height fallback (stdlib cannot decode WebP).

**Files:**
- Modify: `internal/gifjob/run.go`
- Modify: `internal/gifjob/run_test.go` (add a WebP single-pass test; existing GIF tests should still pass unchanged)

**Interfaces:**
- Consumes: `VideoArgs`, `ImagesArgs` (now `[][]string`), `frameOrder`, `effectiveFrameMS`, `OutputHeight`, `Format`.
- Produces: `RunVideo`/`RunImages` unchanged signatures. Internal `statResult(path string, wantW, wantH int) (Result, error)`.

- [ ] **Step 1: Add a WebP single-pass test** (append to `internal/gifjob/run_test.go`)

```go
func TestRunVideoWebPIsSinglePass(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.webp")
	r := &fakeRunner{out: out, writeOn: 1} // single pass writes the output
	tools := ffmpeg.Paths{FFmpeg: "ffmpeg", FFprobe: "ffprobe"}
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 2000, FPS: 12, Width: 320, Loop: LoopForever, Format: FormatWebP, SrcWidth: 640, SrcHeight: 360, Quality: DefaultQuality()}
	before := giflyTempFiles(t)
	res, err := RunVideo(context.Background(), tools, r, c, out, nil)
	if err != nil {
		t.Fatalf("RunVideo webp = %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("WebP must be a single ffmpeg pass, got %d", len(r.calls))
	}
	// WebP cannot be decoded by the stdlib, so dimensions fall back to the
	// config: width 320, height OutputHeight(640,360,free,320)=180.
	if res.Width != 320 || res.Height != 180 {
		t.Errorf("webp result dims = %dx%d, want 320x180 (fallback)", res.Width, res.Height)
	}
	assertNoNewGiflyTempFiles(t, before) // no palette temp for webp
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/gifjob/ -run TestRunVideoWebPIsSinglePass`
Expected: FAIL (currently 2 passes / palette temp created / dims wrong).

- [ ] **Step 3: Implement in `internal/gifjob/run.go`**

Add `_ "image/png"` to the imports (so APNG first-frame dims decode). Replace `statResult`, `RunVideo`, `RunImages`:

```go
// statResult stats a finished animation and reads its dimensions back with the
// standard library. GIF and APNG decode; WebP cannot be decoded by the stdlib,
// so when decoding fails or yields nothing the caller-supplied wantW/wantH
// (computed from the config) are used instead.
func statResult(path string, wantW, wantH int) (Result, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("output not produced: %w", err)
	}
	res := Result{Path: path, Bytes: fi.Size(), Width: wantW, Height: wantH}
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		if cfg, _, err := decodeConfig(f); err == nil && cfg.Width > 0 {
			res.Width, res.Height = cfg.Width, cfg.Height
		}
	}
	return res, nil
}

// runPasses runs each pass in order. Only the last pass (the encode) receives
// onProgress and, on failure, has its partial output removed.
func runPasses(ctx context.Context, bin string, r Runner, passes [][]string, outPath string, onProgress func(ffmpeg.Progress)) error {
	last := len(passes) - 1
	for i, args := range passes {
		var cb func(ffmpeg.Progress)
		if i == last {
			cb = onProgress
		}
		if err := r.Run(ctx, bin, args, cb); err != nil {
			if i == last {
				os.Remove(outPath)
				return fmt.Errorf("encode pass: %w", err)
			}
			return fmt.Errorf("pass %d: %w", i+1, err)
		}
	}
	return nil
}

// normalizeVideo applies engine defaults that keep zero-value configs working:
// an empty Format is GIF and a non-positive Speed is normal (1.0), clamped to 4.
func normalizeVideo(c VideoConfig) VideoConfig {
	if c.Format == "" {
		c.Format = FormatGIF
	}
	if c.Speed <= 0 {
		c.Speed = 1
	}
	if c.Speed > 4 {
		c.Speed = 4
	}
	if c.Quality.WebPQuality <= 0 {
		c.Quality.WebPQuality = 75
	}
	return c
}

// RunVideo validates the config, runs the format's passes (GIF also writes a
// temp palette), removes a partial output on encode failure, and stats the
// result.
func RunVideo(ctx context.Context, tools ffmpeg.Paths, r Runner, c VideoConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error) {
	c = normalizeVideo(c)
	if err := c.Validate(); err != nil {
		return Result{}, err
	}
	var palettePath string
	if c.Format == FormatGIF {
		palette, err := os.CreateTemp("", "gifly-*.png")
		if err != nil {
			return Result{}, err
		}
		palettePath = palette.Name()
		palette.Close()
		defer os.Remove(palettePath)
	}
	if err := runPasses(ctx, tools.FFmpeg, r, VideoArgs(c, palettePath, outPath), outPath, onProgress); err != nil {
		return Result{}, err
	}
	return statResult(outPath, c.Width, OutputHeight(c.SrcWidth, c.SrcHeight, c.Aspect, c.Width))
}

// RunImages normalizes every input onto the shared canvas, applies the play
// order (reverse/boomerang) and speed (per-frame duration), then runs the
// format's passes. All temp files live in one directory removed on every exit.
func RunImages(ctx context.Context, tools ffmpeg.Paths, r Runner, c ImagesConfig, outPath string, onProgress func(ffmpeg.Progress)) (Result, error) {
	if c.Format == "" {
		c.Format = FormatGIF
	}
	if c.Quality.WebPQuality <= 0 {
		c.Quality.WebPQuality = 75
	}
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

	ordered := frameOrder(norm, c.Reverse, c.Boomerang)
	listPath := filepath.Join(tmp, "list.txt")
	list, err := os.Create(listPath)
	if err != nil {
		return Result{}, err
	}
	if err := WriteConcatList(list, ordered, effectiveFrameMS(c.FrameMS, c.Speed)); err != nil {
		list.Close()
		return Result{}, err
	}
	list.Close()

	var palettePath string
	if c.Format == FormatGIF {
		palettePath = filepath.Join(tmp, "palette.png")
	}
	if err := runPasses(ctx, tools.FFmpeg, r, ImagesArgs(c, listPath, palettePath, outPath), outPath, onProgress); err != nil {
		return Result{}, err
	}
	return statResult(outPath, c.Width, c.Height)
}
```

- [ ] **Step 4: Run all gifjob unit tests + build**

Run: `go test ./internal/gifjob/ && go build ./... && gofmt -l internal/gifjob/`
Expected: PASS (the existing GIF `TestRunVideoRunsBothPasses`, `TestRunImagesRunsBothPasses`, partial-output tests still pass - GIF is still 2 passes with a palette temp), build OK, gofmt clean.

- [ ] **Step 5: Commit**

```bash
git add internal/gifjob/run.go internal/gifjob/run_test.go
git commit -m "feat(engine): passes-slice runner with per-format temp handling and dim fallback"
```

---

### Task 7: giflycli flags for every new option

Expose format, aspect, speed, reverse, boomerang, dither, and webp quality on the CLI, default the output extension from the format, and fill `SrcWidth/SrcHeight` from the probe so WebP/APNG report correct dimensions. The CLI is the end-to-end driver the gated tests (Task 8) exercise.

**Files:**
- Modify: `cmd/giflycli/main.go`
- Modify: `cmd/giflycli/main_test.go` (extend flag-parsing/helper tests)

**Interfaces:**
- Consumes: all gifjob types from Tasks 1-6.
- Produces: CLI flags `-format gif|webp|apng`, `-aspect free|1:1|16:9|9:16`, `-speed 1.0`, `-reverse`, `-boomerang`, `-dither sierra2|none|bayer|floyd`, `-q 75`. New helpers `parseFormat`, `parseAspect`, `parseDither`, `defaultOut(base, f Format) string`.

- [ ] **Step 1: Read the current CLI + its tests**

Run: (Read `cmd/giflycli/main.go` and `cmd/giflycli/main_test.go` in full before editing, so the new flags follow the existing `flag.FlagSet` pattern and the test style.)

- [ ] **Step 2: Write failing helper tests** (append to `cmd/giflycli/main_test.go`)

```go
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
```

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./cmd/giflycli/ -run 'TestParseFormatAspectDither|TestDefaultOut'`
Expected: FAIL (undefined helpers).

- [ ] **Step 4: Implement in `cmd/giflycli/main.go`**

Add the parse helpers and `defaultOut`:

```go
func parseFormat(s string) (gifjob.Format, error) {
	f := gifjob.Format(s)
	if !f.Valid() {
		return "", fmt.Errorf("format must be gif, webp or apng, got %q", s)
	}
	return f, nil
}

func parseAspect(s string) (gifjob.Aspect, error) {
	if s == "free" || s == "" {
		return gifjob.AspectFree, nil
	}
	a := gifjob.Aspect(s)
	if !a.Valid() {
		return "", fmt.Errorf("aspect must be free, 1:1, 16:9 or 9:16, got %q", s)
	}
	return a, nil
}

func parseDither(s string) (gifjob.DitherMethod, error) {
	switch gifjob.DitherMethod(s) {
	case gifjob.DitherNone, gifjob.DitherBayer, gifjob.DitherSierra, gifjob.DitherFloyd:
		return gifjob.DitherMethod(s), nil
	}
	return "", fmt.Errorf("dither must be none, bayer, sierra2 or floyd, got %q", s)
}

// defaultOut swaps the extension of the default "out.gif" to match the format.
// A caller who set any other output name keeps it verbatim.
func defaultOut(out string, f gifjob.Format) string {
	if out == "out.gif" {
		return "out" + f.Ext()
	}
	return out
}
```

In both the `video` and `images` subcommands, add the flags (alongside the existing ones):

```go
	format := fs.String("format", "gif", "output format: gif | webp | apng")
	aspect := fs.String("aspect", "free", "crop aspect: free | 1:1 | 16:9 | 9:16")
	speed := fs.Float64("speed", 1.0, "playback speed 0.25..4")
	reverse := fs.Bool("reverse", false, "reverse playback")
	boomerang := fs.Bool("boomerang", false, "play forward then backward")
	dither := fs.String("dither", "sierra2", "dither: none | bayer | sierra2 | floyd")
	webpQ := fs.Int("q", 75, "webp quality 0..100")
```

After parsing, resolve them (`parseFormat`, `parseAspect`, `parseDither` - return their errors) and:
- `*out = defaultOut(*out, fmtVal)` before running.
- Build `Quality{MaxColors: *colors, Dither: ditherVal, WebPQuality: *webpQ}` (drop the `-nodither` bool in favor of `-dither`, or keep `-nodither` mapping to `none` when set - keep `-nodither` for back-compat: if `*noDither` then `ditherVal = gifjob.DitherNone`).
- For `video`: set `Aspect: aspectVal, Speed: *speed, Reverse: *reverse, Boomerang: *boomerang, Format: fmtVal, SrcWidth: m.Width, SrcHeight: m.Height` on the `VideoConfig`.
- For `images`: set `Speed: *speed, Reverse: *reverse, Boomerang: *boomerang, Format: fmtVal` on the `ImagesConfig`, and compute `height := imagesHeight(*h, *w, first.Width, first.Height)` using `gifjob.OutputHeight(first.Width, first.Height, aspectVal, *w)` when `*h == 0` (extend `imagesHeight` to take an aspect, or compute inline). For images, apply the aspect to the canvas height.

Update the usage comment at the top of `main.go` to list the new flags.

- [ ] **Step 5: Run tests + build**

Run: `go test ./cmd/giflycli/ && go build ./... && gofmt -l cmd/ && go vet ./...`
Expected: PASS, build OK, gofmt clean, vet clean.

- [ ] **Step 6: Commit**

```bash
git add cmd/giflycli/main.go cmd/giflycli/main_test.go
git commit -m "feat(cli): expose format, aspect, speed, reverse, boomerang, dither flags"
```

---

### Task 8: Gated real-ffmpeg tests for every format and option

Prove the argv actually produce valid files - the one thing the argv-equality unit tests cannot. These are `//go:build ffmpeg` and are run by the CONTROLLER with real ffmpeg (`GIFLY_FFMPEG_DIR` set). The implementer writes them but cannot run them.

**Files:**
- Modify: `internal/gifjob/integration_test.go`

**Interfaces:**
- Consumes: `RunVideo`, `RunImages`, all config options.

- [ ] **Step 1: Add gated tests** (append to `internal/gifjob/integration_test.go`, which already has `//go:build ffmpeg`)

```go
// makeTestVideo writes a short lavfi testsrc clip and returns its path.
func makeTestVideo(t *testing.T, tools ffmpeg.Paths, dir string) string {
	t.Helper()
	in := filepath.Join(dir, "in.mp4")
	if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
		[]string{"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15", "-c:v", "mpeg4", "-q:v", "3", in}, nil); err != nil {
		t.Fatalf("making test video: %v", err)
	}
	return in
}

func TestVideoWebPAndAPNGAreReal(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	for _, f := range []struct {
		format gifjob.Format
		out    string
	}{
		{gifjob.FormatWebP, "out.webp"},
		{gifjob.FormatAPNG, "out.png"},
	} {
		c := VideoConfig{Input: in, StartMS: 0, EndMS: 1500, FPS: 10, Width: 160, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: f.format, Quality: DefaultQuality()}
		res, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, f.out), nil)
		if err != nil {
			t.Fatalf("%s encode: %v", f.format, err)
		}
		if res.Bytes < 100 {
			t.Errorf("%s output suspiciously small: %d bytes", f.format, res.Bytes)
		}
		if res.Width != 160 {
			t.Errorf("%s width = %d, want 160", f.format, res.Width)
		}
	}
}

func TestVideoBoomerangDoublesFrames(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	base := VideoConfig{Input: in, StartMS: 0, EndMS: 1000, FPS: 10, Width: 120, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: FormatGIF, Quality: DefaultQuality()}
	plain := base
	if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), plain, filepath.Join(dir, "p.gif"), nil); err != nil {
		t.Fatalf("plain: %v", err)
	}
	boom := base
	boom.Boomerang = true
	if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), boom, filepath.Join(dir, "b.gif"), nil); err != nil {
		t.Fatalf("boomerang: %v", err)
	}
	np := frameCount(t, filepath.Join(dir, "p.gif"))
	nb := frameCount(t, filepath.Join(dir, "b.gif"))
	if nb <= np {
		t.Errorf("boomerang produced %d frames, want more than the plain %d", nb, np)
	}
}

func TestVideoAspectSquareIsSquare(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	c := VideoConfig{Input: in, StartMS: 0, EndMS: 1000, FPS: 10, Width: 200, SrcWidth: 320, SrcHeight: 240, Aspect: AspectSquare, Loop: LoopForever, Format: FormatGIF, Quality: DefaultQuality()}
	res, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, "sq.gif"), nil)
	if err != nil {
		t.Fatalf("aspect square: %v", err)
	}
	if res.Width != res.Height {
		t.Errorf("1:1 output is %dx%d, want a square", res.Width, res.Height)
	}
}

func TestVideoDitherMethodsAllEncode(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := makeTestVideo(t, tools, dir)
	for _, d := range []gifjob.DitherMethod{gifjob.DitherNone, gifjob.DitherBayer, gifjob.DitherSierra, gifjob.DitherFloyd} {
		c := VideoConfig{Input: in, StartMS: 0, EndMS: 800, FPS: 8, Width: 120, SrcWidth: 320, SrcHeight: 240, Loop: LoopForever, Format: FormatGIF, Quality: Quality{MaxColors: 128, Dither: d}}
		out := filepath.Join(dir, "d_"+string(d)+".gif")
		if _, err := RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, out, nil); err != nil {
			t.Fatalf("dither %s: %v", d, err)
		}
	}
}

func TestImagesReverseAndSpeedEncode(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	var imgs []string
	for i, col := range []string{"red", "green", "blue"} {
		p := filepath.Join(dir, col+".png")
		if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
			[]string{"-y", "-f", "lavfi", "-i", "color=c=" + col + ":s=64x48:d=1", "-frames:v", "1", p}, nil); err != nil {
			t.Fatalf("image %d: %v", i, err)
		}
		imgs = append(imgs, p)
	}
	c := ImagesConfig{Inputs: imgs, FrameMS: 200, Width: 64, Height: 48, Reverse: true, Boomerang: true, Speed: 2.0, Loop: LoopForever, Format: FormatWebP, Quality: DefaultQuality()}
	res, err := RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, filepath.Join(dir, "out.webp"), nil)
	if err != nil {
		t.Fatalf("images webp with reverse+boomerang+speed: %v", err)
	}
	if res.Bytes < 100 {
		t.Errorf("webp output suspiciously small: %d bytes", res.Bytes)
	}
}

// frameCount decodes a GIF and returns its frame count.
func frameCount(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("%s is not a valid GIF: %v", path, err)
	}
	return len(g.Image)
}
```

Ensure the file imports `gif` (`image/gif`), `os`, `path/filepath`, `context`, `testing`, and the gifjob-external `gifjob` names are referenced as bare identifiers (this test file is IN package `gifjob`, so use `FormatWebP` not `gifjob.FormatWebP` - adjust the snippet's `gifjob.` prefixes to bare names to match the package, except where the snippet already uses bare names). Keep it consistent with the existing `integration_test.go` which is `package gifjob`.

- [ ] **Step 2: Build the gated tests (compile only - implementer has no ffmpeg)**

Run: `go vet -tags ffmpeg ./internal/gifjob/ && gofmt -l internal/gifjob/`
Expected: compiles (vet clean), gofmt clean. The implementer reports DONE_WITH_CONCERNS noting the tests cannot be *run* without ffmpeg - the controller runs them.

- [ ] **Step 3: Commit**

```bash
git add internal/gifjob/integration_test.go
git commit -m "test(engine): gated real-ffmpeg tests for webp, apng, boomerang, aspect, dither, images options"
```

- [ ] **Step 4: CONTROLLER runs the gated suite with real ffmpeg**

The controller (not the implementer) runs:
`GIFLY_FFMPEG_DIR=C:\Users\hoijun\ffmpeg-dev\ffmpeg-master-latest-win64-lgpl\bin go test -tags ffmpeg ./internal/gifjob/`
Expected: PASS. If any encode fails, that is a real argv bug to fix before the task is complete.

---

## Self-Review

**1. Spec coverage** (decided scope -> task):
- Output formats GIF + WebP + APNG -> Tasks 1 (Format type), 4 (VideoArgs), 5 (ImagesArgs), 6 (runner), 8 (gated proof). Covered.
- Crop/aspect 1:1/16:9/9:16/free -> Tasks 1 (Aspect, OutputHeight), 3 (cropExpr, filterChain), 7 (CLI), 8 (square test). Covered.
- Boomerang + reverse -> Tasks 3 (videoGraph), 5 (frameOrder), 6 (wiring), 8 (frame-count test). Covered.
- Playback speed 0.5..3x -> Tasks 3 (setptsExpr), 5 (effectiveFrameMS), 6 (normalize/clamp), 7 (CLI). Covered.
- Dither methods none/bayer/sierra2/floyd -> Tasks 1 (DitherMethod.ffmpeg), 2 (paletteuseTail), 7 (CLI), 8 (all-methods test). Covered.
- CLI-driven, real-ffmpeg-tested before GUI -> Tasks 7 + 8. Covered.
- Size-target fit loop still works across formats -> unchanged (width-based, in app/convert.go); GIF path unchanged, webp/apng shrink the same way. No task needed; note only.

**2. Placeholder scan:** Task 5 Step 3 intentionally shows a `p2Head(...)` placeholder and immediately replaces it with the real plain implementation, with an explicit "delete the placeholder" instruction. No `TODO`/`TBD`/"handle edge cases" placeholders remain. All test code is concrete.

**3. Type consistency:** `Format`/`Aspect`/`DitherMethod` methods (`Valid`, `Ext`, `ffmpeg`), `Quality{MaxColors, Dither, WebPQuality}`, `OutputHeight(srcW, srcH, aspect, outW)`, `VideoArgs`/`ImagesArgs` returning `[][]string`, `statResult(path, wantW, wantH)`, `loopArgs(f, loop) (flag, val)`, `frameOrder(frames, reverse, boomerang)`, `effectiveFrameMS(frameMS, speed)` are used identically across Tasks 1-8. The GIF loop uses the `LoopMode` literal; WebP/APNG go through `webpApngLoop`. Consistent.

**Note carried to Plan 5 (GUI):** WebP output height is computed (`OutputHeight`) not decoded, so for the `16:9` preset at some widths the displayed height can differ by 2px from the actual file (GIF/APNG decode exact). Cosmetic. Plan 5 threads real source dims from the frontend into `SrcWidth/SrcHeight`; until then the app path leaves them 0 (GIF unaffected; only WebP/APNG height display).
