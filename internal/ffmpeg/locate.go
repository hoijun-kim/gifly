// Package ffmpeg is the only part of gifly that shells out to an external
// process. It locates the bundled ffmpeg and ffprobe binaries and runs them;
// nothing else in the codebase spawns a process.
package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Paths holds the resolved locations of the two binaries gifly drives.
type Paths struct {
	FFmpeg  string
	FFprobe string
}

// resolveDir returns the directory the bundled binaries live in. Preference,
// highest first: the GIFLY_FFMPEG_DIR override (used in development and tests),
// then an "ffmpeg" folder beside the running executable (how a release ships).
// It returns the first directory that actually contains ffmpeg.exe.
func resolveDir() (string, error) {
	var tried []string
	consider := func(dir string) (string, bool) {
		if dir == "" {
			return "", false
		}
		tried = append(tried, dir)
		if _, err := os.Stat(filepath.Join(dir, "ffmpeg.exe")); err == nil {
			return dir, true
		}
		return "", false
	}

	if dir, ok := consider(os.Getenv("GIFLY_FFMPEG_DIR")); ok {
		return dir, nil
	}
	if exe, err := os.Executable(); err == nil {
		if dir, ok := consider(filepath.Join(filepath.Dir(exe), "ffmpeg")); ok {
			return dir, nil
		}
	}
	return "", fmt.Errorf("ffmpeg not found; looked in %v", tried)
}

// Tools resolves ffmpeg and ffprobe. It first tries the bundled directory
// (resolveDir); if that fails it falls back to PATH, so a developer with ffmpeg
// installed can work without a bundle. A clear error names where it looked.
func Tools() (Paths, error) {
	if dir, err := resolveDir(); err == nil {
		return Paths{
			FFmpeg:  filepath.Join(dir, "ffmpeg.exe"),
			FFprobe: filepath.Join(dir, "ffprobe.exe"),
		}, nil
	}
	ff, errFF := exec.LookPath("ffmpeg")
	fp, errFP := exec.LookPath("ffprobe")
	if errFF == nil && errFP == nil {
		return Paths{FFmpeg: ff, FFprobe: fp}, nil
	}
	return Paths{}, fmt.Errorf("ffmpeg/ffprobe not found: not beside the exe, not in GIFLY_FFMPEG_DIR, not on PATH")
}
