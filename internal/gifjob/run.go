package gifjob

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// decodeConfig is image.DecodeConfig, named locally so statResult reads clearly.
var decodeConfig = image.DecodeConfig

// Runner runs one ffmpeg invocation. ffmpeg.RunnerFunc(ffmpeg.Run) is the real
// implementation; tests pass a fake.
type Runner interface {
	Run(ctx context.Context, bin string, args []string, onProgress func(ffmpeg.Progress)) error
}

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
