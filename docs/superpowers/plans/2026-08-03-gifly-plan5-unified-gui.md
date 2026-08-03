# gifly Plan 5 - Unified GUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two-mode (Video / Images) GUI with ONE unified screen that takes a video OR a set of images and exposes every engine v2 option - format (GIF/WebP/APNG), crop/aspect, boomerang, reverse, speed, dither method, colors, WebP quality, platform presets, target size, and a live size estimate.

**Architecture:** The engine (internal/gifjob) already does everything; this plan (1) widens the app binding layer (internal/app: VideoRequest/ImagesRequest gain the new fields, mapping fills the gifjob config, preview serves the right content-type per format) and (2) rebuilds the frontend as a single `Workspace` orchestrator over a shared `settings` store and a `source` store, with the pure request-building / estimate / validation logic in `lib/` (vitest-tested) and small focused Svelte components for each option group. Function and structure first; visual polish is deliberately deferred.

**Tech Stack:** Go 1.26 (cgo-free), Wails v2, Svelte 4 + TypeScript + Vite, vitest.

## Global Constraints

- Module `github.com/hoijun-kim/gifly`. Only non-stdlib Go dependency allowed: `github.com/wailsapp/wails/v2`. No `golang.org/x/image` or other third-party Go imports. Frontend deps stay as they are (no new npm packages).
- Gates that MUST pass after every task: `CGO_ENABLED=0 go build ./...`, untagged `go test ./...`, `gofmt -l .` (empty), `go vet ./...`, and in `frontend/`: `npm run build` (Vite build incl. svelte compile) and `npx vitest run` (unit tests). If a task touches no Go, still run the frontend gates; if it touches no frontend, still run the Go gates.
- ASCII only in code/comments: plain hyphen `-`, never a unicode dash. Never two adjacent single quotes (`''`) in a Go doc comment (gofmt rewrites it).
- No AI/Claude co-author trailer on any commit. Plain conventional-commit subjects.
- `frontend/dist/` is git-ignored except a committed `frontend/dist/.gitkeep` marker (main.go has `//go:embed all:frontend/dist`). A Vite build DELETES `.gitkeep` in the working tree - after any `npm run build`, restore it with `git checkout -- frontend/dist/.gitkeep` before committing so it stays tracked.
- Real-ffmpeg gated tests (`//go:build ffmpeg`) are run by the CONTROLLER (implementers have no ffmpeg): `GIFLY_FFMPEG_DIR=C:\Users\hoijun\ffmpeg-dev\ffmpeg-master-latest-win64-lgpl\bin`.
- Engine v2 API already on master (do NOT reimplement): `gifjob.Format` (`FormatGIF`/`FormatWebP`/`FormatAPNG`), `gifjob.Aspect` (`AspectFree`=""/`AspectSquare`="1:1"/`AspectWide`="16:9"/`AspectTall`="9:16"), `gifjob.DitherMethod` (`DitherNone`/`DitherBayer`/`DitherSierra`/`DitherFloyd`), `gifjob.OutputHeight(srcW, srcH, aspect, outW)`, `gifjob.Quality{MaxColors, Dither DitherMethod, WebPQuality}`, `gifjob.VideoConfig{Input,StartMS,EndMS,FPS,Width,SrcWidth,SrcHeight,Aspect,Speed,Reverse,Boomerang,Loop,Format,Quality}`, `gifjob.ImagesConfig{Inputs,FrameMS,Width,Height,Speed,Reverse,Boomerang,Loop,Format,Quality}`. Speed valid range 0.25..4 (Validate rejects <0 or >4; 0 is normalized to 1 by the runner).

## File Structure

**Backend (internal/app):**
- `convert.go` - MODIFY: widen `VideoRequest`/`ImagesRequest` (add `SrcWidth,SrcHeight,Aspect,Speed,Reverse,Boomerang,Format,WebPQuality`; change `Dither bool` -> `Dither string`); add `parseFormatReq`/`parseAspectReq`/`parseDitherReq`; map all fields into the gifjob configs; images height + fit-loop height use `OutputHeight` with the aspect. Remove the obsolete `ditherFromBool`.
- `preview.go` - MODIFY: `PreviewHandler` sets `Content-Type` by the preview file's extension (.gif->image/gif, .webp->image/webp, .png->image/apng).
- `convert_test.go`, `binding_test.go`, `e2e_test.go` - MODIFY: new request shape (`Dither` is now a string); add mapping assertions; add gated WebP/aspect e2e.
- `preview_test.go` - CREATE (or extend if present): content-type per extension.
- `frontend/wailsjs/go/models.ts` - MODIFY (regenerate): the TS DTOs must mirror the new Go request fields.

**Frontend (frontend/src):**
- `lib/settings.ts` - CREATE: `Settings` type, `defaultSettings()`, `settings` writable store; the format/aspect/dither/loop TS unions.
- `lib/source.ts` - CREATE: `Source` discriminated union + `source` writable store.
- `lib/format.ts` - MODIFY: add `outputHeight`, `formatExt`, `formatLabel`; replace `estimateBytes` with a format-aware version.
- `lib/validate.ts` - MODIFY: add `settingsValid`; keep `videoValid`/`imagesValid` for timing.
- `lib/request.ts` - CREATE: `PLATFORM_PRESETS`, `deriveOut`, `buildVideoRequest`, `buildImagesRequest`.
- `components/Workspace.svelte` - CREATE: the single-screen orchestrator (owns convert/progress/result; renders the sections).
- `components/SourcePanel.svelte` - CREATE: empty-state pickers + source summary (video meta | images reorderable list).
- `components/VideoTiming.svelte`, `components/ImagesTiming.svelte` - CREATE: source-specific timing.
- `components/FormatPicker.svelte`, `components/SizeControls.svelte`, `components/PlaybackControls.svelte`, `components/QualityControls.svelte`, `components/OutputControls.svelte` - CREATE: shared option groups bound to `settings`.
- `App.svelte` - MODIFY: render `Workspace` (drop `ModeSwitch`).
- `components/ModeSwitch.svelte`, `modes/Video.svelte`, `modes/Images.svelte` - DELETE (Task 8). The `mode` store in `lib/wails.ts` - DELETE (Task 8).
- `lib/format.test.ts`, `lib/validate.test.ts`, `lib/request.test.ts`, `lib/settings.test.ts` - unit tests (vitest).

---

### Task 1: Backend - expose all engine options through the binding layer

**Files:**
- Modify: `internal/app/convert.go`
- Modify: `internal/app/convert_test.go`, `internal/app/binding_test.go`, `internal/app/e2e_test.go`
- Modify: `frontend/wailsjs/go/models.ts` (regenerate)

