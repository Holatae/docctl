package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuild(t *testing.T) {
	cwd, id := setupTestOrg(t)

	err := CreateProtokoll(cwd, id, "styrelsen", "2025-05-05")
	if err != nil {
		t.Errorf("Failed to create protokoll: %s", err)
	}

	parentDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05")
	pathToSources := filepath.Join(parentDir, "källor")
	archiveDir := filepath.Join(parentDir, "arkiv")

	fmt.Println(pathToSources)

	err = DoBuild(pathToSources, false)
	if err != nil {
		t.Errorf("Failed to build %s: %s", id, err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "protokoll.pdf")); os.IsNotExist(err) {
		t.Errorf("Protokoll pdf does not exist")
	}
}
