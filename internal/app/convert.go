package app

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
	"github.com/hoijun-kim/gifly/internal/gifjob"
	"github.com/hoijun-kim/gifly/internal/probe"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// VideoRequest is the JSON-friendly request sent from the frontend for a
// video-to-GIF conversion.
type VideoRequest struct {
	Input   string
	StartMS int64
	EndMS   int64
	FPS     int
	Width   int

	// SrcWidth/SrcHeight is the source video's pixel size, used to compute the
	// aspect-aware output height.
	SrcWidth  int
	SrcHeight int
	// Aspect is an optional center-crop applied before scaling ("", "1:1",
	// "16:9", "9:16").
	Aspect string
	// Speed is a playback rate multiplier; <=0 means normal speed.
	Speed     float64
	Reverse   bool
	Boomerang bool

	Loop   string
	Colors int
	// Dither is a gifjob.DitherMethod name ("none", "bayer", "sierra2",
	// "floyd"); empty or unknown defaults to sierra2.
	Dither string
	// WebPQuality is 0..100 for the WebP encoder; <=0 defaults to 75.
	WebPQuality int
	// Format is the output container ("gif", "webp", "apng"); empty or
	// unknown defaults to gif.
	Format string

	// OutDir is the folder to write into; OutName is the base file name (no
	// extension - the extension follows Format); OnExist is the collision policy
	// ("overwrite", "number", "timestamp"). resolveOut turns these into the
	// final path.
	OutDir  string
	OutName string
	OnExist string

	// TargetKB is the desired output size in kilobytes. 0 means no target: the
	// first encode is kept as-is. When positive and the first encode comes out
	// larger, ConvertVideo re-encodes at shrinking widths to fit.
	TargetKB int
}

// ImagesRequest is the JSON-friendly request sent from the frontend for an
// images-to-GIF conversion.
type ImagesRequest struct {
	Inputs  []string
	FrameMS int
	Width   int
	// Aspect is an optional center-crop applied before scaling ("", "1:1",
	// "16:9", "9:16").
	Aspect string
	// Speed is a playback rate multiplier; <=0 means normal speed.
	Speed     float64
	Reverse   bool
	Boomerang bool

	Loop   string
	Colors int
	// Dither is a gifjob.DitherMethod name ("none", "bayer", "sierra2",
	// "floyd"); empty or unknown defaults to sierra2.
	Dither string
	// WebPQuality is 0..100 for the WebP encoder; <=0 defaults to 75.
	WebPQuality int
	// Format is the output container ("gif", "webp", "apng"); empty or
	// unknown defaults to gif.
	Format string

	// OutDir is the folder to write into; OutName is the base file name (no
	// extension - the extension follows Format); OnExist is the collision policy
	// ("overwrite", "number", "timestamp"). resolveOut turns these into the
	// final path.
	OutDir  string
	OutName string
	OnExist string

	// TargetKB is the desired output size in kilobytes. 0 means no target: the
	// first encode is kept as-is. When positive and the first encode comes out
	// larger, ConvertImages re-encodes at shrinking widths to fit.
	TargetKB int
}

// ConvertResult describes a finished GIF returned to the frontend.
type ConvertResult struct {
	Path   string
	Bytes  int64
	Width  int
	Height int
}

// ProgressEvent is emitted during conversion as "convert:progress".
type ProgressEvent struct {
	Phase   string
	Percent int
}

// percent computes a progress percentage from outTimeMS and totalMS, clamped
// to 0..100. If totalMS is zero or negative, returns 0.
func percent(outTimeMS, totalMS int64) int {
	if totalMS <= 0 {
		return 0
	}
	p := (outTimeMS * 100) / totalMS
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return int(p)
}

// parseLoopMode maps a frontend loop string to a gifjob.LoopMode.
func parseLoopMode(loop string) gifjob.LoopMode {
	switch loop {
	case "forever":
		return gifjob.LoopForever
	case "once":
		return gifjob.LoopOnce
	default:
		// Try to parse as a positive integer
		var n int
		if _, err := fmt.Sscanf(loop, "%d", &n); err == nil && n > 0 {
			return gifjob.LoopMode(n)
		}
		return gifjob.LoopForever
	}
}

// fitFloorWidth is the smallest width the fit loop will try before giving up.
const fitFloorWidth = 120

// fitMaxAttempts bounds how many re-encodes the fit loop will try to hit a
// size target, so a stubborn source cannot loop indefinitely.
const fitMaxAttempts = 8

// nextFitWidth returns the next smaller EVEN width to try when the output is
// over the size target. It shrinks by ~15% but always strictly decreases and
// never goes below floor (120).
func nextFitWidth(w int) int {
	n := int(float64(w) * 0.85)
	if n >= w {
		n = w - 2
	}
	if n%2 != 0 {
		n--
	}
	if n < fitFloorWidth {
		n = fitFloorWidth
	}
	return n
}

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

// videoConfig maps a VideoRequest to a gifjob.VideoConfig.
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

// imagesConfig maps an ImagesRequest to a gifjob.ImagesConfig. height is the
// precomputed aspect-aware canvas height.
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

// emitProgress safely emits a progress event, guarding against panics when
// no Wails runtime is available (e.g., in tests).
func (a *App) emitProgress(phase string, pct int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("gifly: progress emit recovered: %v", r)
		}
	}()
	a.mu.Lock()
	ready := a.ready
	a.mu.Unlock()
	if ready {
		runtime.EventsEmit(a.ctx, "convert:progress", ProgressEvent{Phase: phase, Percent: pct})
	}
}

