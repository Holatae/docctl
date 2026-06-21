package tui

import (
	"os"
	"path/filepath"
)

func isFirstRun(cwd string) bool {
	toolingDir := filepath.Join(cwd, ".tooling")

	if _, err := os.Stat(toolingDir); os.IsNotExist(err) {
		return true
	}
	return false
}
