# gifly - Design

**Date:** 2026-08-01
**Status:** approved for planning

## Goal

A Windows desktop app that turns a video, or an ordered set of images, into an
optimized animated GIF. Two input modes, one high-quality encode path, a small
set of honest controls (fps, size, trim, loop, quality), and a one-file
distribution.

## Summary

gifly is a Wails desktop app (Go core, Svelte-TS front end) that drives a
**bundled ffmpeg** through a two-pass `palettegen` / `paletteuse` pipeline - the
only path that both decodes arbitrary video codecs and produces a genuinely good
GIF (a per-clip optimized palette with dithering, not the weak default
quantizer). The Go side never decodes video itself; it locates the bundled
ffmpeg/ffprobe, builds exact argument lists, runs them, and reports progress.
Everything the GUI shows about a source (duration, size, fps) comes from ffprobe.

## Global Constraints

- **Platform:** Windows only (matches the sibling tools flip / shape / fleet).
- **Language/stack:** Go 1.26 + Wails v2 + Svelte-TS. Single-window Wails app,
  Frameless with its own title bar, like the siblings.
- **cgo-free:** `CGO_ENABLED=0 go build ./...` must succeed. No cgo, no
  golang.org/x/image. Video/GIF work is done by the external ffmpeg process, not
  a Go library.
- **Dependencies:** minimal and pinned. `github.com/wailsapp/wails/v2` is the
  only expected direct dependency beyond the standard library (plus whatever
  Wails pulls in indirectly, e.g. golang.org/x/sys). Image reading for
  thumbnails/validation uses stdlib `image` (png/jpeg/gif) only.
- **ffmpeg is bundled**, not assumed on PATH. An **LGPL** ffmpeg build is
  shipped so the distribution can comply by dynamic-equivalent terms; its
  license text travels with it (see Distribution).
- **Copy/style:** plain ASCII `-` only, never a unicode dash. Numbers shown in
  the UI are data: a mono, tabular face (the sibling convention).
- **Author:** Hoijun Kim. Commits carry no AI co-author trailer.
- **`.gitattributes`** pins `* text=auto eol=lf` so `gofmt -l` stays clean on
  this machine (core.autocrlf=true would otherwise check Go sources out as CRLF).

## Architecture

The system is four Go packages plus the front end. Each has one responsibility
and a small, testable surface.

### `internal/ffmpeg` - the one place that shells out

Locates the bundled `ffmpeg.exe` and `ffprobe.exe` and runs them.

- `func Tools() (Paths, error)` - resolve the binaries next to the running exe
  (`<exeDir>/ffmpeg/ffmpeg.exe`, `.../ffprobe.exe`), falling back to a
  dev-mode env var (`GIFLY_FFMPEG_DIR`) and then PATH. Returns a clear error if
  neither is found, so the GUI can say exactly what is missing.
- `type Paths struct { FFmpeg, FFprobe string }`
- `func Run(ctx, bin string, args []string, onProgress func(Progress)) error` -
  run a command with `-progress pipe:1 -nostats`, parse the `key=value` progress
  stream (`out_time_ms`, `frame`, `progress`), and call `onProgress`. On a
  non-zero exit, return an error that includes the tail of ffmpeg's stderr (the
  real reason: unsupported codec, bad file, etc.). Honors `ctx` cancellation
  (kill the process) so a cancelled job stops promptly.
- `type Progress struct { OutTimeMS int64; Frame int; Done bool }`

Running is behind a tiny interface so `gifjob` can be tested with a fake runner
that records the argv and never launches a process.

### `internal/probe` - read a source's shape

- `func Video(ctx, ffprobe, path string) (Media, error)` - one ffprobe call
  (`-print_format json -show_format -show_streams`), parsed into:
  `type Media struct { DurationMS int64; Width, Height int; FPS float64 }`.
  Feeds the trim range, the default output width, and the fps ceiling in the UI.
