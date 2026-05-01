package app

import (
	"fmt"
	"os"
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
	var mdFile, baseName string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "ATTESTATION.md" {
			mdFile = filepath.Join(kallorDir, e.Name())
			baseName = strings.TrimSuffix(e.Name(), ".md")
			break
		}
	}

	if mdFile == "" {
		return fmt.Errorf("found no .md-file in \"källor\" ")
	}

	// 1. Kolla att mallen faktiskt finns!
	if stat, err := os.Stat(pandocTemplate); os.IsNotExist(err) || stat.Size() == 0 {
		return fmt.Errorf("❌ KATASTROF: Mallen finns inte på disken (eller är tom)!\nFörväntad sökväg: %s", pandocTemplate)
	}

	relMallPath, _ := filepath.Rel(arkivDir, orgMallPath)
	fmt.Printf("🔄 Bygger dokument för %s...\n", orgName)

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

	bilagorSrc := filepath.Join(kallorDir, "bilagor")
	bilagorDest := filepath.Join(arkivDir, "bilagor")
	if stat, err := os.Stat(bilagorSrc); err == nil && stat.IsDir() {
		err := copyDir(bilagorSrc, bilagorDest)
		if err != nil {
			return err
		}
	}

	fmt.Println("✅ Klart! Output: ", arkivDir)

	return nil
}
