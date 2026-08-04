//go:build embedffmpeg

package ffmpeg

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// embeddedBinaries holds the ffmpeg/ffprobe executables compiled into the
// self-contained ("standalone") build. The packaging script copies the real
// binaries into internal/ffmpeg/bin/ before building with -tags embedffmpeg;
// the normal build excludes this file and ships nothing.
//
//go:embed bin/ffmpeg.exe bin/ffprobe.exe
var embeddedBinaries embed.FS

var (
	extractOnce sync.Once
	extractDir  string
	extractErr  error
)

// embeddedDir extracts the embedded binaries to a per-user cache directory
// (once per process) and returns that directory. The directory name carries a
// short hash of ffmpeg, so a differently-versioned build extracts alongside
// rather than clobbering, and an already-extracted copy is reused.
func embeddedDir() (string, error) {
	extractOnce.Do(extractEmbedded)
	return extractDir, extractErr
}

func extractEmbedded() {
	ff, err := embeddedBinaries.ReadFile("bin/ffmpeg.exe")
	if err != nil {
		extractErr = err
		return
	}
	fp, err := embeddedBinaries.ReadFile("bin/ffprobe.exe")
	if err != nil {
		extractErr = err
		return
	}
	sum := sha256.Sum256(ff)
	tag := hex.EncodeToString(sum[:6])
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "gifly", "ffmpeg-"+tag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		extractErr = err
		return
	}
	if err := writeOnce(filepath.Join(dir, "ffmpeg.exe"), ff); err != nil {
		extractErr = err
		return
	}
	if err := writeOnce(filepath.Join(dir, "ffprobe.exe"), fp); err != nil {
		extractErr = err
		return
	}
	extractDir = dir
}

// writeOnce writes data to path unless a same-size file is already there. It
// writes to a pid-suffixed temp file and renames, tolerating a concurrent
// instance that won the race (Windows rename does not overwrite).
func writeOnce(path string, data []byte) error {
	if fi, err := os.Stat(path); err == nil && fi.Size() == int64(len(data)) {
		return nil
	}
	tmp := path + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		if fi, statErr := os.Stat(path); statErr == nil && fi.Size() == int64(len(data)) {
			return nil
		}
		return err
	}
	return nil
}