**Interfaces:**
- Produces (Go DTOs the frontend fills):
  - `VideoRequest{ Input string; StartMS,EndMS int64; FPS,Width,SrcWidth,SrcHeight int; Aspect string; Speed float64; Reverse,Boomerang bool; Loop string; Colors int; Dither string; WebPQuality int; Format string; Out string; TargetKB int }`
  - `ImagesRequest{ Inputs []string; FrameMS,Width int; Aspect string; Speed float64; Reverse,Boomerang bool; Loop string; Colors int; Dither string; WebPQuality int; Format string; Out string; TargetKB int }`
  - `ConvertResult` unchanged `{Path string; Bytes int64; Width,Height int}`.
  - Helpers `parseFormatReq(string) gifjob.Format`, `parseAspectReq(string) gifjob.Aspect`, `parseDitherReq(string) gifjob.DitherMethod`.

- [ ] **Step 1: Read the current binding tests** - Read `internal/app/convert_test.go` and `internal/app/binding_test.go` in full to see the request-construction and mapping style before editing (they set `Dither: true`, which becomes `Dither: "sierra2"`).

- [ ] **Step 2: Write the failing mapping test** - append to `internal/app/convert_test.go`:

```go
func TestVideoConfigMapsAllOptions(t *testing.T) {
	req := VideoRequest{
		Input: "in.mp4", StartMS: 100, EndMS: 2000, FPS: 20, Width: 320,
		SrcWidth: 1280, SrcHeight: 720, Aspect: "1:1", Speed: 2, Reverse: true, Boomerang: true,
		Loop: "once", Colors: 128, Dither: "bayer", WebPQuality: 80, Format: "webp",
	}
	c := videoConfig(req)
	if c.Format != gifjob.FormatWebP {
		t.Errorf("Format = %q, want webp", c.Format)
	}
	if c.Aspect != gifjob.AspectSquare {
		t.Errorf("Aspect = %q, want 1:1", c.Aspect)
	}
	if c.Speed != 2 || !c.Reverse || !c.Boomerang {
		t.Errorf("speed/reverse/boomerang = %v/%v/%v", c.Speed, c.Reverse, c.Boomerang)
	}
	if c.SrcWidth != 1280 || c.SrcHeight != 720 {
		t.Errorf("src dims = %dx%d, want 1280x720", c.SrcWidth, c.SrcHeight)
	}
	if c.Quality.Dither != gifjob.DitherBayer || c.Quality.MaxColors != 128 || c.Quality.WebPQuality != 80 {
		t.Errorf("quality = %+v", c.Quality)
	}
	if c.Loop != gifjob.LoopOnce {
		t.Errorf("loop = %d, want once(-1)", int(c.Loop))
	}
}

func TestParseReqDefaults(t *testing.T) {
	if parseFormatReq("") != gifjob.FormatGIF || parseFormatReq("bogus") != gifjob.FormatGIF {
		t.Error("empty/unknown format should default to gif")
	}
	if parseAspectReq("") != gifjob.AspectFree || parseAspectReq("bogus") != gifjob.AspectFree {
		t.Error("empty/unknown aspect should default to free")
	}
	if parseDitherReq("") != gifjob.DitherSierra || parseDitherReq("bogus") != gifjob.DitherSierra {
		t.Error("empty/unknown dither should default to sierra2")
	}
	if parseFormatReq("apng") != gifjob.FormatAPNG || parseAspectReq("9:16") != gifjob.AspectTall || parseDitherReq("floyd") != gifjob.DitherFloyd {
		t.Error("known values must round-trip")
	}
}
```

- [ ] **Step 3: Run to verify it fails** - `go test ./internal/app/ -run 'TestVideoConfigMapsAllOptions|TestParseReqDefaults'` -> FAIL (fields/helpers undefined; `Dither` still bool).

- [ ] **Step 4: Implement in `internal/app/convert.go`**

Replace the `Dither bool` field in both request structs and add the new fields (final shapes are in the Interfaces block above). Delete `ditherFromBool`. Add:

```go
// parseFormatReq maps a request's format string to a gifjob.Format, defaulting
// to GIF for empty or unknown values.
func parseFormatReq(s string) gifjob.Format {
	if f := gifjob.Format(s); f.Valid() {
		return f
	}
	return gifjob.FormatGIF
}

// parseAspectReq maps a request's aspect string to a gifjob.Aspect, defaulting
// to free for empty or unknown values.
func parseAspectReq(s string) gifjob.Aspect {
	if a := gifjob.Aspect(s); a.Valid() {
		return a
	}
	return gifjob.AspectFree
}

// parseDitherReq maps a request's dither string to a gifjob.DitherMethod,
// defaulting to sierra2 for empty or unknown values.
func parseDitherReq(s string) gifjob.DitherMethod {
	switch gifjob.DitherMethod(s) {
	case gifjob.DitherNone, gifjob.DitherBayer, gifjob.DitherSierra, gifjob.DitherFloyd:
		return gifjob.DitherMethod(s)
	}
	return gifjob.DitherSierra
}

// reqSpeed normalizes a request speed: a non-positive value (the zero value, or
// a frontend that omitted it) means normal speed.
func reqSpeed(s float64) float64 {
	if s <= 0 {
		return 1
	}
	return s
}

// reqWebPQuality defaults a non-positive quality to 75.
func reqWebPQuality(q int) int {
	if q <= 0 {
		return 75
	}
	return q
}
```

Rewrite `videoConfig`:

```go
func videoConfig(req VideoRequest) gifjob.VideoConfig {
	return gifjob.VideoConfig{
		Input:     req.Input,
		StartMS:   req.StartMS,
		EndMS:     req.EndMS,
		FPS:       req.FPS,
		Width:     req.Width,
		SrcWidth:  req.SrcWidth,
		SrcHeight: req.SrcHeight,
		Aspect:    parseAspectReq(req.Aspect),
		Speed:     reqSpeed(req.Speed),
		Reverse:   req.Reverse,
		Boomerang: req.Boomerang,
		Loop:      parseLoopMode(req.Loop),
		Format:    parseFormatReq(req.Format),
		Quality: gifjob.Quality{
			MaxColors:   req.Colors,
			Dither:      parseDitherReq(req.Dither),
			WebPQuality: reqWebPQuality(req.WebPQuality),
		},
	}
}
```

Rewrite `imagesConfig` (height now aspect-aware, so it takes the request AND the computed height, keeping the same signature `imagesConfig(req ImagesRequest, height int)`):

```go
func imagesConfig(req ImagesRequest, height int) gifjob.ImagesConfig {
	return gifjob.ImagesConfig{
		Inputs:    req.Inputs,
		FrameMS:   req.FrameMS,
		Width:     req.Width,
		Height:    height,
		Speed:     reqSpeed(req.Speed),
		Reverse:   req.Reverse,
		Boomerang: req.Boomerang,
		Loop:      parseLoopMode(req.Loop),
		Format:    parseFormatReq(req.Format),
		Quality: gifjob.Quality{
			MaxColors:   req.Colors,
			Dither:      parseDitherReq(req.Dither),
			WebPQuality: reqWebPQuality(req.WebPQuality),
		},
	}
}
```

In `ConvertImages`, compute the initial height with the aspect, and use the same aspect in the fit-loop recompute:

