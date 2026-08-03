package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// invalidChars are the characters Windows forbids in a file name; they are
// stripped from a user-supplied output name.
var invalidChars = strings.NewReplacer(
	`\`, "", "/", "", ":", "", "*", "", "?", "", `"`, "", "<", "", ">", "", "|", "",
)

// sanitizeName strips characters that cannot appear in a Windows file name,
// trims surrounding whitespace and trailing dots/spaces, and falls back to
// "gifly" when nothing usable is left.
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = invalidChars.Replace(name)
	name = strings.TrimRight(name, ". ")
	if name == "" {
		name = "gifly"
	}
	return name
}

// resolveOutPath builds the final output path in dir for the sanitized base
// name plus ext (which includes the dot), under an on-exist policy:
//   - "overwrite": dir/name.ext, replacing any existing file.
//   - "timestamp": dir/name-YYYYMMDD-HHMMSS.ext.
//   - "number" (and the default): dir/name.ext, or the first free
//     "dir/name (N).ext" when that is taken.
//
// exists reports whether a candidate path is already taken; now stamps the
// timestamp policy. resolveOutPath creates nothing.
func resolveOutPath(dir, name, ext, policy string, exists func(string) bool, now time.Time) string {
	name = sanitizeName(name)
	switch policy {
	case "timestamp":
		return filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, now.Format("20060102-150405"), ext))
	case "overwrite":
		return filepath.Join(dir, name+ext)
	default: // "number"
		p := filepath.Join(dir, name+ext)
		if !exists(p) {
			return p
		}
		for i := 1; ; i++ {
			c := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))
			if !exists(c) {
				return c
			}
		}
	}
}

// resolveOut is resolveOutPath wired to the real filesystem and clock.
func (a *App) resolveOut(dir, name, ext, policy string) string {
	return resolveOutPath(dir, name, ext, policy, func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}, time.Now())
}

// PickFolder opens a directory-chooser dialog and returns the chosen folder.
func (a *App) PickFolder() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose an output folder"})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", context.Canceled
	}
	return dir, nil
}

// CopyOutput copies the finished file itself to the clipboard (as a file drop),
// so it can be pasted straight into a chat app. Windows only.
func (a *App) CopyOutput(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("output file not found: %w", err)
	}
	return copyFileToClipboard(path)
}
