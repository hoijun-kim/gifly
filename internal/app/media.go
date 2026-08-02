package app

import (
	"context"

	"github.com/hoijun-kim/gifly/internal/probe"
)

// VideoInfo is what the front end shows about a picked video.
type VideoInfo struct {
	Path       string
	DurationMS int64
	Width      int
	Height     int
	FPS        float64
}

// ImageInfo is what the front end shows about one picked image.
type ImageInfo struct {
	Path   string
	Width  int
	Height int
}

// probeImageInfo reads a still's dimensions via the stdlib probe.
func probeImageInfo(path string) (ImageInfo, error) {
	f, err := probe.Image(path)
	if err != nil {
		return ImageInfo{}, err
	}
	return ImageInfo{Path: path, Width: f.Width, Height: f.Height}, nil
}

// probeVideoInfo reads a video's shape via ffprobe.
func probeVideoInfo(ffprobe, path string) (VideoInfo, error) {
	m, err := probe.Video(context.Background(), ffprobe, path)
	if err != nil {
		return VideoInfo{}, err
	}
	return VideoInfo{Path: path, DurationMS: m.DurationMS, Width: m.Width, Height: m.Height, FPS: m.FPS}, nil
}
