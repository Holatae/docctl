package main

import (
	"fmt"
	"os"
	"os/exec"
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

func setupTestGPGEnv(t *testing.T) string {
	gpgHome := filepath.Join(t.TempDir(), ".gnupg")
	err := os.MkdirAll(gpgHome, 0700)
	if err != nil {
		t.Errorf("Failed to create GNUPG home dir: %s", err)
	}

	t.Setenv("GNUPGHOME", gpgHome)

	testKey := `-----BEGIN PGP PRIVATE KEY BLOCK-----

xVgEafTjLBYJKwYBBAHaRw8BAQdARpBJWVuRtM4bx7QAus5Z+QUyqFgRYgIA
iRCvleqK0DAAAP4kGdupuqCV4W6qH0L7kCD/LGveS+j6BZy0sg588NYk5RK0
zRV0ZXN0IDx0ZXN0QHRlc3QudGVzdD7CwBMEExYKAIUFgmn04ywDCwkHCRCh
W+Mt6yxRoEUUAAAAAAAcACBzYWx0QG5vdGF0aW9ucy5vcGVucGdwanMub3Jn
YpFL/5s4ci5aftzha2Rs1xc33Fe1Z9LcdBkurufjv5EFFQoIDgwEFgACAQIZ
AQKbAwIeARYhBLRbZ4GAGCGKxaaJ6qFb4y3rLFGgAAAiYAEAiB0PjFvN4e5i
FL4NRx9/ZWjdq9f0DmF4KPXDcn6r/wAA/jEpAaWuRRKRx1hlHFj3CrUHHG3W
22BcXdxtBrsgHN4Cx10EafTjLBIKKwYBBAGXVQEFAQEHQBS4iVhi4PWxWo+q
6S/l2mjQ6KCqIEhzBSNxSaQdazkiAwEIBwAA/30zovYSRJnF7kx2NVfT/CUn
vvHO/FAeS2uJ7mouJJpAEPzCvgQYFgoAcAWCafTjLAkQoVvjLessUaBFFAAA
AAAAHAAgc2FsdEBub3RhdGlvbnMub3BlbnBncGpzLm9yZ5rSuCup8xzc2UBI
4ReltyE4gTOXJMSLozoYtfTtSSs5ApsMFiEEtFtngYAYIYrFponqoVvjLess
UaAAAAp0AQCYDGghx6dQdPqMDL7l4BHAjChNXAY99mlRYiHQirzJpgD+P6Jb
M4JF1ODBz4lP2o38ZyS9wYC9N/sL/Y+Ihzhy9A0=
=duWz
-----END PGP PRIVATE KEY BLOCK-----`

	keyPath := filepath.Join(t.TempDir(), "testkey.asc")
	err = os.WriteFile(keyPath, []byte(testKey), 0600)
	if err != nil {
		t.Fatalf("Failed to create test key: %s", err)
	}

	cmd := exec.Command("gpg", "--batch", "--import", keyPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to import test key: %s, GPG säger\n%s", err, string(out))
	}

	return "test@test.test"
}

func TestSeal(t *testing.T) {
	keyId := setupTestGPGEnv(t)

	cwd, id := setupTestOrg(t)

	parentDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05")
	pathToSources := filepath.Join(parentDir, "källor")
	archiveDir := filepath.Join(parentDir, "arkiv")

	_ = createProtokoll(cwd, id, "styrelsen", "2025-05-05")

	_ = doBuild(pathToSources, false)

	if err := doSeal(pathToSources, keyId, false); err != nil {
		t.Fatalf("Failed to seal: %s", err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "ATTESTATION.md.sig")); os.IsNotExist(err) {
		t.Fatalf("Failed to find ATTESTATION FILE: %s", err)
	}
}