```go
	aspect := parseAspectReq(req.Aspect)
	height := gifjob.OutputHeight(frame.Width, frame.Height, aspect, req.Width)
	cfg := imagesConfig(req, height)
	// ... inside the fit loop where it currently recomputes cfg.Height:
	cfg.Height = gifjob.OutputHeight(frame.Width, frame.Height, aspect, width)
```

(`ConvertVideo` needs no height math - the engine derives it from `SrcWidth/SrcHeight/Aspect`. Everything else in ConvertVideo/ConvertImages - the cancel context, progress, fit loop, SetPreview - is unchanged.)

- [ ] **Step 5: Update the existing binding tests** - in `convert_test.go`/`binding_test.go`/`e2e_test.go`, change every `Dither: true` to `Dither: "sierra2"` and every `Dither: false` to `Dither: "none"`. Confirm no other reference treats `req.Dither` as a bool.

- [ ] **Step 6: Regenerate the TS bindings** - from `frontend/`, run `wails generate module` (regenerates `frontend/wailsjs/go/models.ts` from the Go structs). If the `wails` CLI is unavailable in this environment, hand-edit `frontend/wailsjs/go/models.ts`: in the `VideoRequest` and `ImagesRequest` classes add the new fields (`SrcWidth`, `SrcHeight`, `Aspect`, `Speed`, `Reverse`, `Boomerang`, `WebPQuality`, `Format` and change `Dither` from `boolean` to `string`) both as class properties and in the constructor's `this.X = source["X"]` assignments, mirroring the existing field pattern in that file. Verify the file parses: `cd frontend && npx tsc --noEmit -p tsconfig.json` (or `npm run build`).

- [ ] **Step 7: Run tests + gates** - `go test ./... && go build ./... && gofmt -l . && go vet ./...` (all clean), and `cd frontend && npm run build` (clean; then `git checkout -- dist/.gitkeep`).

- [ ] **Step 8: Add gated e2e (controller runs)** - append to `internal/app/e2e_test.go` (which is `//go:build ffmpeg`):

```go
func TestConvertVideoWebPThroughBinding(t *testing.T) {
	tools, err := ffmpeg.Tools()
	if err != nil {
		t.Skipf("no ffmpeg: %v", err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "in.mp4")
	if err := ffmpeg.Run(context.Background(), tools.FFmpeg,
		[]string{"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15", "-c:v", "mpeg4", "-q:v", "3", in}, nil); err != nil {
		t.Fatalf("making test video: %v", err)
	}
	a := NewApp()
	a.ctx = context.Background()
	out := filepath.Join(dir, "out.webp")
	req := VideoRequest{
		Input: in, StartMS: 0, EndMS: 1500, FPS: 10, Width: 200, SrcWidth: 320, SrcHeight: 240,
		Aspect: "1:1", Speed: 1, Loop: "forever", Colors: 256, Dither: "sierra2", WebPQuality: 80,
		Format: "webp", Out: out,
	}
	res, err := a.ConvertVideo(req)
	if err != nil {
		t.Fatalf("ConvertVideo webp = %v", err)
	}
	if res.Bytes < 100 {
		t.Errorf("webp output too small: %d bytes", res.Bytes)
	}
	if res.Width != res.Height {
		t.Errorf("1:1 aspect output %dx%d is not square", res.Width, res.Height)
	}
}
```

- [ ] **Step 9: Commit**

```bash
git add internal/app/convert.go internal/app/convert_test.go internal/app/binding_test.go internal/app/e2e_test.go frontend/wailsjs/go/models.ts
git commit -m "feat(app): expose format, aspect, speed, reverse, boomerang, dither and webp quality through the binding layer"
```

Report DONE_WITH_CONCERNS noting the controller must run the gated e2e.

---

### Task 2: Backend - content-type-aware preview

**Files:**
- Modify: `internal/app/preview.go`
- Create/Modify: `internal/app/preview_test.go`

**Interfaces:**
- Consumes: `App.SetPreview(path)`, `App.PreviewHandler() http.Handler` (existing).
- Produces: same handler, now setting `Content-Type` by the preview file's extension.

- [ ] **Step 1: Read `internal/app/preview.go`** to see `SetPreview`, `previewPath`, and the current handler (it hardcodes `image/gif` + `Cache-Control: no-store` and 404s on empty).

- [ ] **Step 2: Write the failing test** (`internal/app/preview_test.go`):

```go
package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewContentTypeByExt(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"out.gif":  "image/gif",
		"out.webp": "image/webp",
		"out.png":  "image/apng",
	}
	for name, wantCT := range cases {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("stub-bytes"), 0o644); err != nil {
			t.Fatal(err)
		}
		a := NewApp()
		a.SetPreview(p)
		rec := httptest.NewRecorder()
		a.PreviewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/preview.gif", nil))
		if got := rec.Header().Get("Content-Type"); got != wantCT {
			t.Errorf("%s -> Content-Type %q, want %q", name, got, wantCT)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s -> Cache-Control %q, want no-store", name, got)
		}
	}
}
```

- [ ] **Step 3: Run to verify it fails** - `go test ./internal/app/ -run TestPreviewContentTypeByExt` -> FAIL (webp/apng come back as image/gif).

- [ ] **Step 4: Implement** - in `preview.go`'s handler, replace the hardcoded content type with a switch on `strings.ToLower(filepath.Ext(previewPath))`:

```go
	ct := "image/gif"
	switch strings.ToLower(filepath.Ext(path)) { // path = the current previewPath
	case ".webp":
		ct = "image/webp"
	case ".png":
		ct = "image/apng"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
```

(Add `"path/filepath"` and `"strings"` to the imports if not present. Keep the existing empty-path 404 and the file-serving logic.)

- [ ] **Step 5: Run to verify it passes** - `go test ./internal/app/ -run TestPreviewContentTypeByExt && go test ./... && gofmt -l . && go vet ./...` -> all clean.

- [ ] **Step 6: Commit**

```bash
git add internal/app/preview.go internal/app/preview_test.go
git commit -m "feat(app): serve the preview with a content-type matching the output format"
```

---

### Task 3: Frontend lib - settings and source stores

**Files:**
- Create: `frontend/src/lib/settings.ts`, `frontend/src/lib/source.ts`
- Create: `frontend/src/lib/settings.test.ts`

**Interfaces:**
- Produces:
  - `type Format = "gif" | "webp" | "apng"`, `type Aspect = "" | "1:1" | "16:9" | "9:16"`, `type DitherMethod = "none" | "bayer" | "sierra2" | "floyd"`, `type LoopChoice = "forever" | "once" | "custom"`.
  - `interface Settings` and `function defaultSettings(): Settings`; `const settings: Writable<Settings>`.
  - `type Source = { kind: "video"; info: VideoInfo } | { kind: "images"; items: ImageInfo[] }`; `const source: Writable<Source | null>`.

- [ ] **Step 1: Write the failing test** (`frontend/src/lib/settings.test.ts`):

