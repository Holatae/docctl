package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"docctl/internal/assets"
	"docctl/internal/keys"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// ExportPublicKey läser en befintlig nyckelfil och skriver ut den publika delen
// (utan privat nyckelmaterial) till outputPath.
func ExportPublicKey(keyFilePath, outputPath string) error {
	f, err := os.Open(keyFilePath)
	if err != nil {
		return fmt.Errorf("kunde inte öppna nyckelfil: %w", err)
	}
	defer func() { _ = f.Close() }()

	entities, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil || len(entities) == 0 {
		return fmt.Errorf("kunde inte läsa nyckel från %s", keyFilePath)
	}

	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("kunde inte skapa exportfil: %w", err)
	}
	defer func() { _ = out.Close() }()

	w, err := armor.Encode(out, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return fmt.Errorf("kunde inte skapa armor-writer: %w", err)
	}
	if err := entities[0].Serialize(w); err != nil {
		_ = w.Close()
		return fmt.Errorf("kunde inte serialisera publik nyckel: %w", err)
	}
	return w.Close()
}

// GenerateKeyContract skapar ett PDF-nyckelkontrakt för den angivna nyckeln.
// Kontraktet sparas till .keys/{OrgID}_{FP}_kontrakt.pdf och sökvägen returneras.
func GenerateKeyContract(projectRoot string, meta keys.KeyMeta, orgName, orgNumber, keyDirURL string) (string, error) {
	templateData, err := assets.Files.ReadFile("embeds/modul_nyckelkontrakt.typ")
	if err != nil {
		return "", fmt.Errorf("kunde inte läsa kontraktsmall: %w", err)
	}

	giltigTill := "Inget utgångsdatum"
	if meta.ExpiresAt != nil {
		giltigTill = meta.ExpiresAt.Format("2006-01-02")
	}

	content := string(templateData)
	content = strings.ReplaceAll(content, "{{ORG_NAMN}}", orgName)
	content = strings.ReplaceAll(content, "{{ORG_NUMMER}}", orgNumber)
	content = strings.ReplaceAll(content, "{{NAMN}}", meta.Name)
	content = strings.ReplaceAll(content, "{{ETIKETT}}", meta.Label)
	content = strings.ReplaceAll(content, "{{EMAIL}}", strings.ReplaceAll(meta.Email, "@", "\\@"))
	content = strings.ReplaceAll(content, "{{FINGERAVTRYCK}}", meta.ShortFP)
	content = strings.ReplaceAll(content, "{{SKAPAD}}", meta.Created.Format("2006-01-02"))
	content = strings.ReplaceAll(content, "{{GILTIG_TILL}}", giltigTill)
	content = strings.ReplaceAll(content, "{{NYCKEL_KATALOG}}", keyDirURL)

	// Skriv ifylld mall bredvid karna.typ så Typst-importen fungerar
	mallarDir := filepath.Join(projectRoot, ".tooling", "mallar")
	tempPath := filepath.Join(mallarDir, "kontrakt_temp.typ")
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("kunde inte skriva tillfällig kontraktsfil: %w", err)
	}
	defer func() { _ = os.Remove(tempPath) }()

	outputPath := filepath.Join(projectRoot, ".keys",
		fmt.Sprintf("%s_%s_kontrakt.pdf", meta.OrgID, meta.ShortFP))

	if err := RunCmd("typst", "compile", tempPath, outputPath); err != nil {
		return "", fmt.Errorf("kunde inte kompilera nyckelkontrakt: %w", err)
	}

	return outputPath, nil
}
