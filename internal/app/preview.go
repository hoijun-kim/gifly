package app

import (
	"fmt"
	"os"
	"os/exec"
)

// RevealOutput opens the folder containing the output GIF and selects the file.
// On Windows, it uses "explorer /select".
func (a *App) RevealOutput(path string) error {
	// Verify the file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("output file not found: %w", err)
	}

	// On Windows, use explorer /select to highlight the file
	return exec.Command("explorer", "/select,", path).Start()
}
