// Package gifjob turns a typed GIF conversion request into the exact ffmpeg
// argument lists and runs the two-pass palettegen/paletteuse pipeline. The
// argument builders (args.go) are pure and are the load-bearing tests: a wrong
// flag is the likeliest defect and the hardest to see by eye.
package gifjob

import "fmt"

// LoopMode is the literal ffmpeg -loop value for a GIF: 0 loops forever, -1
// plays once, and any positive n loops n times.
type LoopMode int

const (
	LoopForever LoopMode = 0
	LoopOnce    LoopMode = -1
)

// Quality controls the palette. MaxColors is 2..256; Dither on uses ffmpeg's
// sierra2_4a, off uses none (smaller, banding on gradients).
type Quality struct {
	MaxColors int
	Dither    bool
}

// DefaultQuality is a full 256-color dithered palette.
func DefaultQuality() Quality { return Quality{MaxColors: 256, Dither: true} }

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
}

// ImagesConfig is an images-to-GIF request. FrameMS is how long each frame
// shows; Inputs is the ordered list of image paths.
type ImagesConfig struct {
	Inputs  []string
	FrameMS int
	Width   int
	Loop    LoopMode
	Quality Quality
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
	return validateShared(c.Width, c.Loop, c.Quality)
}
