package tui

import (
	"os"
	"path/filepath"
)

func isFirstRun() bool {
	cwd, _ := os.Getwd()
	toolingDir := filepath.Join(cwd, ".tooling")

	if _, err := os.Stat(toolingDir); os.IsNotExist(err) {
		return true
	}
	return false
}
