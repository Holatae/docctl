package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
)

const testPrivateKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

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

func loadTestEntity(t *testing.T) *openpgp.Entity {
	t.Helper()
	entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(testPrivateKey))
	if err != nil || len(entities) == 0 {
		t.Fatalf("Failed to load test PGP entity: %v", err)
	}
	return entities[0]
}

func TestSeal(t *testing.T) {
	entity := loadTestEntity(t)

	cwd, id := setupTestOrg(t)

	parentDir := filepath.Join(cwd, id, "Årsakter", "2025", "styrelsen", "2025-05-05")
	pathToSources := filepath.Join(parentDir, "källor")
	archiveDir := filepath.Join(parentDir, "arkiv")

	_ = CreateProtokoll(cwd, id, "styrelsen", "2025-05-05")

	_ = DoBuild(pathToSources, false)

	if err := DoSeal(pathToSources, entity, false); err != nil {
		t.Fatalf("Failed to seal: %s", err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "ATTESTATION.md.sig")); os.IsNotExist(err) {
		t.Fatalf("Failed to find ATTESTATION FILE: %s", err)
	}
}
