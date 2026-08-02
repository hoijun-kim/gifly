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
	result, err := gifjob.RunImages(ctx, tools, runner, cfg, req.Out, func(p ffmpeg.Progress) {
		// For images, we emit frame-based progress
		// Estimate: assume total frames = len(inputs), estimate progress by frame count
		if len(req.Inputs) > 0 {
			pct := (p.Frame * 100) / len(req.Inputs)
			if pct > 100 {
				pct = 100
			}
			a.emitProgress("encoding", pct)
		}
	})
	if err != nil {
		return ConvertResult{}, err
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