```ts
import { describe, it, expect } from "vitest";
import { defaultSettings } from "./settings";

describe("defaultSettings", () => {
  it("starts as a dithered 256-color GIF at normal speed, no crop, forever loop", () => {
    const s = defaultSettings();
    expect(s.format).toBe("gif");
    expect(s.aspect).toBe("");
    expect(s.colors).toBe(256);
    expect(s.dither).toBe("sierra2");
    expect(s.webpQuality).toBe(75);
    expect(s.speed).toBe(1);
    expect(s.reverse).toBe(false);
    expect(s.boomerang).toBe(false);
    expect(s.loopChoice).toBe("forever");
    expect(s.width).toBe(480);
    expect(s.fps).toBe(15);
    expect(s.frameMs).toBe(100);
    expect(s.targetMB).toBe(0);
    expect(s.preset).toBe("");
  });
});
```

- [ ] **Step 2: Run to verify it fails** - `cd frontend && npx vitest run src/lib/settings.test.ts` -> FAIL (module not found).

- [ ] **Step 3: Implement `frontend/src/lib/settings.ts`**

```ts
import { writable, type Writable } from "svelte/store";

export type Format = "gif" | "webp" | "apng";
export type Aspect = "" | "1:1" | "16:9" | "9:16";
export type DitherMethod = "none" | "bayer" | "sierra2" | "floyd";
export type LoopChoice = "forever" | "once" | "custom";

// Settings holds every SHARED output option (everything that does not depend on
// whether the source is a video or images). Source-specific timing (trim, fps,
// frame duration) is owned by the timing components, not here.
export interface Settings {
  format: Format;
  aspect: Aspect;
  width: number;
  fps: number; // used when the source is a video
  frameMs: number; // used when the source is images
  loopChoice: LoopChoice;
  loopCount: number;
  speed: number; // 0.25..4, 1 = normal
  reverse: boolean;
  boomerang: boolean;
  colors: number; // GIF palette size 2..256
  dither: DitherMethod; // GIF only
  webpQuality: number; // WebP only, 0..100
  targetMB: number; // 0 = no target
  preset: string; // "" | "discord" | "slack" | "twitter"
}

export function defaultSettings(): Settings {
  return {
    format: "gif",
    aspect: "",
    width: 480,
    fps: 15,
    frameMs: 100,
    loopChoice: "forever",
    loopCount: 2,
    speed: 1,
    reverse: false,
    boomerang: false,
    colors: 256,
    dither: "sierra2",
    webpQuality: 75,
    targetMB: 0,
    preset: "",
  };
}

export const settings: Writable<Settings> = writable<Settings>(defaultSettings());
```

- [ ] **Step 4: Implement `frontend/src/lib/source.ts`**

```ts
import { writable, type Writable } from "svelte/store";
import type { VideoInfo, ImageInfo } from "./wails";

// Source is the single picked input: either one video (trimmed) or an ordered
// set of images (sequenced). A null source is the empty state.
export type Source =
  | { kind: "video"; info: VideoInfo }
  | { kind: "images"; items: ImageInfo[] };

export const source: Writable<Source | null> = writable<Source | null>(null);
```

- [ ] **Step 5: Run to verify it passes** - `cd frontend && npx vitest run src/lib/settings.test.ts` -> PASS; `npm run build` clean; restore `dist/.gitkeep`.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/lib/settings.ts frontend/src/lib/source.ts frontend/src/lib/settings.test.ts
git commit -m "feat(ui): shared settings store and source union for the unified screen"
```

---

### Task 4: Frontend lib - format math (aspect height, size estimate, extensions)

**Files:**
- Modify: `frontend/src/lib/format.ts`
- Modify: `frontend/src/lib/format.test.ts`

**Interfaces:**
- Consumes: `Format`, `Aspect` (Task 3).
- Produces:
  - `outputHeight(srcW: number, srcH: number, aspect: Aspect, outW: number): number` - mirrors Go `OutputHeight`.
  - `formatExt(format: Format): string` -> ".gif"/".webp"/".png"; `formatLabel(format: Format): string` -> "GIF"/"WebP"/"APNG".
  - `estimateBytes(o: { format: Format; width: number; height: number; frames: number; colors: number; dither: boolean; webpQuality: number }): number` (REPLACES the old positional signature).

- [ ] **Step 1: Rewrite the estimate + add tests** - update `frontend/src/lib/format.test.ts` for the new `estimateBytes` object signature and add `outputHeight`/`formatExt`/`formatLabel` tests:

```ts
import { outputHeight, formatExt, formatLabel, estimateBytes } from "./format";

describe("outputHeight", () => {
  it("free aspect follows the source ratio (even)", () => {
    expect(outputHeight(1600, 900, "", 800)).toBe(450);
  });
  it("1:1 is square, 16:9 and 9:16 follow the ratio (even)", () => {
    expect(outputHeight(1600, 900, "1:1", 200)).toBe(200);
    expect(outputHeight(1600, 900, "16:9", 200) % 2).toBe(0);
    expect(outputHeight(1600, 900, "9:16", 200)).toBe(356);
  });
  it("degenerate inputs floor at 2", () => {
    expect(outputHeight(0, 0, "", 0)).toBe(2);
  });
});

describe("format ext/label", () => {
  it("maps each format", () => {
    expect(formatExt("gif")).toBe(".gif");
    expect(formatExt("webp")).toBe(".webp");
    expect(formatExt("apng")).toBe(".png");
    expect(formatLabel("apng")).toBe("APNG");
  });
});

describe("estimateBytes", () => {
  const base = { width: 200, height: 200, frames: 20, colors: 256, dither: true, webpQuality: 75 };
  it("is zero for non-positive dimensions", () => {
    expect(estimateBytes({ ...base, format: "gif", width: 0 })).toBe(0);
  });
  it("orders apng > gif > webp for the same content", () => {
    const gif = estimateBytes({ ...base, format: "gif" });
    const webp = estimateBytes({ ...base, format: "webp" });
    const apng = estimateBytes({ ...base, format: "apng" });
    expect(webp).toBeLessThan(gif);
    expect(apng).toBeGreaterThan(gif);
  });
  it("webp estimate grows with quality", () => {
    const lo = estimateBytes({ ...base, format: "webp", webpQuality: 20 });
    const hi = estimateBytes({ ...base, format: "webp", webpQuality: 95 });
    expect(hi).toBeGreaterThan(lo);
  });
});
```

- [ ] **Step 2: Run to verify it fails** - `cd frontend && npx vitest run src/lib/format.test.ts` -> FAIL (new exports missing; old estimateBytes signature).

- [ ] **Step 3: Implement in `frontend/src/lib/format.ts`** - keep `msToTimecode`/`fpsFromFrameMs`/`frameMsFromFps`/`humanBytes`/`aspectHeight` as they are; add:

```ts
import type { Format, Aspect } from "./settings";

