package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestOrg(t *testing.T) (string, string) {
	t.Helper()

	cwd := t.TempDir()
	toolingDir := filepath.Join(cwd, ".tooling")
	err := firstTimeRun(toolingDir, cwd)
	if err != nil {
		t.Fatalf("Failed to initialize tooling: %s", err)
	}

	id := "TestOrg"
	name := "Test Organization"
	orgNumber := "111111-1111"

	_ = createOrgTemplate(cwd, id, name, orgNumber)

	return cwd, id

}

func TestInitialization(t *testing.T) {
	cwd := t.TempDir()

	toolingDir := filepath.Join(cwd, "tooling")

	err := firstTimeRun(toolingDir, cwd)

	//Verify it works
	if err != nil {
		t.Fatalf("Failed to initialize tooling: %s", err)
	}

	// Verify mallar
	mallarDir := filepath.Join(toolingDir, "mallar")
	if stat, err := os.Stat(mallarDir); os.IsNotExist(err) || !stat.IsDir() {
		t.Errorf("Mallar tooling directory %s does not exist", mallarDir)
	}

	// Verify .gitignore
	gitignorePath := filepath.Join(cwd, ".gitignore")
	gitignoreData, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Errorf("Failed to read .gitignore: %s", err)
	} else {
		if !strings.Contains(string(gitignoreData), "arkiv/*.pdf") {
			t.Errorf(".gitignore does not contain arkiv/*.pdf")
		}
	}

	// Verify embeds
	unpackedFiles, err := os.ReadDir(mallarDir)
	if err != nil {
		t.Errorf("Failed to read dir: %s", err)
	}
	if len(unpackedFiles) == 0 {
		t.Errorf("Found no unpacked files")
	}

}

func TestAddingOrganization(t *testing.T) {
	cwd, id := setupTestOrg(t)

	expectedFile := filepath.Join(cwd, id, "mallar", id+".typ")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Missing expected file %s", expectedFile)
	}
}

func TestBuild(t *testing.T) {
	cwd, id := setupTestOrg(t)

	err := createProtokoll(cwd, id, "styrelsen", "2025-05-05")
	if err != nil {
		t.Errorf("Failed to create protokoll: %s", err)
	}

	parentDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05")
	pathToSources := filepath.Join(parentDir, "källor")
	archiveDir := filepath.Join(parentDir, "arkiv")

	fmt.Println(pathToSources)

	err = doBuild(pathToSources, false)
	if err != nil {
		t.Errorf("Failed to build %s: %s", id, err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "protokoll.pdf")); os.IsNotExist(err) {
		t.Errorf("Protokoll pdf does not exist")
	}

}
