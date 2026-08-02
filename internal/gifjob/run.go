package gifjob

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
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
