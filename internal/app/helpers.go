package app

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"docctl/internal/assets"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// RunCmd runs a given command in the terminal
func RunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	// Fånga stderr (felmeddelanden) i en buffer istället för en komplicerad pipe
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// cmd.Run() utför kommandot och väntar tills det är klart
	err := cmd.Run()
	if err != nil {
		// Baka in stderr i det returnerade felet så att det syns tydligt i terminalen
		return fmt.Errorf("fel vid körning av %s: %w\nOutput: %s", name, err, stderr.String())
	}

	return nil
}

// ============================================
// HJÄLPFUNKTIONER
// ============================================

func HashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		srcFile, _ := os.Open(path)
		defer func(srcFile *os.File) {
			_ = srcFile.Close()
		}(srcFile)
		dstFile, _ := os.Create(dstPath)
		defer func(dstFile *os.File) {
			_ = dstFile.Close()
		}(dstFile)

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("could not copy %s to %s: %w", path, dstPath, err)
		}
		return nil
	})
}

// createOrgTemplateFiles hanterar endast filsystemet för en ny organisation.
// Liten begynnelsebokstav gör att den inte syns utåt (t.ex. för TUI:t).
func createOrgTemplateFiles(projRoot, orgID, orgName, orgNumber string) error {
	mallDir := filepath.Join(projRoot, orgID, "mallar")
	if err := os.MkdirAll(mallDir, 0o755); err != nil {
		return fmt.Errorf("kunde inte skapa mallmappen: %w", err)
	}

	outPath := filepath.Join(mallDir, orgID+".typ")

	// Skapa bara filen om den inte redan finns (så vi inte skriver över egna anpassningar)
	if _, err := os.Stat(outPath); err == nil {
		// Filen finns redan, vi är klara.
		return nil
	} else if !os.IsNotExist(err) {
		// Ett oväntat läsfel inträffade
		return fmt.Errorf("kunde inte kontrollera status på filen %s: %w", outPath, err)
	}

	// Läs in mallen från assets (vi behöver inte skicka in embed.FS som parameter längre)
	templateData, err := assets.Files.ReadFile("embeds/org_mall_template.typ")
	if err != nil {
		return fmt.Errorf("kunde inte läsa inbyggd mall 'org_mall_template.typ': %w", err)
	}

	// Hitta och ersätt våra platshållare!
	content := string(templateData)
	content = strings.ReplaceAll(content, "{{ORG_NAMN}}", orgName)
	content = strings.ReplaceAll(content, "{{ORG_NUMMER}}", orgNumber)

	// Spara den nya filen
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("kunde inte skriva mallfilen %s: %w", outPath, err)
	}

	return nil
}

// CreateOrganization är huvudfunktionen för att skapa en ny förening.
// Den skapar konfigurationen och genererar standardmallar.
func CreateOrganization(projRoot, orgID, orgName, orgNumber string) error {
	// 1. Ladda befintlig konfiguration
	cfg, err := LoadConfig(projRoot)
	if err != nil {
		return fmt.Errorf("kunde inte ladda konfiguration: %w", err)
	}

	// 2. Säkerställ att mappen (Foreningar) är initialiserad så vi slipper nil-pointer krascher
	if cfg.Foreningar == nil {
		cfg.Foreningar = make(map[string]Association)
	}

	// 3. Kolla om organisationen redan existerar
	if _, exists := cfg.Foreningar[orgID]; exists {
		return fmt.Errorf("organisationen '%s' existerar redan", orgID)
	}

	// 4. Skapa organisationen i konfigurationen med standardorgan
	cfg.Foreningar[orgID] = Association{
		Name:      orgName,
		OrgNummer: orgNumber,
		Body:      []string{"styrelsen", "årsmöte"}, // Lägg till dina standardorgan här
	}

	// 5. Spara konfigurationen först. Går detta fel, backar vi ur direkt.
	if err := SaveConfig(projRoot, cfg); err != nil {
		return fmt.Errorf("kunde inte spara konfigurationen för %s: %w", orgID, err)
	}

	// 6. Skapa mallfiler på disken
	if err := createOrgTemplateFiles(projRoot, orgID, orgName, orgNumber); err != nil {
		return fmt.Errorf("kunde inte skapa mallfiler för %s: %w", orgID, err)
	}

	return nil
}

func CreateZipArchive(srcDir string, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer func(zipFile *os.File) {
		_ = zipFile.Close()
	}(zipFile)

	archive := zip.NewWriter(zipFile)
	defer func(archive *zip.Writer) {
		_ = archive.Close()
	}(archive)

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(srcDir, path)
		header, _ := zip.FileInfoHeader(info)
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		_, err = io.Copy(writer, file)
		return err
	})
}

