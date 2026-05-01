package app

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
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

func CreateOrgTemplate(projRoot, orgID, orgNamn, orgNummer string, embeddedFiles embed.FS) error {
	cfg, err := LoadConfig(projRoot)
	if err != nil {
		return err
	}

	mallDir := filepath.Join(projRoot, orgID, "mallar")
	if err := os.MkdirAll(mallDir, 0o755); err != nil {
		return fmt.Errorf("could not make mallar directory: %w", err)
	}
	outPath := filepath.Join(mallDir, orgID+".typ")

	// Skapa bara filen om den inte redan finns (så vi inte skriver över egna anpassningar)
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		templateData, err := embeddedFiles.ReadFile("embeds/org_mall_template.typ")
		if err == nil {
			content := string(templateData)
			// Hitta och ersätt våra platshållare!
			content = strings.ReplaceAll(content, "{{ORG_NAMN}}", orgNamn)
			content = strings.ReplaceAll(content, "{{ORG_NUMMER}}", orgNummer)

			cfg.Foreningar[orgID] = Association{Name: orgNamn, OrgNummer: orgNummer, Body: []string{"styrelsen", "årsmöte"}}
			if err := SaveConfig(projRoot, cfg); err != nil {
				return err
			}

			if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("could not write org_mall_template.typ: %w", err)
			}
		} else {
			fmt.Printf("❌ Kunde inte hitta 'org_mall_template.typ' i embeds: %v\n", err)
		}
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

// firstTimeRun creates all relevant folders and files
func FirstTimeRun(toolingDir string, cwd string, embeddedFiles embed.FS) (err error) {
	fmt.Println("\n🚀 Initierar nytt arbetsutrymme...")
	fmt.Printf("🕵️ AVSLÖJANDE: firstTimeRun sparar mallar i mappen: %s\n", filepath.Join(toolingDir, "mallar"))

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
				fmt.Printf("   -> Packade upp systemmall: %s\n", file.Name())
			}
		}
	}

	// 3. Skapa den perfekta .gitignore-filen
	gitignore := `# Dolda macOS/Windows-filer
.DS_Store
Thumbs.db

# Temporära filer från byggmotorn
*_temp.typ

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

	// needs to be more safe
	safeCategory := strings.ReplaceAll(subcategory, " ", "-")
	safeDocName := strings.ReplaceAll(docName, " ", "-")
	ymlCategory := strings.ToLower(subcategory)

	standardText := fmt.Sprintf("---\ntyp: %s\ntitle: %s\nversion: 1.0\nantagen: ÅÅÅÅ-MM-DD av Styrelsen\n---\n\n## 1. Syfte\nSyftet med detta dokument är...\n", ymlCategory, docName)

	basePath := filepath.Join(cwd, orgId, "Grundakter", "Styrdokument")
	mdPath := filepath.Join(basePath, safeCategory, safeDocName, "källor", "document.md")

	if err := os.MkdirAll(filepath.Join(basePath, safeCategory, safeDocName, "källor"), os.ModePerm); err != nil {
		return fmt.Errorf("could create folders")
	}

	f, err := os.Create(mdPath)
	if err != nil {
		return fmt.Errorf("could not create markdown file")
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString(standardText); err != nil {
		return fmt.Errorf("could not write to file")
	}

	return nil
}