// outputHeight mirrors Go's gifjob.OutputHeight: for the presets the height
// follows the target ratio; for free it follows the source ratio. Even, min 2.
export function outputHeight(srcW: number, srcH: number, aspect: Aspect, outW: number): number {
  if (outW <= 0) return 2;
  let h: number;
  switch (aspect) {
    case "1:1":
      h = outW;
      break;
    case "16:9":
      h = Math.round((outW * 9) / 16);
      break;
    case "9:16":
      h = Math.round((outW * 16) / 9);
      break;
    default:
      if (srcW <= 0 || srcH <= 0) return 2;
      h = Math.round((outW * srcH) / srcW);
  }
  if (h < 2) h = 2;
  if (h % 2) h++;
  return h;
}

export function formatExt(format: Format): string {
  if (format === "webp") return ".webp";
  if (format === "apng") return ".png";
  return ".gif";
}

export function formatLabel(format: Format): string {
  if (format === "webp") return "WebP";
  if (format === "apng") return "APNG";
  return "GIF";
}

// estimateBytes is a rough live readout, not a byte-accurate prediction (real
// size depends on frame content this function cannot see). GIF uses a palette-
// and dither-scaled bits-per-pixel; WebP is lossy (bpp scales with quality) and
// smaller; APNG is lossless and larger.
export function estimateBytes(o: {
  format: Format;
  width: number;
  height: number;
  frames: number;
  colors: number;
  dither: boolean;
  webpQuality: number;
}): number {
  if (o.width <= 0 || o.height <= 0 || o.frames <= 0) return 0;
  let bpp: number;
  switch (o.format) {
    case "webp":
      bpp = 0.06 + (Math.max(0, Math.min(100, o.webpQuality)) / 100) * 0.34;
      break;
    case "apng":
      bpp = 0.9;
      break;
    default:
      bpp = (0.12 + (o.colors / 256) * 0.45) * (o.dither ? 1.15 : 1.0);
  }
  return Math.round(o.frames * o.width * o.height * bpp);
}
```

- [ ] **Step 4: Run to verify it passes** - `cd frontend && npx vitest run src/lib/format.test.ts` -> PASS. (`npm run build` will fail until the old Video/Images callers of the positional `estimateBytes` are gone - those files are deleted in Task 8; for THIS task run only vitest, which does not compile the components. Note this in the report.)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/format.ts frontend/src/lib/format.test.ts
git commit -m "feat(ui): aspect-aware output height, format extensions and a format-aware size estimate"
```

---

### Task 5: Frontend lib - request builders and platform presets

**Files:**
- Create: `frontend/src/lib/request.ts`
- Modify: `frontend/src/lib/validate.ts`
- Create: `frontend/src/lib/request.test.ts`
- Modify: `frontend/src/lib/validate.test.ts`

**Interfaces:**
- Consumes: `Settings` (Task 3), `formatExt` (Task 4), `VideoInfo`/`ImageInfo`/`VideoRequest`/`ImagesRequest` (lib/wails).
- Produces:
  - `PLATFORM_PRESETS: Record<string, { label: string; maxWidth: number; targetMB: number }>` with keys `discord`,`slack`,`twitter`.
  - `deriveOut(inputPath: string, format: Format): string` and `deriveImagesOut(firstPath: string, format: Format): string`.
  - `loopValue(s: Settings): string`.
  - `buildVideoRequest(info: VideoInfo, s: Settings, timing: { startMs: number; endMs: number }): VideoRequest`.
  - `buildImagesRequest(items: ImageInfo[], s: Settings): ImagesRequest`.
  - `settingsValid(s: Settings): string | null` (in validate.ts).

- [ ] **Step 1: Write the failing tests** (`frontend/src/lib/request.test.ts`):

```ts
import { describe, it, expect } from "vitest";
import { buildVideoRequest, buildImagesRequest, deriveOut, PLATFORM_PRESETS } from "./request";
import { defaultSettings } from "./settings";

describe("buildVideoRequest", () => {
  it("carries every option and derives the output extension from the format", () => {
    const info = { Path: "C:/clips/a.mp4", DurationMS: 3000, Width: 1280, Height: 720, FPS: 30 };
    const s = { ...defaultSettings(), format: "webp" as const, aspect: "1:1" as const, width: 320, fps: 24, speed: 2, reverse: true, boomerang: true, colors: 128, dither: "bayer" as const, webpQuality: 82, loopChoice: "once" as const, targetMB: 5 };
    const req = buildVideoRequest(info, s, { startMs: 100, endMs: 2500 });
    expect(req.Input).toBe("C:/clips/a.mp4");
    expect(req.Out.endsWith(".webp")).toBe(true);
    expect(req.Format).toBe("webp");
    expect(req.Aspect).toBe("1:1");
    expect(req.SrcWidth).toBe(1280);
    expect(req.SrcHeight).toBe(720);
    expect(req.Speed).toBe(2);
    expect(req.Reverse).toBe(true);
    expect(req.Boomerang).toBe(true);
    expect(req.FPS).toBe(24);
    expect(req.Width).toBe(320);
    expect(req.Dither).toBe("bayer");
    expect(req.WebPQuality).toBe(82);
    expect(req.Loop).toBe("once");
    expect(req.StartMS).toBe(100);
    expect(req.EndMS).toBe(2500);
    expect(req.TargetKB).toBe(Math.round(5 * 1024));
  });
});

describe("buildImagesRequest", () => {
  it("maps the ordered paths, frame duration and a timestamped output", () => {
    const items = [
      { Path: "C:/pics/1.png", Width: 100, Height: 80 },
      { Path: "C:/pics/2.png", Width: 100, Height: 80 },
    ];
    const s = { ...defaultSettings(), format: "apng" as const, frameMs: 120, width: 400 };
    const req = buildImagesRequest(items, s);
    expect(req.Inputs).toEqual(["C:/pics/1.png", "C:/pics/2.png"]);
    expect(req.FrameMS).toBe(120);
    expect(req.Format).toBe("apng");
    expect(req.Out.endsWith(".png")).toBe(true);
  });
});

describe("presets", () => {
  it("defines discord, slack and twitter with a max width and target MB", () => {
    for (const key of ["discord", "slack", "twitter"]) {
      expect(PLATFORM_PRESETS[key].maxWidth).toBeGreaterThan(0);
      expect(PLATFORM_PRESETS[key].targetMB).toBeGreaterThan(0);
    }
  });
});

describe("deriveOut", () => {
  it("swaps the extension to match the format", () => {
    expect(deriveOut("C:/x/a.mp4", "gif")).toBe("C:/x/a.gif");
    expect(deriveOut("C:/x/a.mov", "webp")).toBe("C:/x/a.webp");
  });
});
```

Add to `frontend/src/lib/validate.test.ts`:

```ts
import { settingsValid } from "./validate";
import { defaultSettings } from "./settings";

describe("settingsValid", () => {
  it("passes defaults, rejects bad width/speed/colors", () => {
    expect(settingsValid(defaultSettings())).toBeNull();
    expect(settingsValid({ ...defaultSettings(), width: 0 })).toMatch(/width/i);
    expect(settingsValid({ ...defaultSettings(), speed: 0 })).toMatch(/speed/i);
    expect(settingsValid({ ...defaultSettings(), speed: 9 })).toMatch(/speed/i);
    expect(settingsValid({ ...defaultSettings(), colors: 1 })).toMatch(/color/i);
  });
});
```