// FirstTimeRun creates all relevant folders and files
func FirstTimeRun(toolingDir string, cwd string, embeddedFiles embed.FS) (err error) {
	//fmt.Println("\n🚀 Initierar nytt arbetsutrymme...")
	//fmt.Printf("🕵️ AVSLÖJANDE: firstTimeRun sparar mallar i mappen: %s\n", filepath.Join(toolingDir, "mallar"))

	// 1. Skapa mappar
	mallarDir := filepath.Join(toolingDir, "mallar")
	if err := os.MkdirAll(mallarDir, 0o755); err != nil {
		return fmt.Errorf("could not create mallar directory: %v", err)
	}

	// 2. Automagisk uppackning: Läs alla filer som bäddades in i "embeds"-mappen!
	files, err := embeddedFiles.ReadDir("embeds")
	if err != nil {
		_ = fmt.Errorf("couldn't read embeds dir: %v", err)
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			// Läs filen inifrån binären
			contains, err := embeddedFiles.ReadFile("embeds/" + file.Name())
			if err == nil {
				// Skriv ut den till hårddisken
				path := filepath.Join(mallarDir, file.Name())
				if err := os.WriteFile(path, contains, 0o644); err != nil {
					return fmt.Errorf("cannot write file: %v", err)
				}
				//fmt.Printf("   -> Packade upp systemmall: %s\n", file.Name())
			}
		}
	}

	// 3. Skapa den perfekta .gitignore-filen
	gitignore := `# Dolda macOS/Windows-filer
.DS_Store
Thumbs.db

# Temporära filer från byggmotorn
*_temp.typ

# Privata signeringsnycklar – ska ALDRIG versionshanteras
.keys/

# Valfritt: Ignorera tunga leveransformat i Git
arkiv/*.pdf
arkiv/*.docx
`
	if err := os.WriteFile(filepath.Join(cwd, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return fmt.Errorf("could not write .gitignore: %v", err)
	}

	// 4. Initiera Git automatiskt (om git finns installerat)
	if _, err := exec.LookPath("git"); err == nil {
		fmt.Println("   -> Sätter upp versionshantering (git init)...")

		cmd := exec.Command("git", "init")
		cmd.Dir = cwd

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}

	}

	fmt.Println("✅ Arbetsutrymme skapat! Du är redo att köra.")
	return nil
}

// UpdateTemplates overwrites all embedded standard templates in the active workspace's .tooling/mallar/ directory
func UpdateTemplates(cwd string, embeddedFiles embed.FS) error {
	toolingDir := filepath.Join(cwd, ".tooling")
	mallarDir := filepath.Join(toolingDir, "mallar")

	if err := os.MkdirAll(mallarDir, 0o755); err != nil {
		return fmt.Errorf("could not create mallar directory: %w", err)
	}

	files, err := embeddedFiles.ReadDir("embeds")
	if err != nil {
		return fmt.Errorf("could not read embedded templates: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			contains, err := embeddedFiles.ReadFile("embeds/" + file.Name())
			if err != nil {
				return fmt.Errorf("could not read embedded file %s: %w", file.Name(), err)
			}
			path := filepath.Join(mallarDir, file.Name())
			if err := os.WriteFile(path, contains, 0o644); err != nil {
				return fmt.Errorf("could not write file %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

func CreateProtokoll(cwd string, orgId string, body string, date string) error {
	cfg, _ := LoadConfig(cwd)

	currentOrg := cfg.Foreningar[orgId]

	// Check if body exists otherwise create new

	if !slices.Contains(currentOrg.Body, body) {
		currentOrg.Body = append(currentOrg.Body, body)
		cfg.Foreningar[orgId] = currentOrg
		if err := SaveConfig(cwd, cfg); err != nil {
			return err
		}
	}

	if len(date) < 4 {
		return fmt.Errorf("date is too short")
	}
	year := date[:4]
	basePath := filepath.Join(cwd, orgId, "Årsakter", year, body, date, "källor")
	mdPath := filepath.Join(basePath, "protokoll.md")
	standardText := fmt.Sprintf("---\ntyp: protokoll\ntitle: Protokoll %s\ndatum: %s\ntid: 18:00\nplats: Föreningslokalen\nordforande: Namn Namnsson\nsekreterare: Namn Namnsson\njusterare:\n  - Justerare 1\n---\n\n## Mötets öppnande\n", body, date)

	err := os.MkdirAll(filepath.Join(basePath, "bilagor"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create directories")
	}
	f, err := os.Create(filepath.Join(mdPath))
	if err != nil {
		return fmt.Errorf("could not create markdown file")
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = f.WriteString(standardText)
	if err != nil {
		return fmt.Errorf("could not write to file")
	}

	return nil
}

func CreateGuidanceDocuments(cwd string, orgId string, subcategory string, docName string) error {
	base := filepath.Join(cwd, orgId, "Grundakter", "Styrdokument")
	return createGoverningDocument(base, subcategory, docName)
}

func CreateOtherGoverningDocuments(cwd string, orgId string, category string, docName string) error {
	base := filepath.Join(cwd, orgId, "Grundakter")
	return createGoverningDocument(base, category, docName)
}

func createGoverningDocument(baseDir string, category string, docName string) error {
	safeCategory := strings.ReplaceAll(category, " ", "-")
	safeDocName := strings.ReplaceAll(docName, " ", "-")
	ymlCategory := strings.ToLower(category)

	standardText := fmt.Sprintf("---\ntyp: %s\ntitle: %s\nversion: 1.0\nantagen: ÅÅÅÅ-MM-DD av Styrelsen\n---\n\n## 1. Syfte\nSyftet med detta dokument är...\n", ymlCategory, docName)
	mdPath := filepath.Join(baseDir, safeCategory, safeDocName, "källor", "document.md")

	if err := os.MkdirAll(filepath.Join(baseDir, safeCategory, safeDocName, "källor"), os.ModePerm); err != nil {
		return fmt.Errorf("kunde inte skapa mappar: %w", err)
	}

	f, err := os.Create(mdPath)
	if err != nil {
		return fmt.Errorf("kunde inte skapa markdown-fil: %w", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString(standardText); err != nil {
		return fmt.Errorf("kunde inte skriva till fil: %w", err)
	}

	return nil
}