- `func Image(path string) (Frame, error)` - stdlib decode of a still to get
  `Width, Height` (and validate it is a real PNG/JPEG/GIF) for the images list.

### `internal/gifjob` - config -> exact ffmpeg argv -> run

The heart. Pure argument construction is separated from execution so it can be
unit-tested without ffmpeg.

- `type VideoConfig struct { Input string; StartMS, EndMS int64; FPS int; Width int; Loop LoopMode; Quality Quality }`
- `type ImagesConfig struct { Inputs []string; FrameMS int; Width int; Loop LoopMode; Quality Quality }`
  (`FrameMS` is per-frame duration; a uniform value = fps of 1000/FrameMS.)
- `type LoopMode int` - `Forever` (ffmpeg `-loop 0`), `Once` (`-loop -1`),
  `Times(n)` (`-loop n`).
- `type Quality struct { MaxColors int; Dither bool }` - MaxColors 2..256
  (default 256), Dither default on (`sierra2_4a`); off = `dither=none`.
- `func VideoArgs(c VideoConfig, palettePath, outPath string) (pass1, pass2 []string, err error)`
- `func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (pass1, pass2 []string, err error)`
- `func WriteConcatList(w io.Writer, inputs []string, frameMS int) error` - the
  images concat-demuxer list (see Data Flow), repeating the final frame because
  the concat demuxer drops the last `duration`.
- `func RunVideo(ctx, tools, c VideoConfig, out string, onProgress) (Result, error)`
  and `RunImages(...)` - orchestrate: temp palette (and list) file, pass 1, pass
  2, clean up temps, stat the result. `type Result struct { Path string; Bytes int64; Width, Height int }`.

Validation (`ValidateVideo` / `ValidateImages`) refuses an empty input, a
non-positive fps/width, `EndMS <= StartMS`, MaxColors out of 2..256, or an empty
image list - each with its own message - before any process starts.

### `internal/app` - Wails bindings

