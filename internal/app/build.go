package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// doBuild generates document in multiple formats (PDF, HTML, DOCX)
// from source files in pathToSources.
// force: forces the rebuilding of the documents even if the archive is Sealed
func DoBuild(pathToSources string, force bool) (err error) {
	kallorDir, _ := filepath.Abs(pathToSources)
	motesDir := filepath.Dir(kallorDir)
	arkivDir := filepath.Join(motesDir, "arkiv")
	sigPath := filepath.Join(arkivDir, "ATTESTATION.md.sig")

	if _, err := os.Stat(sigPath); err == nil {
		if !force {
			return fmt.Errorf("❌ AVSLAGET: Arkivet är förseglat! Använd --force för att skriva över")
		}

		fmt.Println("⚠️ FORCE aktivt: Raderar gamla manifest och signaturer...")

		// Scarryy. Please don't do same as steam
		if err := os.RemoveAll(arkivDir); err != nil {
			return fmt.Errorf("could not remove arkiv directory: %w", err)
		}
	}

	if err := os.MkdirAll(arkivDir, 0o755); err != nil {
		return fmt.Errorf("could not create arkiv directory: %w", err)
	}

	// Kopiera bilagor innan bygget så att typst/pandoc hittar dem via relativa sökvägar
	bilagorSrc := filepath.Join(kallorDir, "bilagor")
	bilagorDest := filepath.Join(arkivDir, "bilagor")
	if stat, err := os.Stat(bilagorSrc); err == nil && stat.IsDir() {
		if err := copyDir(bilagorSrc, bilagorDest); err != nil {
			return fmt.Errorf("kunde inte kopiera bilagor: %w", err)
		}
	}

	curr := kallorDir
	var projRoot string
	for {
		if _, err := os.Stat(filepath.Join(curr, ".tooling")); !os.IsNotExist(err) {
			projRoot = curr
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return fmt.Errorf("❌ Hittade inte rot-mappen (.tooling)!")
		}
		curr = parent
	}

	// Räkna ut Orgname
	relPath, err := filepath.Rel(projRoot, kallorDir)
	if err != nil {
		return fmt.Errorf("couldn't get relative path of kallor: %v", err)
	}

	orgName := strings.Split(relPath, string(filepath.Separator))[0]

	orgMallPath := filepath.Join(projRoot, orgName, "mallar", orgName+".typ")
	pandocTemplate := filepath.Join(projRoot, ".tooling", "mallar", "pandoc_klister.typ")

	entries, _ := os.ReadDir(kallorDir)
	var mdFile, fodtFile, baseName string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "ATTESTATION.md" {
			mdFile = filepath.Join(kallorDir, e.Name())
			baseName = strings.TrimSuffix(e.Name(), ".md")
			break
		}
		if strings.HasSuffix(e.Name(), ".fodt") {
			fodtFile = filepath.Join(kallorDir, e.Name())
			baseName = strings.TrimSuffix(e.Name(), ".fodt")
		}
	}

	if mdFile == "" && fodtFile == "" {
		return fmt.Errorf("hittade varken .md- eller .fodt-fil i \"källor\"")
	}

	fmt.Printf("🔄 Bygger dokument för %s...\n", orgName)

	if fodtFile != "" {
		return buildFODT(fodtFile, baseName, arkivDir)
	}

	// Markdown → Typst → PDF/A + HTML + DOCX
	if stat, err := os.Stat(pandocTemplate); os.IsNotExist(err) || stat.Size() == 0 {
		return fmt.Errorf("❌ KATASTROF: Mallen finns inte på disken (eller är tom)!\nFörväntad sökväg: %s", pandocTemplate)
	}

	relMallPath, _ := filepath.Rel(arkivDir, orgMallPath)

	tempTypst := filepath.Join(arkivDir, baseName+"_temp.typ")
	if err := RunCmd("pandoc", mdFile, "-t", "typst", "-o", tempTypst, "--template", pandocTemplate, "-V", "org_mall="+relMallPath); err != nil {
		return fmt.Errorf("pandoc misslyckades %w", err)
	}

	pdfOut := filepath.Join(arkivDir, baseName+".pdf")
	if err := RunCmd("typst", "compile", "--root", projRoot, "--pdf-standard", "a-2b", tempTypst, pdfOut); err != nil {
		return fmt.Errorf("typst failed to compile: %w", err)
	}
	_ = os.Remove(tempTypst)

	htmlOut := filepath.Join(arkivDir, baseName+".html")
	if err := RunCmd("pandoc", mdFile, "-o", htmlOut, "--standalone"); err != nil {
		return fmt.Errorf("pandoc failed to compile (HTML): %w", err)
	}

	docxOut := filepath.Join(arkivDir, baseName+".docx")
	if err := RunCmd("pandoc", mdFile, "-o", docxOut); err != nil {
		return fmt.Errorf("pandoc failed to compile (DOCX): %w", err)
	}

	fmt.Println("✅ Klart! Output: ", arkivDir)
	return nil
}

