// Package probe reads a source's shape - a video's duration, dimensions and
// frame rate via ffprobe, or a still image's dimensions via the standard
// library - to populate the conversion controls (trim range, default width).
package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// Media is what gifly needs to know about a video before converting it.
type Media struct {
	DurationMS int64
	Width      int
	Height     int
	FPS        float64
}

// Frame is a still image's pixel size.
type Frame struct {
	Width  int
	Height int
}

type probeDoc struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		AvgFrame  string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// parseProbeJSON turns ffprobe's -print_format json output into a Media. It uses
// the first video stream for size and frame rate and format.duration for length.
func parseProbeJSON(b []byte) (Media, error) {
	var d probeDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return Media{}, fmt.Errorf("probe: bad ffprobe json: %w", err)
	}
	var m Media
	for _, s := range d.Streams {
		if s.CodecType == "video" {
			m.Width, m.Height = s.Width, s.Height
			m.FPS = parseRate(s.AvgFrame)
			break
		}
	}
	if m.Width == 0 {
		return Media{}, fmt.Errorf("probe: no video stream found")
	}
	if secs, err := strconv.ParseFloat(d.Format.Duration, 64); err == nil {
		m.DurationMS = int64(secs*1000 + 0.5)
	}
	return m, nil
}

// parseRate turns ffprobe's "num/den" frame-rate string into fps, tolerating a
// zero denominator (returns 0).
func parseRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	n, err1 := strconv.ParseFloat(num, 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d == 0 {
		return 0
	}
	return n / d
}

// Video runs ffprobe on path and parses the result.
func Video(ctx context.Context, ffprobe, path string) (Media, error) {
	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "error",
		"-print_format", "json",
		"-show_format", "-show_streams",
		path,
	)
	ffmpeg.HideConsole(cmd) // no console window flash when the GUI probes
	out, err := cmd.Output()
	if err != nil {
		return Media{}, fmt.Errorf("probe: ffprobe on %q failed: %w", path, err)
	}
	return parseProbeJSON(out)
}

// Image decodes just the header of a still to read its dimensions, which also
// validates that the file is a real PNG, JPEG or GIF.
func Image(path string) (Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return Frame{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Frame{}, fmt.Errorf("probe: %q is not a readable image: %w", path, err)
	}
	return Frame{Width: cfg.Width, Height: cfg.Height}, nil
}
