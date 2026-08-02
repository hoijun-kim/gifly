// Package app is the Wails binding layer: the App struct's exported methods are
// callable from the Svelte front end, and this is the only place the GUI reaches
// the engine (internal/gifjob, internal/probe, internal/ffmpeg).
package app

import (
	"context"
	"sync"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application object.
type App struct {
	ctx         context.Context
	mu          sync.Mutex
	cancelFn    context.CancelFunc
	previewPath string
}

// NewApp constructs the App.
func NewApp() *App { return &App{} }

// Startup captures the Wails runtime context (bound in main.go's OnStartup). It
// is exported because Wails' OnStartup requires an accessible method value.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

// toolsOrErr resolves the ffmpeg and ffprobe binaries, shared by the pickers.
func (a *App) toolsOrErr() (ffmpeg.Paths, error) {
	return ffmpeg.Tools()
}

// PickVideo opens a file dialog for video selection, then probes the picked file.
func (a *App) PickVideo() (VideoInfo, error) {
	tools, err := a.toolsOrErr()
	if err != nil {
		return VideoInfo{}, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select a video",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Video files",
				Pattern:     "*.mp4;*.webm;*.mov;*.avi;*.mkv;*.flv",
			},
		},
	})
	if err != nil {
		return VideoInfo{}, err
	}
	if path == "" {
		return VideoInfo{}, context.Canceled
	}
	return probeVideoInfo(tools.FFprobe, path)
}

// PickImages opens a multi-select dialog for image selection, then probes each file.
func (a *App) PickImages() ([]ImageInfo, error) {
	if _, err := a.toolsOrErr(); err != nil {
		return nil, err
	}
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select images",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Image files",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, context.Canceled
	}
	var images []ImageInfo
	for _, p := range paths {
		img, err := probeImageInfo(p)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}
