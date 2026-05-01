package app

import (
	"docctl/internal/assets"
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
	err := FirstTimeRun(toolingDir, cwd, assets.Files)
	if err != nil {
		t.Fatalf("Failed to initialize tooling: %s", err)
	}

	id := "TestOrg"
	name := "Test Organization"
	orgNumber := "111111-1111"

	_ = CreateOrgTemplate(cwd, id, name, orgNumber, assets.Files)

	return cwd, id

}

func setupTestGPGEnv(t *testing.T) string {
	t.Helper()

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
