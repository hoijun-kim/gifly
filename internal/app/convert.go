package app

import (
	"context"
	"fmt"
	"log"

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
	Loop    string
	Colors  int
	Dither  bool
	Out     string

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
	Loop    string
	Colors  int
	Dither  bool
	Out     string

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

// videoConfig maps a VideoRequest to a gifjob.VideoConfig.
func videoConfig(req VideoRequest) gifjob.VideoConfig {
	return gifjob.VideoConfig{
		Input:   req.Input,
		StartMS: req.StartMS,
		EndMS:   req.EndMS,
		FPS:     req.FPS,
		Width:   req.Width,
		Loop:    parseLoopMode(req.Loop),
		Quality: gifjob.Quality{
			MaxColors: req.Colors,
			Dither:    req.Dither,
		},
	}
}

// imagesConfig maps an ImagesRequest to a gifjob.ImagesConfig. It probes the
// first image to compute the canvas height.
func imagesConfig(req ImagesRequest, height int) gifjob.ImagesConfig {
	return gifjob.ImagesConfig{
		Inputs:  req.Inputs,
		FrameMS: req.FrameMS,
		Width:   req.Width,
		Height:  height,
		Loop:    parseLoopMode(req.Loop),
		Quality: gifjob.Quality{
			MaxColors: req.Colors,
			Dither:    req.Dither,
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
	result, err := gifjob.RunVideo(ctx, tools, runner, cfg, req.Out, func(p ffmpeg.Progress) {
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
			result, err = gifjob.RunVideo(ctx, tools, runner, cfg, req.Out, func(p ffmpeg.Progress) {
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

	height := gifjob.CanvasHeight(frame.Width, frame.Height, req.Width)
	cfg := imagesConfig(req, height)

	if err := cfg.Validate(); err != nil {
		return ConvertResult{}, err
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

	result, err := gifjob.RunImages(ctx, tools, runner, cfg, req.Out, imagesProgress("encoding"))
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
			cfg.Height = gifjob.CanvasHeight(frame.Width, frame.Height, width)

			a.emitProgress("fitting", 0)
			result, err = gifjob.RunImages(ctx, tools, runner, cfg, req.Out, imagesProgress("fitting"))
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