- [ ] **Step 2: Run to verify it fails** - `cd frontend && npx vitest run src/lib/request.test.ts src/lib/validate.test.ts` -> FAIL.

- [ ] **Step 3: Implement `frontend/src/lib/request.ts`**

```ts
import type { VideoInfo, ImageInfo, VideoRequest, ImagesRequest } from "./wails";
import type { Settings, Format } from "./settings";
import { formatExt } from "./format";

// PLATFORM_PRESETS set a max width and a size target for common chat platforms.
export const PLATFORM_PRESETS: Record<string, { label: string; maxWidth: number; targetMB: number }> = {
  discord: { label: "Discord", maxWidth: 480, targetMB: 8 },
  slack: { label: "Slack", maxWidth: 480, targetMB: 2 },
  twitter: { label: "Twitter", maxWidth: 506, targetMB: 15 },
};

// swapExt replaces (or appends) the extension of a path without touching its
// directory. Windows and POSIX separators are both handled.
function swapExt(path: string, ext: string): string {
  const slash = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
  const dot = path.lastIndexOf(".");
  const base = dot > slash ? path.slice(0, dot) : path;
  return base + ext;
}

// deriveOut names a video's output next to the source with the format's ext.
export function deriveOut(inputPath: string, format: Format): string {
  return swapExt(inputPath, formatExt(format));
}

// deriveImagesOut names an images output next to the first image with a
// timestamp (so repeated runs in one folder do not collide) and the format ext.
export function deriveImagesOut(firstPath: string, format: Format): string {
  const slash = Math.max(firstPath.lastIndexOf("\\"), firstPath.lastIndexOf("/"));
  const dir = slash >= 0 ? firstPath.slice(0, slash + 1) : "";
  return `${dir}gifly-${Date.now()}${formatExt(format)}`;
}

// loopValue turns the loop choice into the string the backend parses.
export function loopValue(s: Settings): string {
  return s.loopChoice === "custom" ? String(Math.max(1, Math.round(s.loopCount))) : s.loopChoice;
}

function targetKB(targetMB: number): number {
  const mb = Number.isFinite(targetMB) && targetMB > 0 ? targetMB : 0;
  return Math.round(mb * 1024);
}

export function buildVideoRequest(info: VideoInfo, s: Settings, timing: { startMs: number; endMs: number }): VideoRequest {
  return {
    Input: info.Path,
    StartMS: timing.startMs,
    EndMS: timing.endMs,
    FPS: Math.round(s.fps),
    Width: Math.round(s.width),
    SrcWidth: info.Width,
    SrcHeight: info.Height,
    Aspect: s.aspect,
    Speed: s.speed,
    Reverse: s.reverse,
    Boomerang: s.boomerang,
    Loop: loopValue(s),
    Colors: Math.round(s.colors),
    Dither: s.dither,
    WebPQuality: Math.round(s.webpQuality),
    Format: s.format,
    Out: deriveOut(info.Path, s.format),
    TargetKB: targetKB(s.targetMB),
  };
}

export function buildImagesRequest(items: ImageInfo[], s: Settings): ImagesRequest {
  return {
    Inputs: items.map((i) => i.Path),
    FrameMS: Math.round(s.frameMs),
    Width: Math.round(s.width),
    Aspect: s.aspect,
    Speed: s.speed,
    Reverse: s.reverse,
    Boomerang: s.boomerang,
    Loop: loopValue(s),
    Colors: Math.round(s.colors),
    Dither: s.dither,
    WebPQuality: Math.round(s.webpQuality),
    Format: s.format,
    Out: deriveImagesOut(items[0].Path, s.format),
    TargetKB: targetKB(s.targetMB),
  };
}
```

Add `settingsValid` to `frontend/src/lib/validate.ts`:

```ts
import type { Settings } from "./settings";

// settingsValid checks the SHARED output options; the timing components add
// their own trim/frame-duration checks via videoValid/imagesValid.
export function settingsValid(s: Settings): string | null {
  if (s.width <= 0) return "Output width must be positive";
  if (!(s.speed >= 0.25 && s.speed <= 4)) return "Speed must be between 0.25 and 4";
  if (s.colors < 2 || s.colors > 256) return "Colors must be between 2 and 256";
  return null;
}
```

- [ ] **Step 4: Run to verify it passes** - `cd frontend && npx vitest run` -> all lib tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/request.ts frontend/src/lib/validate.ts frontend/src/lib/request.test.ts frontend/src/lib/validate.test.ts
git commit -m "feat(ui): pure request builders, platform presets and settings validation"
```

---

### Task 6: Unified Workspace - source, timing, convert flow (GIF parity, no regression)

Build the single-screen orchestrator and the source/timing pieces. This task proves the unification end to end at feature-parity with today: pick a video OR images and produce a GIF (default settings) with progress, preview, and result. The full option UI arrives in Task 7; here the `settings` store stays at defaults except width/fps/frameMs which the timing controls set.

**Files:**
- Create: `frontend/src/components/Workspace.svelte`, `frontend/src/components/SourcePanel.svelte`, `frontend/src/components/VideoTiming.svelte`, `frontend/src/components/ImagesTiming.svelte`
- Modify: `frontend/src/App.svelte`

**Interfaces:**
- Consumes: `source`/`Source` (Task 3), `settings` (Task 3), `buildVideoRequest`/`buildImagesRequest` (Task 5), `PickVideo`/`PickImages`/`ConvertVideo`/`ConvertImages`/`Cancel`/`RevealOutput`/`onProgress` (lib/wails), `outputHeight`/`estimateBytes`/`humanBytes`/`msToTimecode`/`formatExt`/`formatLabel` (lib/format), `videoValid`/`imagesValid`/`settingsValid` (lib/validate).
- Produces: a mounted `Workspace` that App renders. `SourcePanel`/`VideoTiming`/`ImagesTiming` read/write the `source` store and Workspace-local timing state via props/events.

**Structure spec** (use the existing `modes/Video.svelte` and `modes/Images.svelte` as the pattern to mirror - same style.css classes, same convert/progress/onDestroy discipline):

- `Workspace.svelte` owns: `let src = get(source)` via `$source`; local timing `startSec`,`endSec` (video) and reads `frameMs`/`fps`/`width` from `$settings`; `converting`,`progressPhase`,`progressPercent`,`stopProgress`,`result`,`convertError`,`previewUrl`. It renders, top to bottom: `<SourcePanel/>`, then if a source exists: the timing component for the kind, a temporary minimal actions block (Make button + validation hint), progress, error, result card. The convert function:
  - builds the request via `buildVideoRequest($source.info, $settings, { startMs, endMs })` or `buildImagesRequest($source.items, $settings)`;
  - calls `ConvertVideo`/`ConvertImages`; on success sets `previewUrl = \`/preview.gif?t=${Date.now()}\``; wires `onProgress`; `onDestroy` stops progress + `Cancel()` if converting (mirror the existing components exactly).
  - Make-button label uses `formatLabel($settings.format)` (e.g. "Make GIF").
  - `canConvert` = source present AND not converting AND `settingsValid($settings) == null` AND the kind's timing validation passes (`videoValid`/`imagesValid`).
