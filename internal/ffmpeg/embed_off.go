//go:build !embedffmpeg

package ffmpeg

// embeddedDir returns "" in the normal build: no ffmpeg is embedded, so the
// caller falls through to a beside-exe "ffmpeg" folder or PATH. The
// self-contained build (-tags embedffmpeg) provides a real implementation.
func embeddedDir() (string, error) { return "", nil }