// buildFODT converts a .fodt source file to PDF/A-2b (via LibreOffice) and HTML (via pandoc),
// then copies the .fodt itself into arkivDir.
func buildFODT(fodtFile, baseName, arkivDir string) error {
	if _, err := exec.LookPath("libreoffice"); err != nil {
		return fmt.Errorf("❌ LibreOffice hittades inte på PATH — krävs för att bygga FODT-dokument")
	}

	// PDF/A-2b via LibreOffice
	if err := RunCmd("libreoffice", "--headless",
		"--convert-to", "pdf:writer_pdf_Export:SelectPdfVersion=2,EmbedStandardFonts=true",
		"--outdir", arkivDir,
		fodtFile,
	); err != nil {
		return fmt.Errorf("libreoffice misslyckades med PDF-export: %w", err)
	}

	// Verify that LibreOffice produced the expected filename (it uses the source basename)
	// LibreOffice names the output after the source file, so rename if needed
	expectedPDF := filepath.Join(arkivDir, baseName+".pdf")
	if _, err := os.Stat(expectedPDF); os.IsNotExist(err) {
		return fmt.Errorf("❌ LibreOffice producerade ingen PDF på förväntad sökväg: %s", expectedPDF)
	}

	// HTML via pandoc: pandoc's ODF-läsare kräver zip-baserad .odt, inte flat XML.
	// Konvertera FODT → temp ODT med LibreOffice, kör sedan pandoc på ODT-filen.
	tmpDir, err := os.MkdirTemp("", "docctl-fodt-*")
	if err != nil {
		return fmt.Errorf("kunde inte skapa temporär katalog: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := RunCmd("libreoffice", "--headless",
		"--convert-to", "odt",
		"--outdir", tmpDir,
		fodtFile,
	); err != nil {
		return fmt.Errorf("libreoffice misslyckades med ODT-konvertering: %w", err)
	}

	tmpODT := filepath.Join(tmpDir, baseName+".odt")
	htmlOut := filepath.Join(arkivDir, baseName+".html")
	if err := RunCmd("pandoc", tmpODT, "-o", htmlOut, "--standalone"); err != nil {
		return fmt.Errorf("pandoc misslyckades med HTML-export: %w", err)
	}

	// Kopiera .fodt till arkiv
	fodtDest := filepath.Join(arkivDir, baseName+".fodt")
	src, err := os.Open(fodtFile)
	if err != nil {
		return fmt.Errorf("kunde inte öppna källfil: %w", err)
	}
	defer func() { _ = src.Close() }()
	dst, err := os.Create(fodtDest)
	if err != nil {
		return fmt.Errorf("kunde inte skapa arkivfil: %w", err)
	}
	defer func() { _ = dst.Close() }()
	if _, err := copyFile(src, dst); err != nil {
		return fmt.Errorf("kunde inte kopiera .fodt till arkiv: %w", err)
	}

	fmt.Println("✅ Klart! Output: ", arkivDir)
	return nil
}

func copyFile(src, dst *os.File) (int64, error) {
	return dst.ReadFrom(src)
}