- `SourcePanel.svelte`:
  - Empty state (`$source == null`): a `.pick-panel` with two `.btn-primary`/`.btn-ghost` buttons - "Pick a video" (calls `PickVideo`, on success sets `source = { kind:"video", info }` and initializes Workspace trim to full duration via an event/prop) and "Add images" (calls `PickImages`, sets `source = { kind:"images", items }`). A one-line `.hint`: "Trim a video, or sequence a set of images." Handle the `context.Canceled` reject quietly (user closed the dialog): treat a cancel as no-op, not an error.
  - Video source: a `.section` with `.source-row` (eyebrow "Source" + "Change" button re-invoking PickVideo) + `.source-name` + a `.meta` dl (Duration via `msToTimecode`, Source size `WxH`).
  - Images source: a `.section` with `.source-row` (eyebrow "Source (N)" + "Add images" + a "Clear" `.btn-ghost` that empties the list) + the reorderable `<ol class="list">` (thumb, name, dims, Up/Down/Remove) - port this list verbatim from `modes/Images.svelte` including its `<style>` and the `fileUrl`/`onThumbError`/`moveUp`/`moveDown`/`removeAt`/`extOf` helpers and the thumbnail fallback.
- `VideoTiming.svelte`: a `.section` "Timing" with Start/End numeric fields (seconds, clamped to `[0, durationSec]`, timecode readouts via `msToTimecode`) + an FPS field bound to `$settings.fps` + FPS preset chips `[10,15,24,30]`. Emits/binds `startMs`,`endMs` back to Workspace (via `bind:` props or a store). Port the clamp logic from `modes/Video.svelte`.
- `ImagesTiming.svelte`: a `.section` "Timing" with a per-frame Duration (ms) field bound to `$settings.frameMs`, a `= N fps` readout via `fpsFromFrameMs`, and FPS preset chips that set `frameMs = frameMsFromFps(preset)`.

**App.svelte** becomes:

```svelte
<script lang="ts">
  import "./style.css";
  import TitleBar from "./components/TitleBar.svelte";
  import Workspace from "./components/Workspace.svelte";
</script>

<div id="app">
  <TitleBar />
  <main class="work">
    <Workspace />
  </main>
</div>
```

- [ ] **Step 1: Build the components** per the structure spec (mirror the existing modes for the convert/progress/result/onDestroy code, which is already correct and tested against real ffmpeg). Keep each file focused; reuse `style.css` classes - do not add new visual design yet.

- [ ] **Step 2: Typecheck + build** - `cd frontend && npm run build`. Because the old `modes/Video.svelte`/`modes/Images.svelte` still import the now-removed positional `estimateBytes` and the changed `models.ts`, they may fail to compile. To keep this task's build green WITHOUT deleting them yet (Task 8 does that), TEMPORARILY stop App.svelte from importing them (App now imports only Workspace) AND, if the Vite build still compiles the orphaned mode files, delete them in THIS task instead of Task 8 (moving the deletion earlier is acceptable - update Task 8's scope note). Prefer: delete `modes/Video.svelte`, `modes/Images.svelte`, `components/ModeSwitch.svelte` here if the build requires it, and remove the `mode` store import. Report which files you removed.

