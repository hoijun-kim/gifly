//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
)

// copyFileToClipboard puts the file on the Windows clipboard as a file drop
// (CF_HDROP) via PowerShell, so pasting into Explorer or a chat app yields the
// actual animated file. The path is passed through the environment to avoid any
// quoting or injection in the -Command string, and the console window is hidden.
func copyFileToClipboard(path string) error {
	const script = `Add-Type -AssemblyName System.Windows.Forms; ` +
		`$c = New-Object System.Collections.Specialized.StringCollection; ` +
		`[void]$c.Add($env:GIFLY_CLIP_PATH); ` +
		`[System.Windows.Forms.Clipboard]::SetFileDropList($c)`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-STA", "-Command", script)
	cmd.Env = append(os.Environ(), "GIFLY_CLIP_PATH="+path)
	ffmpeg.HideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clipboard copy failed: %w: %s", err, out)
	}
	return nil
}