// ConvertVideo performs a video-to-GIF conversion with progress tracking.
func (a *App) ConvertVideo(req VideoRequest) (ConvertResult, error) {
	tools, err := a.toolsOrErr()
	if err != nil {
		return ConvertResult{}, err
	}

	cfg := videoConfig(req)
	if err := cfg.Validate(); err != nil {
		return ConvertResult{}, err
	}

	out := a.resolveOut(req.OutDir, req.OutName, parseFormatReq(req.Format).Ext(), req.OnExist)
	if req.OutDir != "" {
		_ = os.MkdirAll(req.OutDir, 0o755)
	}

	// Create a cancellable context and store the cancel func under lock
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancelFn = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.cancelFn = nil
		a.mu.Unlock()
	}()

	// Compute total duration for progress calculation
	totalMS := cfg.EndMS - cfg.StartMS

	runner := ffmpeg.RunnerFunc(ffmpeg.Run)
	result, err := gifjob.RunVideo(ctx, tools, runner, cfg, out, func(p ffmpeg.Progress) {
		pct := percent(p.OutTimeMS, totalMS)
		a.emitProgress("encoding", pct)
	})
	if err != nil {
		return ConvertResult{}, err
	}

	// If a size target is set and the first encode is over it, re-encode at
	// shrinking widths until it fits, the floor width is hit, or attempts run out.
	if targetBytes := int64(req.TargetKB) * 1024; req.TargetKB > 0 && result.Bytes > targetBytes {
		width := cfg.Width
		for i := 0; i < fitMaxAttempts; i++ {
			if ctx.Err() != nil {
				return ConvertResult{}, ctx.Err()
			}
			newWidth := nextFitWidth(width)
			if newWidth == width {
				break
			}
			width = newWidth
			cfg.Width = width

			a.emitProgress("fitting", 0)
			result, err = gifjob.RunVideo(ctx, tools, runner, cfg, out, func(p ffmpeg.Progress) {
				pct := percent(p.OutTimeMS, totalMS)
				a.emitProgress("fitting", pct)
			})
			if err != nil {
				return ConvertResult{}, err
			}

			if result.Bytes <= targetBytes || width == fitFloorWidth {
				break
			}
		}
	}

	// Store the result as the preview
	a.SetPreview(result.Path)

	return ConvertResult{
		Path:   result.Path,
		Bytes:  result.Bytes,
		Width:  result.Width,
		Height: result.Height,
	}, nil
}

// ConvertImages performs an images-to-GIF conversion with progress tracking.
func (a *App) ConvertImages(req ImagesRequest) (ConvertResult, error) {
	tools, err := a.toolsOrErr()
	if err != nil {
		return ConvertResult{}, err
	}

	if len(req.Inputs) == 0 {
		return ConvertResult{}, fmt.Errorf("no input images")
	}

	// Probe the first image to compute canvas height
	frame, err := probe.Image(req.Inputs[0])
	if err != nil {
		return ConvertResult{}, err
	}

	aspect := parseAspectReq(req.Aspect)
	height := gifjob.OutputHeight(frame.Width, frame.Height, aspect, req.Width)
	cfg := imagesConfig(req, height)

	if err := cfg.Validate(); err != nil {
		return ConvertResult{}, err
	}

	out := a.resolveOut(req.OutDir, req.OutName, parseFormatReq(req.Format).Ext(), req.OnExist)
	if req.OutDir != "" {
		_ = os.MkdirAll(req.OutDir, 0o755)
	}

	// Create a cancellable context and store the cancel func under lock
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancelFn = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.cancelFn = nil
		a.mu.Unlock()
	}()

	runner := ffmpeg.RunnerFunc(ffmpeg.Run)
	imagesProgress := func(phase string) func(ffmpeg.Progress) {
		return func(p ffmpeg.Progress) {
			// For images, we emit frame-based progress
			// Estimate: assume total frames = len(inputs), estimate progress by frame count
			if len(req.Inputs) > 0 {
				pct := (p.Frame * 100) / len(req.Inputs)
				if pct > 100 {
					pct = 100
				}
				a.emitProgress(phase, pct)
			}
		}
	}

	result, err := gifjob.RunImages(ctx, tools, runner, cfg, out, imagesProgress("encoding"))
	if err != nil {
		return ConvertResult{}, err
	}

	// If a size target is set and the first encode is over it, re-encode at
	// shrinking widths (recomputing the canvas height each time) until it fits,
	// the floor width is hit, or attempts run out.
	if targetBytes := int64(req.TargetKB) * 1024; req.TargetKB > 0 && result.Bytes > targetBytes {
		width := cfg.Width
		for i := 0; i < fitMaxAttempts; i++ {
			if ctx.Err() != nil {
				return ConvertResult{}, ctx.Err()
			}
			newWidth := nextFitWidth(width)
			if newWidth == width {
				break
			}
			width = newWidth
			cfg.Width = width
			cfg.Height = gifjob.OutputHeight(frame.Width, frame.Height, aspect, width)

			a.emitProgress("fitting", 0)
			result, err = gifjob.RunImages(ctx, tools, runner, cfg, out, imagesProgress("fitting"))
			if err != nil {
				return ConvertResult{}, err
			}

			if result.Bytes <= targetBytes || width == fitFloorWidth {
				break
			}
		}
	}

	// Store the result as the preview
	a.SetPreview(result.Path)

	return ConvertResult{
		Path:   result.Path,
		Bytes:  result.Bytes,
		Width:  result.Width,
		Height: result.Height,
	}, nil
}

// Cancel aborts the in-flight conversion.
func (a *App) Cancel() {
	a.mu.Lock()
	cancelFn := a.cancelFn
	a.cancelFn = nil
	a.mu.Unlock()
	if cancelFn != nil {
		cancelFn()
	}
}
