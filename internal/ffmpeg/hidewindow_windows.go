//go:build windows

package ffmpeg

import (
	"os/exec"
	"syscall"
)

// createNoWindow is the Windows CREATE_NO_WINDOW process-creation flag. A GUI
// app (gifly's Wails process has no console) that launches a console program
// like ffmpeg/ffprobe otherwise flashes a console window for the child; this
// flag runs the child with no console at all.
const createNoWindow = 0x08000000

// HideConsole makes cmd run without popping up a console window. It must be set
// before the command is started.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
