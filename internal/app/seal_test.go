package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeal(t *testing.T) {
	keyId := setupTestGPGEnv(t)

	cwd, id := setupTestOrg(t)

	parentDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05")
	pathToSources := filepath.Join(parentDir, "källor")
	archiveDir := filepath.Join(parentDir, "arkiv")

	_ = CreateProtokoll(cwd, id, "styrelsen", "2025-05-05")

	_ = DoBuild(pathToSources, false)

	if err := DoSeal(pathToSources, keyId, false); err != nil {
		t.Fatalf("Failed to seal: %s", err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "ATTESTATION.md.sig")); os.IsNotExist(err) {
		t.Fatalf("Failed to find ATTESTATION FILE: %s", err)
	}
}
