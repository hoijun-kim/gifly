package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
)

// writeExe creates a fake executable file so location logic (which only checks
// existence, never runs the file) has something to find.
func writeExe(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestToolsFindsBinariesInAGivenDir(t *testing.T) {
	dir := t.TempDir()
	writeExe(t, dir, "ffmpeg.exe")
	writeExe(t, dir, "ffprobe.exe")
	t.Setenv("GIFLY_FFMPEG_DIR", dir)

	got, err := Tools()
	if err != nil {
		t.Fatalf("Tools() = %v, want the binaries in %s", err, dir)
	}
	if got.FFmpeg != filepath.Join(dir, "ffmpeg.exe") {
		t.Errorf("FFmpeg = %q, want it under %s", got.FFmpeg, dir)
	}
	if got.FFprobe != filepath.Join(dir, "ffprobe.exe") {
		t.Errorf("FFprobe = %q, want it under %s", got.FFprobe, dir)
	}
}

func TestToolsErrorsWhenMissing(t *testing.T) {
	// An empty dir with the env pointed at it, and an empty PATH, so neither the
	// bundle nor a system ffmpeg can be found - the test does not depend on
	// whether the machine running it happens to have ffmpeg installed.
	t.Setenv("GIFLY_FFMPEG_DIR", t.TempDir())
	t.Setenv("PATH", "")
	if _, err := Tools(); err == nil {
		t.Fatal("Tools() with no binaries present returned nil error; a missing bundle must be reported")
	}
}
