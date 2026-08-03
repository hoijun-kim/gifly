Target output size feature - implementation report
====================================================

Scope
-----
Added a "fit to a target size" feature to gifly's backend conversion path
(internal/app/convert.go). When a caller sets TargetKB > 0 and the first
2-pass palette encode comes out larger than that budget, ConvertVideo and
ConvertImages now re-encode at shrinking widths (to the same Out path) until
the result fits, the floor width (120px) is hit, or 8 attempts are spent.

Changes
-------
1. internal/app/convert.go
   - Added `TargetKB int` to both VideoRequest and ImagesRequest (0 = no
     target, matches the JSON-friendly zero-value convention already used by
     the rest of the request structs).
   - Added `nextFitWidth(w int) int`: shrinks by ~15%, forces strict
     progress (w-2 floor step if 15% rounds to no change), rounds to even,
     and clamps to a 120px floor. Also added `fitFloorWidth` and
     `fitMaxAttempts` constants (120 and 8) so the magic numbers in the loop
     are named once.
   - ConvertVideo: after the first successful gifjob.RunVideo + implicit
     statResult (RunVideo already stats internally), if
     `req.TargetKB > 0 && result.Bytes > TargetKB*1024`, loops up to 8 times:
     computes nextFitWidth, breaks if it did not move, mutates cfg.Width,
     emits a "fitting" progress event, re-runs RunVideo to the same Out path
     (encode progress inside an attempt is also reported under phase
     "fitting" rather than "encoding"), and stops once the result fits or the
     floor width was just tried. ctx.Err() is checked at the top of each
     iteration and returns early (ConvertResult{}, ctx.Err()) on
     cancellation, consistent with how cancellation surfaces elsewhere in
     this file (as an error from the function).
   - ConvertImages: same loop shape, but also recomputes cfg.Height via
     gifjob.CanvasHeight(frame.Width, frame.Height, width) each iteration,
     reusing the Frame already probed from the first input image before the
     first encode. Progress callback was refactored into a small
     `imagesProgress(phase string) func(ffmpeg.Progress)` closure so both the
     initial "encoding" call and the "fitting" re-encodes share the same
     frame-count-based percent logic instead of duplicating it.
   - SetPreview and the returned ConvertResult always reflect the *last*
     result produced (whichever attempt the loop stopped on), so Width/Height
     in the response match what was actually written to disk.

2. internal/app/convert_test.go
   - Added TestNextFitWidth: a table-driven check that nextFitWidth(600) and
     nextFitWidth(601) both strictly decrease and stay even; an explicit
     check that nextFitWidth(120) == 120 (already at the floor) and
     nextFitWidth(122) == 120 (floors); and a loop applying nextFitWidth
     repeatedly from 2000 that asserts strict decrease on every step until it
     lands exactly on the 120 floor.

3. frontend/wailsjs/go/models.ts
   - Regenerated via `wails generate module` (wails v2.12.0). Only this file
     changed: `TargetKB: number` was added to the generated `ImagesRequest`
     and `VideoRequest` TS classes (both the field declaration and the
     constructor's `source["TargetKB"]` assignment). frontend/wailsjs/go/app/
     App.js and App.d.ts were unchanged, since the bound method signatures
     (ConvertVideo/ConvertImages taking the request object) did not change.
   - No frontend UI wiring (e.g. a Svelte input for target size) was added;
     the task scope was backend + binding regeneration only.

Design notes / judgment calls
------------------------------
- The fit loop reuses the *same* Out path for every attempt (as specified),
  so each re-encode simply overwrites the previous attempt's file; no temp
  file juggling was needed since gifjob.RunVideo/RunImages already handle
  writing to outPath and gifjob's internal statResult reads it back after
  each run.
- "AFTER the first successful encode + statResult" - RunVideo/RunImages
  already call statResult internally as their last step, so the returned
  Result from the initial call already has Bytes/Width/Height for the
  target-size check; no separate stat call was added in convert.go.
- Progress during the fit loop always starts each attempt's emitProgress at
  0 (phase "fitting") before re-running, so the UI can show a fresh
  "fitting" phase even for a fast image job where per-frame progress may not
  fire often; the ffmpeg progress callback itself still reports percent
  under "fitting" as it streams.
- Cancellation: only checked with ctx.Err() at the top of each loop
  iteration (not mid-encode) since ffmpeg.Run already honors ctx internally
  during an attempt already in flight and will surface context.Canceled as
  the attempt's err, which the loop already returns on any err.

Verification performed (no real ffmpeg available in this environment)
-----------------------------------------------------------------------
- CGO_ENABLED=0 go build ./...                                   -> clean
- CGO_ENABLED=0 go test ./internal/app/ -run "NextFitWidth|Percent|
  VideoConfig|ImagesConfig|ParseLoop|Cancel|Preview|ProbeImage" -count=1
  -> all PASS
- CGO_ENABLED=0 go test ./... -count=1                            -> all ok
- go vet -tags ffmpeg ./internal/app/                             -> exit 0
- gofmt -l .                                                      -> no output (clean)
- The //go:build ffmpeg gated end-to-end tests in internal/app/e2e_test.go
  were NOT run (no ffmpeg binary in this environment, and the task explicitly
  said not to run them here). Those tests do not yet exercise TargetKB;
  the controller should verify with real ffmpeg that the fit loop actually
  shrinks output size for a case where the first encode exceeds the target.

Files touched
-------------
- C:\Users\hoijun\Projects\gifly\internal\app\convert.go
- C:\Users\hoijun\Projects\gifly\internal\app\convert_test.go
- C:\Users\hoijun\Projects\gifly\frontend\wailsjs\go\models.ts

Files intentionally left untouched (pre-existing dirty state, unrelated to
this task, present before this session started):
- frontend/dist/.gitkeep (deleted in the working tree prior to this task)
- frontend/package.json.md5 (untracked, generated by wails, prior to this task)
