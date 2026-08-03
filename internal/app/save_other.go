//go:build !windows

package app

import "errors"

// copyFileToClipboard is unsupported off Windows. gifly ships Windows-only; this
// keeps the package building for tooling on other platforms.
func copyFileToClipboard(string) error {
	return errors.New("clipboard copy is only supported on Windows")
}
