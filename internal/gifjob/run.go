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

	passes := VideoArgs(c, palettePath, outPath)
	last := len(passes) - 1
	for i, args := range passes {
		var cb func(ffmpeg.Progress)
		if i == last {
			cb = onProgress
		}
		if err := r.Run(ctx, tools.FFmpeg, args, cb); err != nil {
			if i == last {
				os.Remove(outPath)
				return Result{}, fmt.Errorf("encode pass: %w", err)
			}
			return Result{}, fmt.Errorf("palette pass: %w", err)
		}
	}
	return statResult(outPath)
}

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
	passes := ImagesArgs(c, listPath, palettePath, outPath)
	last := len(passes) - 1
	for i, args := range passes {
		var cb func(ffmpeg.Progress)
		if i == last {
			cb = onProgress
		}
		if err := r.Run(ctx, tools.FFmpeg, args, cb); err != nil {
			if i == last {
				os.Remove(outPath)
				return Result{}, fmt.Errorf("encode pass: %w", err)
			}
			return Result{}, fmt.Errorf("palette pass: %w", err)
		}
	}
	return statResult(outPath)
}
