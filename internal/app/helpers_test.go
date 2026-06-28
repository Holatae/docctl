package app

import (
	"docctl/internal/assets"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestOrg(t *testing.T) (string, string) {
	t.Helper()

	cwd := t.TempDir()
	toolingDir := filepath.Join(cwd, ".tooling")
	err := FirstTimeRun(toolingDir, cwd, assets.Files)
	if err != nil {
		t.Fatalf("Failed to initialize tooling: %s", err)
	}

	id := "TestOrg"
	name := "Test Organization"
	orgNumber := "111111-1111"

	_ = CreateOrganization(cwd, id, name, orgNumber)

	return cwd, id

}


func TestInitialization(t *testing.T) {
	cwd := t.TempDir()

	toolingDir := filepath.Join(cwd, "tooling")

	err := FirstTimeRun(toolingDir, cwd, assets.Files)

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
		if !strings.Contains(string(gitignoreData), ".keys/") {
			t.Errorf(".gitignore does not contain .keys/")
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

func TestUpdateTemplates(t *testing.T) {
	cwd := t.TempDir()
	toolingDir := filepath.Join(cwd, ".tooling")

	// First initialization
	err := FirstTimeRun(toolingDir, cwd, assets.Files)
	if err != nil {
		t.Fatalf("Failed to initialize tooling: %s", err)
	}

	mallarDir := filepath.Join(toolingDir, "mallar")
	testFilePath := filepath.Join(mallarDir, "router.typ")

	// Modify one of the templates in the destination to see if it gets overwritten
	originalContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read router.typ: %s", err)
	}

	err = os.WriteFile(testFilePath, []byte("MODIFIED_CONTENT_FOR_TESTING"), 0644)
	if err != nil {
		t.Fatalf("Failed to write modified router.typ: %s", err)
	}

	// Now run UpdateTemplates
	err = UpdateTemplates(cwd, assets.Files)
	if err != nil {
		t.Fatalf("Failed to update templates: %s", err)
	}

	// Read content again, it should match the original content
	updatedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read updated router.typ: %s", err)
	}

	if string(updatedContent) != string(originalContent) {
		t.Errorf("Expected template to be restored to original content, but got: %s", string(updatedContent))
	}
}

