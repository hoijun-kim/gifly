//go:build !windows

package ffmpeg

import "os/exec"

// HideConsole is a no-op off Windows: only Windows pops up a console window for
// a console child of a GUI process. gifly is Windows-only; this file exists so
// the package still builds on other platforms for tooling.
func HideConsole(cmd *exec.Cmd) {}