Bridges the GUI and `gifjob`. Bound methods: pick a video / pick images / probe
a source / start a conversion (emitting progress events) / cancel / reveal the
output in Explorer. Holds the single in-flight job and its `context.CancelFunc`.
Serves a preview of the finished GIF (a `/preview.gif` handler, cache-busted,
the pattern flip's preview uses).

### Front end (Svelte-TS)

One window, a mode switch at the top: **Video** or **Images**.

- Video mode: a drop/pick target; once a file is chosen, show its duration and
  size, a trim range (two handles + numeric start/end), fps, output width
  (keep-aspect, shown with the resulting height), loop, and quality (colors +
  dither). A "Make GIF" button, a progress bar during the run, then the result
  (dimensions, file size, a preview, "Open folder" / "Save as").
- Images mode: a drop/pick target that adds to an ordered list with drag-to-
  reorder and per-item remove; a per-frame duration (ms) with a live "= N fps"
  read-out; output width; loop; quality. Same run/result flow.

Pure front-end logic (config validation, `ms <-> fps` and `ms <-> mm:ss.mmm`
formatting, keep-aspect height) lives in `frontend/src/lib` and is unit-tested
(vitest), the sibling pattern.

## Data Flow

### Video -> GIF (two pass)

1. On pick: `probe.Video` -> duration, dimensions, fps populate the controls.
2. On run, `gifjob.VideoArgs` builds:
   - **Pass 1 (palette):**
     `ffmpeg -y -ss <start> -i <input> -t <dur> -vf "fps=<fps>,scale=<w>:-2:flags=lanczos,palettegen=max_colors=<n>:stats_mode=diff" -f image2 <palette.png>`
   - **Pass 2 (encode):**
     `ffmpeg -y -ss <start> -i <input> -t <dur> -i <palette.png> -lavfi "fps=<fps>,scale=<w>:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=<d>" -loop <loop> <out.gif>`
   - `-ss` before `-i` for a fast seek; `-t <dur>` (dur = EndMS-StartMS) rather
     than `-to`, to avoid the well-known `-ss`/`-to` origin ambiguity.
   - `scale=<w>:-2` keeps aspect and forces an even height.
3. Progress from `-progress pipe:1` against the known trimmed duration drives the
   bar. Result is stat-ed and previewed.

### Images -> GIF (two pass, concat demuxer)

1. Each added image is validated by `probe.Image`.
2. `WriteConcatList` writes a temp list:
   ```
   file '<abs path 1>'
   duration <frameMS/1000>
   file '<abs path 2>'
   duration <frameMS/1000>
   ...
   file '<abs path N>'
   duration <frameMS/1000>
   file '<abs path N>'
   ```
   (last file repeated - the concat demuxer ignores the final `duration`.)
3. Pass 1/2 as above but with `-f concat -safe 0 -i <list.txt>` as the input and
   no `fps` filter (durations come from the list); `scale`/`palettegen`/
   `paletteuse` are identical.

## Error Handling

- **ffmpeg/ffprobe not found:** `ffmpeg.Tools` returns a specific error; the GUI
  shows "the bundled ffmpeg is missing" with the path it looked in, not a
  generic failure.
- **Bad/uncodable input:** pass 1 or 2 exits non-zero; the error carries
  ffmpeg's stderr tail so the message names the real cause.
- **Cancel:** context cancellation kills the process; partial `out.gif` and temp
  palette/list files are removed. A cancelled run reports "cancelled", distinct
  from a failure.
- **Validation** refuses impossible configs before spawning anything.
- Temp files (palette, concat list) always cleaned in a `defer`, on success and
  on every error path.

## Testing

- `gifjob` argv builders: table tests asserting the **exact** argument slices for
  representative video and images configs (trim, fps, width, each LoopMode, dither
  on/off, custom MaxColors). These are the load-bearing tests - a wrong flag is
  the most likely defect and the one hardest to see by eye.
- `WriteConcatList`: asserts ordering, per-frame duration, and the repeated final
  frame.
- `ffmpeg.Tools`: locating the binaries (found next to exe / via env / missing).
- `internal/ffmpeg` progress parsing: feed a captured `-progress` stream, assert
  the parsed `Progress` values.
- One **integration test**, gated behind a build tag that requires the bundled
  ffmpeg present: a tiny generated clip and a couple of solid-color PNGs each
  produce a valid, non-empty GIF of the expected dimensions (decoded back with
  stdlib `image/gif`).
- Front end: vitest over the pure `lib` logic (validation, ms/fps/timecode
  formatting, keep-aspect height).
- Every test is written to actually fail if the code it covers is mutated (no
  vacuous assertions) - the standard the sibling projects hold.

## Distribution

Mirrors flip's flow:

- **License:** gifly's own code under PolyForm Noncommercial 1.0.0.
- **ffmpeg licensing:** ship an **LGPL** ffmpeg build under `ffmpeg/`, with its
  `LICENSE`/`COPYING` text alongside the binaries and a `NOTICE` in the repo and
  on the download page stating ffmpeg is bundled under LGPL 2.1+. This is a hard
  release gate, not optional.
- **Public repo** `github.com/hoijun-kim/gifly`, a landing site under `docs/`
  served at `hoijun-kim.github.io/gifly/`, and a row on the work board in the
  hoijun-kim.github.io blog, like the siblings.
- **Release:** a checksummed zip of `gifly.exe` + the `ffmpeg/` folder. The zip
  is large (ffmpeg is tens of MB); the download page says so.
- Author Hoijun Kim; no AI co-author on any commit.

## Non-Goals (v1)

Crop, rotate, text/caption/watermark overlays, batch conversion, output formats
other than GIF (no APNG/WebP), a timeline scrubber beyond simple trim handles,
and any in-app video playback. Each is a clean later addition on top of the
`gifjob` config, deliberately out of the first version.