- [ ] **Step 3: Restore the dist marker + run unit tests** - `git checkout -- frontend/dist/.gitkeep`; `cd frontend && npx vitest run` (lib tests still pass).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ frontend/src/App.svelte frontend/src/modes/ frontend/src/lib/wails.ts
git commit -m "feat(ui): unified Workspace with shared source and per-kind timing"
```

- [ ] **Step 5: CONTROLLER visual check** - the controller builds the Wails app (`wails build`) and confirms: pick a video -> trim -> Make GIF -> preview + result; pick images -> reorder -> Make GIF -> preview + result. No mode toggle. (Implementers cannot run the GUI.)

---

### Task 7: Option controls - format, size/aspect, playback, quality, output

Add the shared option groups as focused components bound to `$settings`, and insert them into `Workspace` between Timing and Actions. After this task the full option set is live.

**Files:**
- Create: `frontend/src/components/FormatPicker.svelte`, `SizeControls.svelte`, `PlaybackControls.svelte`, `QualityControls.svelte`, `OutputControls.svelte`
- Modify: `frontend/src/components/Workspace.svelte` (render the new sections + compute the live estimate + apply presets)

**Interfaces:**
- Consumes: `settings` store, `PLATFORM_PRESETS` (Task 5), `outputHeight`/`estimateBytes`/`humanBytes`/`formatLabel` (Task 4), `Aspect`/`Format`/`DitherMethod` types (Task 3).
- Produces: five components that read/write `$settings`. Workspace computes `outHeight` and `displayEstimate` reactively and passes source frame count.

**Structure spec** (each is a `.section` reusing existing classes; bind directly to the `settings` store, e.g. `bind:value={$settings.width}`):

- `FormatPicker.svelte`: eyebrow "Format"; a `.chip-row` of three chips GIF/WebP/APNG toggling `$settings.format` (active chip = current). One `.hint` line describing the current format (e.g. GIF "universal, 256 colors"; WebP "smaller, lossy"; APNG "lossless, larger").
- `SizeControls.svelte`: eyebrow "Size"; an aspect `.chip-row` Free/1:1/16:9/9:16 setting `$settings.aspect`; a Width `.field` (number, bound to `$settings.width`) with width preset chips `[240,360,480,640]`; a Height readout `.field` showing the computed output height (Workspace passes `outHeight` in as a prop, computed via `outputHeight(srcW, srcH, $settings.aspect, $settings.width)` where srcW/srcH come from the source - video info or first image; for images with no source yet, 0).
- `PlaybackControls.svelte`: eyebrow "Playback"; a Repeat `<select>` bound to `$settings.loopChoice` (Forever/Once/Custom) + a Times field when custom (`$settings.loopCount`); a Speed range input 0.25..4 step 0.05 bound to `$settings.speed` with a `= N x` readout; two `.toggle` checkboxes Reverse (`$settings.reverse`) and Boomerang (`$settings.boomerang`).
- `QualityControls.svelte`: eyebrow "Quality"; format-conditional (`{#if $settings.format === "gif"}` a Colors range 2..256 bound to `$settings.colors` + a Dither `.chip-row` None/Bayer/Sierra2/Floyd setting `$settings.dither`; `{:else if $settings.format === "webp"}` a Quality range 0..100 bound to `$settings.webpQuality`; `{:else}` a `.hint` "APNG is lossless - no quality options").
- `OutputControls.svelte`: eyebrow "Output"; a preset `.chip-row` None/Discord/Slack/Twitter - selecting a preset sets `$settings.preset`, `$settings.width = PLATFORM_PRESETS[key].maxWidth`, `$settings.targetMB = PLATFORM_PRESETS[key].targetMB`; a Target size (MB) `.field` bound to `$settings.targetMB` (editing it or the width sets `$settings.preset = ""`); the live estimate `.estimate` line (Workspace passes `displayEstimate`).

**Workspace changes:** insert `<FormatPicker/> <SizeControls outHeight={outHeight}/> <PlaybackControls/> <QualityControls/> <OutputControls displayEstimate={displayEstimate}/>` between the timing section and the actions. Compute reactively:

```ts
$: srcW = $source?.kind === "video" ? $source.info.Width : $source?.kind === "images" ? ($source.items[0]?.Width ?? 0) : 0;
$: srcH = $source?.kind === "video" ? $source.info.Height : $source?.kind === "images" ? ($source.items[0]?.Height ?? 0) : 0;
$: outHeight = outputHeight(srcW, srcH, $settings.aspect, $settings.width);
$: frames = $source?.kind === "video"
    ? Math.round(($settings.fps * Math.max(0, endMs - startMs)) / 1000)
    : $source?.kind === "images" ? $source.items.length : 0;
$: rawEstimate = estimateBytes({ format: $settings.format, width: Math.round($settings.width), height: outHeight, frames, colors: Math.round($settings.colors), dither: $settings.dither !== "none", webpQuality: $settings.webpQuality });
$: targetBytes = $settings.targetMB > 0 ? Math.round($settings.targetMB * 1024 * 1024) : 0;
$: displayEstimate = targetBytes > 0 ? Math.min(rawEstimate, targetBytes) : rawEstimate;
```

- [ ] **Step 1: Build the five components + wire them into Workspace** per the spec. Bind to `$settings` throughout; keep each component a single focused `.section`.

- [ ] **Step 2: Typecheck + build + unit tests** - `cd frontend && npm run build && npx vitest run` -> clean; `git checkout -- dist/.gitkeep`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/
git commit -m "feat(ui): format, size/aspect, playback, quality and output option controls"
```

- [ ] **Step 4: CONTROLLER visual + functional check** - build the Wails app and confirm: switching Format shows the right Quality controls and changes the output extension; Aspect 1:1 makes the height readout square; Speed/Reverse/Boomerang toggles reach the backend (produce a boomerang WebP, an APNG, a 1:1 GIF); a platform preset sets width + target; the estimate updates live.

---

### Task 8: Cleanup - remove the mode split

Remove whatever old mode machinery still remains (some may already be gone from Task 6) and confirm a clean whole-app build.

**Files:**
- Delete (if still present): `frontend/src/components/ModeSwitch.svelte`, `frontend/src/modes/Video.svelte`, `frontend/src/modes/Images.svelte` (and the now-empty `frontend/src/modes/` directory).
- Modify (if still present): `frontend/src/lib/wails.ts` - remove the `Mode` type and the `mode` store (no longer imported anywhere).

- [ ] **Step 1: Grep for dead references** - `cd frontend && grep -rn "ModeSwitch\|modes/Video\|modes/Images\|\\bmode\\b" src/` - confirm nothing outside the files being deleted imports them. Remove the `mode`/`Mode` export from `lib/wails.ts` if no importer remains.

- [ ] **Step 2: Delete the files** and remove the dead exports.

- [ ] **Step 3: Full build + tests** - `cd frontend && npm run build && npx vitest run` -> clean; `go build ./... && go test ./... && gofmt -l . && go vet ./...` -> clean; `git checkout -- frontend/dist/.gitkeep`.

- [ ] **Step 4: Commit**

```bash
git add -A frontend/src
git commit -m "refactor(ui): remove the Video/Images mode split now that Workspace is unified"
```

- [ ] **Step 5: CONTROLLER final build** - `wails build`, launch, and walk the whole unified flow for both source kinds and all three formats.

---

## Self-Review

**1. Spec coverage** (user decisions -> task):
- Unify Video + Images into one screen, no mode toggle -> Tasks 6 (Workspace/SourcePanel), 8 (remove ModeSwitch/mode store). Covered.
- All output options shared, only timing differs -> Task 6 (VideoTiming/ImagesTiming source-specific; settings shared), 7 (shared option components). Covered.
- Format picker GIF/WebP/APNG -> Tasks 1 (binding), 7 (FormatPicker), 2 (preview content-type). Covered.
- Crop/aspect 1:1/16:9/9:16/free -> Tasks 1 (mapping + images height via OutputHeight), 4 (outputHeight), 7 (SizeControls). Covered.
- Boomerang + reverse + speed -> Tasks 1 (mapping), 7 (PlaybackControls). Covered.
- Dither method + colors + webp quality -> Tasks 1 (mapping), 7 (QualityControls, format-conditional). Covered.
- Platform presets Discord/Slack/Twitter -> Tasks 5 (PLATFORM_PRESETS), 7 (OutputControls). Covered.
- Target size + live estimate -> Tasks 4 (estimateBytes v2), 7 (OutputControls + Workspace estimate). Covered.
- Structure/specs before visuals -> the plan builds function first (option controls reuse existing style.css, no new visual design); a later polish pass is out of scope here. Covered.

**2. Placeholder scan:** Component tasks (6-7) give structural specs + full script logic + representative code rather than every line of markup, and point at the existing `modes/*.svelte` as the concrete pattern to mirror (the convert/progress/result/onDestroy code is ported verbatim from working, real-ffmpeg-tested components). This is deliberate - the components are view glue over the tested lib logic - not a "TODO". No `TBD`/`implement later`/vague-error placeholders remain. Backend (1-2) and lib (3-5) tasks are fully coded with concrete tests.

**3. Type consistency:** `Settings`/`Source`/`Format`/`Aspect`/`DitherMethod`, `buildVideoRequest(info, s, timing)`/`buildImagesRequest(items, s)`, `outputHeight(srcW, srcH, aspect, outW)`, `estimateBytes({format,width,height,frames,colors,dither,webpQuality})`, `deriveOut(path, format)`, `PLATFORM_PRESETS`, and the Go `VideoRequest`/`ImagesRequest` fields (`Dither string`, `Format`, `Aspect`, `Speed`, `Reverse`, `Boomerang`, `WebPQuality`, `SrcWidth`, `SrcHeight`) are used identically across tasks. The TS `Dither` union values (`none`/`bayer`/`sierra2`/`floyd`) match `gifjob.DitherMethod`; TS `Format`/`Aspect` values match the Go engine's string constants, so the request strings parse without translation.

**Note on the intermediate build (Task 6):** removing the mode split may need to happen in Task 6 rather than Task 8 to keep `npm run build` green (the orphaned `modes/*.svelte` reference the changed `estimateBytes`/`models.ts`). Task 6 Step 2 handles this explicitly and Task 8 tolerates the files already being gone (grep-and-confirm, delete if present).
