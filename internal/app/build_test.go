package app

import (
	"docctl/internal/assets"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildFODT(t *testing.T) {
	if _, err := exec.LookPath("libreoffice"); err != nil {
		t.Skip("libreoffice saknas på PATH — hoppar över FODT-build-test")
	}

	cwd, id := setupTestOrg(t)

	källorDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05", "källor")
	arkivDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05", "arkiv")

	if err := CreateFODTDocument(källorDir, "protokoll", "__blank__", assets.Files); err != nil {
		t.Fatalf("CreateFODTDocument misslyckades: %v", err)
	}

	if err := DoBuild(källorDir, false); err != nil {
		t.Fatalf("DoBuild misslyckades: %v", err)
	}

	for _, f := range []string{"protokoll.pdf", "protokoll.html", "protokoll.fodt"} {
		if _, err := os.Stat(filepath.Join(arkivDir, f)); os.IsNotExist(err) {
			t.Errorf("Saknar förväntad fil i arkiv: %s", f)
		}
	}
}

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
