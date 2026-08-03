// Package gifjob turns a typed GIF conversion request into the exact ffmpeg
// argument lists and runs the two-pass palettegen/paletteuse pipeline. The
// argument builders (args.go) are pure and are the load-bearing tests: a wrong
// flag is the likeliest defect and the hardest to see by eye.
package gifjob

import (
	"fmt"
	"math"
)

// LoopMode is the literal ffmpeg -loop value for a GIF: 0 loops forever, -1
// plays once, and any positive n loops n times.
type LoopMode int

const (
	LoopForever LoopMode = 0
	LoopOnce    LoopMode = -1
)

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

// VideoConfig is a video-to-GIF request. StartMS/EndMS is the trim window;
// Width is the output width in pixels (height follows, keeping aspect).
type VideoConfig struct {
	Input   string
	StartMS int64
	EndMS   int64
	FPS     int
	Width   int
	Loop    LoopMode
	Quality Quality

	// SrcWidth/SrcHeight is the source video's pixel size, used to compute the
	// WebP/APNG output height.
	SrcWidth  int
	SrcHeight int
	// Aspect is an optional center-crop applied before scaling.
	Aspect Aspect
	// Speed is a playback rate multiplier; 1.0 is normal and <=0 is treated as
	// 1.0.
	Speed     float64
	Reverse   bool
	Boomerang bool
	// Format is the output container; "" is treated as FormatGIF.
	Format Format
}

// ImagesConfig is an images-to-GIF request. FrameMS is how long each frame
// shows; Inputs is the ordered list of image paths.
type ImagesConfig struct {
	Inputs  []string
	FrameMS int
	Width   int
	Height  int
	Loop    LoopMode
	Quality Quality

	// Speed is a playback rate multiplier; 1.0 is normal and <=0 is treated as
	// 1.0.
	Speed     float64
	Reverse   bool
	Boomerang bool
	// Format is the output container; "" is treated as FormatGIF.
	Format Format
}

// Result describes a finished GIF.
type Result struct {
	Path   string
	Bytes  int64
	Width  int
	Height int
}

func validateShared(width int, loop LoopMode, q Quality) error {
	if width <= 0 {
		return fmt.Errorf("output width must be positive, got %d", width)
	}
	if loop < LoopOnce {
		return fmt.Errorf("loop must be -1 (once), 0 (forever) or positive, got %d", int(loop))
	}
	if q.MaxColors < 2 || q.MaxColors > 256 {
		return fmt.Errorf("palette colors must be 2..256, got %d", q.MaxColors)
	}
	return nil
}

// Validate refuses a video config that cannot produce a GIF.
func (c VideoConfig) Validate() error {
	if c.Input == "" {
		return fmt.Errorf("no input video")
	}
	if c.EndMS <= c.StartMS {
		return fmt.Errorf("trim end (%d ms) must be after start (%d ms)", c.EndMS, c.StartMS)
	}
	if c.FPS <= 0 {
		return fmt.Errorf("fps must be positive, got %d", c.FPS)
	}
	if c.Format != "" && !c.Format.Valid() {
		return fmt.Errorf("unknown format %q", string(c.Format))
	}
	if c.Speed < 0 || c.Speed > 4 {
		return fmt.Errorf("speed must be 0..4, got %v", c.Speed)
	}
	if c.Aspect != "" && !c.Aspect.Valid() {
		return fmt.Errorf("unknown aspect %q", string(c.Aspect))
	}
	return validateShared(c.Width, c.Loop, c.Quality)
}

// Validate refuses an images config that cannot produce a GIF.
func (c ImagesConfig) Validate() error {
	if len(c.Inputs) == 0 {
		return fmt.Errorf("no input images")
	}
	if c.FrameMS <= 0 {
		return fmt.Errorf("frame duration must be positive, got %d ms", c.FrameMS)
	}
	if c.Height <= 0 {
		return fmt.Errorf("canvas height must be positive, got %d", c.Height)
	}
	if c.Format != "" && !c.Format.Valid() {
		return fmt.Errorf("unknown format %q", string(c.Format))
	}
	if c.Speed < 0 || c.Speed > 4 {
		return fmt.Errorf("speed must be 0..4, got %v", c.Speed)
	}
	return validateShared(c.Width, c.Loop, c.Quality)
}

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
