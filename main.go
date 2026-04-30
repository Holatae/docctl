package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea" // <--- NY!
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed embeds

var embeddedFiles embed.FS

// ============================================
// DATAMODELLER FÖR CONFIG
// ============================================
type Config struct {
	Foreningar map[string]Forening `yaml:"foreningar"`
	Settings   Settings            `yaml:"settings"`
}

type Forening struct {
	Namn      string   `yaml:"namn"`
	OrgNummer string   `yaml:"org_nummer"`
	Organ     []string `yaml:"organ"`
}

type Settings struct {
	CreateZIP bool `yaml:"create_zip"`
}

func loadConfig(projRoot string) Config {
	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	var cfg Config

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg = Config{
			Foreningar: map[string]Forening{},
			Settings:   Settings{CreateZIP: true},
		}
		saveConfig(projRoot, cfg)
		return cfg
	}
	yaml.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(projRoot string, cfg Config) {
	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	os.MkdirAll(filepath.Dir(configPath), 0o755)
	data, _ := yaml.Marshal(&cfg)
	os.WriteFile(configPath, data, 0o644)
}

// ============================================
// HJÄLPFUNKTIONER
// ============================================
func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	slurp, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		fmt.Printf("❌ Fel vid körning av %s:\n%s\n", name, string(slurp))
		os.Exit(1)
	}
}

func hashFile(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}

func createOrgTemplate(projRoot, orgID, orgNamn, orgNummer string) {
	mallDir := filepath.Join(projRoot, orgID, "mallar")
	os.MkdirAll(mallDir, 0o755)
	outPath := filepath.Join(mallDir, orgID+".typ")

	// Skapa bara filen om den inte redan finns (så vi inte skriver över egna anpassningar)
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		templateData, err := embeddedFiles.ReadFile("embeds/org_mall_template.typ")
		if err == nil {
			content := string(templateData)
			// Hitta och ersätt våra platshållare!
			content = strings.ReplaceAll(content, "{{ORG_NAMN}}", orgNamn)
			content = strings.ReplaceAll(content, "{{ORG_NUMMER}}", orgNummer)

			os.WriteFile(outPath, []byte(content), 0o644)
		} else {
			fmt.Printf("❌ Kunde inte hitta 'org_mall_template.typ' i embeds: %v\n", err)
		}
	}
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
		defer srcFile.Close()
		dstFile, _ := os.Create(dstPath)
		defer dstFile.Close()
		io.Copy(dstFile, srcFile)
		return nil
	})
}

func createZipArchive(srcDir string, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

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
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// ------------------------------------------
// PAUSFUNKTIONEN (För att du ska hinna se loggarna)
// ------------------------------------------
func pausePrompt() {
	fmt.Println("\n[ Tryck på Enter för att återgå till menyn... ]")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ============================================
// FIRST RUN / ONBOARDING
// ============================================

func checkFirstRun() {
	cwd, _ := os.Getwd()
	toolingDir := filepath.Join(cwd, ".tooling")

	// Om .tooling redan finns, är allt frid och fröjd. Avbryt och starta programmet.
	if _, err := os.Stat(toolingDir); !os.IsNotExist(err) {
		return
	}

	// Mappen är tom! Vi frågar användaren om de vill bygga ett arkiv.
	var confirm bool
	err := runForm(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Tom mapp upptäckt!").
				Description("Det verkar inte finnas något arkiv här.\nVill du initiera ett nytt Föreningsarkiv i denna mapp?").
				Affirmative("Ja, bygg arkivet!").
				Negative("Nej, avbryt").
				Value(&confirm),
		),
	))

	if err != nil || !confirm {
		fmt.Println("❌ Avbröt. Kör docctl i en befintlig arkivmapp.")
		os.Exit(0)
	}

	fmt.Println("\n🚀 Initierar nytt arbetsutrymme...")

	// 1. Skapa mappar
	mallarDir := filepath.Join(toolingDir, "mallar")
	os.MkdirAll(mallarDir, 0o755)

	// 2. Automagisk uppackning: Läs alla filer som bäddades in i "embeds"-mappen!
	filer, err := embeddedFiles.ReadDir("embeds")
	if err != nil {
		fmt.Println("❌ LARM: Kunde inte läsa inbäddade filer. Finns mappen 'embeds' i Go-koden?")
	} else {
		for _, fil := range filer {
			if !fil.IsDir() {
				// Läs filen inifrån binären
				innehall, errLäs := embeddedFiles.ReadFile("embeds/" + fil.Name())
				if errLäs == nil {
					// Skriv ut den till hårddisken
					utSökväg := filepath.Join(mallarDir, fil.Name())
					os.WriteFile(utSökväg, innehall, 0o644)
					fmt.Printf("   -> Packade upp systemmall: %s\n", fil.Name())
				}
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
# arkiv/*.pdf
# arkiv/*.docx
`
	os.WriteFile(filepath.Join(cwd, ".gitignore"), []byte(gitignore), 0o644)

	// 4. Initiera Git automatiskt (om git finns installerat)
	if _, err := exec.LookPath("git"); err == nil {
		fmt.Println("   -> Sätter upp versionshantering (git init)...")
		exec.Command("git", "init").Run()
	}

	fmt.Println("✅ Arbetsutrymme skapat! Du är redo att köra.")
	time.Sleep(2 * time.Second) // Pausa i 2 sekunder så användaren hinner läsa innan TUI:t tar över skärmen
}

// ============================================
// LOGIK MOTOR (doBuild & doSeal)
// ============================================

func doBuild(orgName string, kallorPath string, force bool) {
	kallorDir, _ := filepath.Abs(kallorPath)
	motesDir := filepath.Dir(kallorDir)
	arkivDir := filepath.Join(motesDir, "arkiv")
	sigPath := filepath.Join(arkivDir, "ATTESTATION.md.sig")

	if _, err := os.Stat(sigPath); err == nil {
		if !force {
			fmt.Println("❌ AVSLAGET: Arkivet är förseglat! Använd --force för att skriva över.")
			os.Exit(1)
		} else {
			fmt.Println("⚠️ FORCE aktivt: Raderar gamla manifest och signaturer...")
			os.Remove(sigPath)
			os.Remove(filepath.Join(arkivDir, "ATTESTATION.md"))
			os.Remove(filepath.Join(arkivDir, "PUBLIC_KEY.asc"))
		}
	}

	os.MkdirAll(arkivDir, 0o755)

	curr := kallorDir
	var projRoot string
	for {
		if _, err := os.Stat(filepath.Join(curr, ".tooling")); !os.IsNotExist(err) {
			projRoot = curr
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			fmt.Println("❌ Hittade inte rot-mappen (.tooling)!")
			os.Exit(1)
		}
		curr = parent
	}

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
		fmt.Println("❌ Hittade ingen .md-fil i källor !")
		os.Exit(1)
	}

	relMallPath, _ := filepath.Rel(arkivDir, orgMallPath)
	fmt.Printf("🔄 Bygger dokument för %s...\n", orgName)

	tempTypst := filepath.Join(arkivDir, baseName+"_temp.typ")
	runCmd("pandoc", mdFile, "-t", "typst", "-o", tempTypst, "--template", pandocTemplate, "-V", "org_mall="+relMallPath)

	pdfOut := filepath.Join(arkivDir, baseName+".pdf")
	runCmd("typst", "compile", "--root", projRoot, "--pdf-standard", "a-2b", tempTypst, pdfOut)
	os.Remove(tempTypst)

	htmlOut := filepath.Join(arkivDir, baseName+".html")
	runCmd("pandoc", mdFile, "-o", htmlOut, "--standalone")

	docxOut := filepath.Join(arkivDir, baseName+".docx")
	runCmd("pandoc", mdFile, "-o", docxOut)

	bilagorSrc := filepath.Join(kallorDir, "bilagor")
	bilagorDest := filepath.Join(arkivDir, "bilagor")
	if stat, err := os.Stat(bilagorSrc); err == nil && stat.IsDir() {
		copyDir(bilagorSrc, bilagorDest)
	}

	fmt.Println("✅ Klart! Output: ", arkivDir)
}

func doSeal(kallorPath string, key string, force bool) {
	kallorDir, _ := filepath.Abs(kallorPath)
	motesDir := filepath.Dir(kallorDir)
	arkivDir := filepath.Join(motesDir, "arkiv")
	manifestPath := filepath.Join(arkivDir, "ATTESTATION.md")
	sigPath := filepath.Join(arkivDir, "ATTESTATION.md.sig")
	pubKeyPath := filepath.Join(arkivDir, "PUBLIC_KEY.asc")

	if _, err := os.Stat(arkivDir); os.IsNotExist(err) {
		fmt.Println("❌ Hittar inte arkiv-mappen! Kör 'build' först.")
		os.Exit(1)
	}

	if _, err := os.Stat(sigPath); err == nil && !force {
		fmt.Println("❌ AVSLAGET: Arkivet är redan förseglat! Kör med --force.")
		os.Exit(1)
	}

	fmt.Println("🔒 Förseglar arkivet...")
	fmt.Println("   -> Exporterar signeringsnyckel (Public Key)...")
	os.Remove(pubKeyPath)
	runCmd("gpg", "--armor", "--export", "--output", pubKeyPath, key)

	var files []string
	filepath.WalkDir(arkivDir, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() && d.Name() != "ATTESTATION.md" && d.Name() != "ATTESTATION.md.sig" {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)

	f, _ := os.Create(manifestPath)
	f.WriteString("# Arkivmanifest\n\n")
	f.WriteString(fmt.Sprintf("**Förseglat:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	f.WriteString("Undertecknad intygar härmed att nedanstående filer\nuthör det formellt justerade och godkända dokumentet.\n\n")
	f.WriteString("### Arkivinnehåll:\n")
	for _, file := range files {
		relPath, _ := filepath.Rel(arkivDir, file)
		f.WriteString(fmt.Sprintf("- `%s` (SHA-256: `%s`)\n", relPath, hashFile(file)))
	}
	f.Close()

	os.Remove(sigPath)
	runCmd("gpg", "--detach-sign", "--armor", "--local-user", key, "--output", sigPath, manifestPath)

	// Koll om vi ska ZIPPA enligt inställningarna!
	var projRoot string
	curr := kallorDir
	for {
		if _, err := os.Stat(filepath.Join(curr, ".tooling")); !os.IsNotExist(err) {
			projRoot = curr
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	if projRoot != "" {
		cfg := loadConfig(projRoot)
		if cfg.Settings.CreateZIP {
			fmt.Println("   -> Paketerar AIP (Archival Information Package)...")
			zipName := filepath.Base(motesDir) + "_Arkivpaket.zip"
			zipPath := filepath.Join(motesDir, zipName)
			err := createZipArchive(arkivDir, zipPath)
			if err != nil {
				fmt.Println("❌ Misslyckades med att skapa ZIP:", err)
			} else {
				fmt.Println("   -> ZIP sparad som:", zipName)
			}
		}
	}

	// ============================================
	// OPENTIMESTAMPS (Tidsstämpling på Blockkedjan)
	// ============================================
	fmt.Println("   -> Söker efter OpenTimestamps för oantastligt tidsbevis...")

	// Kolla om 'ots' finns installerat på datorn
	_, errOTS := exec.LookPath("ots")
	if errOTS == nil {
		fmt.Println("   -> 'ots' hittades! Tidsstämplar manifestet mot Bitcoin-nätverket...")

		// Kör kommandot: ots stamp ATTESTATION.md.sig
		cmdOTS := exec.Command("ots", "stamp", sigPath)
		if err := cmdOTS.Run(); err != nil {
			fmt.Println("   ⚠️ Kunde inte nå OTS-servern just nu. Hoppar över tidsstämpel.")
		} else {
			fmt.Println("   ✅ Tidsstämpel skapad (.ots-fil sparad i arkivet).")
		}
	} else {
		fmt.Println("   💡 Tips: Installera 'opentimestamps-client' för att få gratis, kryptografiska tidsstämplar!")
	}

	fmt.Println("✅ Arkivet är låst och GPG-signerat.")
}

// ============================================
// CLI KOMMANDON
// ============================================
var (
	forceBuild bool
	forceSeal  bool
)

var rootCmd = &cobra.Command{
	Use: "docctl",
	Run: func(cmd *cobra.Command, args []string) {
		startTUI()
	},
}

var buildCmd = &cobra.Command{
	Use:  "build [org] [sökväg]",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		doBuild(args[0], args[1], forceBuild)
	},
}

var sealCmd = &cobra.Command{
	Use:  "seal [sökväg]",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gpgKey, _ := cmd.Flags().GetString("key")
		doSeal(args[0], gpgKey, forceSeal)
	},
}

func main() {
	sealCmd.Flags().StringP("key", "k", "", "GPG Key")
	sealCmd.MarkFlagRequired("key")
	buildCmd.Flags().BoolVarP(&forceBuild, "force", "f", false, "Tvinga ombyggnad")
	sealCmd.Flags().BoolVarP(&forceSeal, "force", "f", false, "Tvinga omförsegling")

	// Init behövs inte längre via argument nu när TUI ritar upp miljön så bra,
	// men vi behåller rootCmd för gränssnittet.
	rootCmd.AddCommand(buildCmd, sealCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ============================================
// TUI OCH FLÖDEN (MED ALTSCREEN)
// ============================================

// runForm tvingar alla formulär att lyda under Escape-tangenten och starta i egen AltScreen!
func runForm(f *huh.Form) error {
	km := huh.NewDefaultKeyMap()
	km.Quit.SetKeys("esc", "ctrl+c")

	// tea.WithAltScreen() är the holy grail här. Den ritar formuläret på en ren skärm,
	// och återskapar din terminalhistorkik perfekt när den stängs!
	return f.WithKeyMap(km).WithProgramOptions(tea.WithAltScreen()).Run()
}

func askSelect(title string, options []huh.Option[string], target *string) error {
	return runForm(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title(title).Options(options...).Value(target),
	)))
}

func askInput(title string, target *string) error {
	return runForm(huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(target),
	)))
}

func askConfirm(title string, target *bool) error {
	return runForm(huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(target),
	)))
}

// ------------------------------------------

func startTUI() {
	checkFirstRun()

	for {
		var action string
		options := []huh.Option[string]{
			huh.NewOption("✨ Skapa dokument / möte (Init)", "init"),
			huh.NewOption("🏗️ Bygg dokument (Build)", "build"),
			huh.NewOption("🔒 Försegla arkiv (Seal)", "seal"),
			huh.NewOption("⚙️ Inställningar", "settings"),
			huh.NewOption("❌ Avsluta", "exit"),
		}

		err := askSelect("🗄️ Föreningsarkivet", options, &action)

		if err != nil || action == "exit" {
			fmt.Println("Avslutar DocCtl. 👋")
			return
		}

		switch action {
		case "init":
			runInitFlow()
		case "build":
			runActionFlow("build")
		case "seal":
			runActionFlow("seal")
		case "settings":
			runSettingsFlow()
		}
	}
}

func runSettingsFlow() {
	for {
		cwd, _ := os.Getwd()
		cfg := loadConfig(cwd)
		var valdAction string

		options := []huh.Option[string]{
			huh.NewOption("➕ Lägg till ny förening", "ny_org"),
			huh.NewOption("✏️ Redigera befintlig förening", "redigera_org"),
			huh.NewOption("📦 Hantera ZIP-arkivering (AIP)", "toggle_zip"),
			huh.NewOption("⬅️ Tillbaka till Huvudmenyn", "back"),
		}

		err := askSelect("⚙️ Inställningar", options, &valdAction)
		if err != nil || valdAction == "back" {
			return
		}

		switch valdAction {
		case "ny_org":
			var nyID, nyNamn, nyOrgNr string
			err := runForm(huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("Kortnamn/ID (t.ex. SVDK):").Value(&nyID),
					huh.NewInput().Title("Fullt namn:").Value(&nyNamn),
					huh.NewInput().Title("Org.Nr:").Value(&nyOrgNr),
				),
			))

			if err != nil || nyID == "" {
				continue
			}

			cfg.Foreningar[nyID] = Forening{Namn: nyNamn, OrgNummer: nyOrgNr, Organ: []string{"styrelsen", "årsmöte"}}
			saveConfig(cwd, cfg)
			createOrgTemplate(cwd, nyID, nyNamn, nyOrgNr)

			fmt.Println("✅ Förening sparad! Du hittar den nu i menyerna.")
			pausePrompt() // <---- Skaparen får en chans att läsa detta innan menyn tar över skärmen igen

		case "redigera_org":
			orgOptions := []huh.Option[string]{}
			for key := range cfg.Foreningar {
				orgOptions = append(orgOptions, huh.NewOption(key, key))
			}
			if len(orgOptions) == 0 {
				continue
			}

			var valdOrg string
			if askSelect("Vilken förening?", orgOptions, &valdOrg) != nil {
				continue
			}

			f := cfg.Foreningar[valdOrg]
			nyNamn := f.Namn
			nyOrgNr := f.OrgNummer

			err = runForm(huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("Fullt namn:").Value(&nyNamn),
					huh.NewInput().Title("Org.Nr:").Value(&nyOrgNr),
				),
			))
			if err != nil {
				continue
			}
			f.Namn = nyNamn
			f.OrgNummer = nyOrgNr
			cfg.Foreningar[valdOrg] = f
			saveConfig(cwd, cfg)
			fmt.Println("✅ Ändringarna sparade!")
			pausePrompt()

		case "toggle_zip":
			sysZip := cfg.Settings.CreateZIP
			if askConfirm(fmt.Sprintf("Skapa automatiskt ZIP-arkiv? (Nu: %v)", sysZip), &sysZip) != nil {
				continue
			}

			cfg.Settings.CreateZIP = sysZip
			saveConfig(cwd, cfg)
			fmt.Println("✅ Inställningen sparad!")
			pausePrompt()
		}
	}
}

func runInitFlow() {
	cwd, _ := os.Getwd()
	cfg := loadConfig(cwd)
	var valdOrg, dokTyp, datum, organ, dokNamn string

	// 1. VÄLJ FÖRENING
	orgOptions := []huh.Option[string]{}
	for key, f := range cfg.Foreningar {
		orgOptions = append(orgOptions, huh.NewOption(fmt.Sprintf("%s (%s)", key, f.Namn), key))
	}
	orgOptions = append(orgOptions, huh.NewOption("➕ Lägg till ny förening...", "_NEW_ORG"))

	if askSelect("Vilken förening?", orgOptions, &valdOrg) != nil {
		return
	} // ESC

	if valdOrg == "_NEW_ORG" {
		var nyID, nyNamn, nyOrgNr string
		err := runForm(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Kortnamn/ID:").Value(&nyID),
				huh.NewInput().Title("Fullt namn:").Value(&nyNamn),
				huh.NewInput().Title("Org.Nr:").Value(&nyOrgNr),
			),
		))
		if err != nil || nyID == "" {
			return
		}
		cfg.Foreningar[nyID] = Forening{Namn: nyNamn, OrgNummer: nyOrgNr, Organ: []string{"styrelsen", "årsmöte"}}
		saveConfig(cwd, cfg)

		createOrgTemplate(cwd, nyID, nyNamn, nyOrgNr)
		valdOrg = nyID
	}

	// 2. VÄLJ DOKUMENTTYP

	// 2. VÄLJ HUVUDKATEGORI
	huvudKategoriOptions := []huh.Option[string]{
		huh.NewOption("📝 Protokoll (Årsakter)", "protokoll"),
		huh.NewOption("📜 Styrdokument/Policy (Grundakter)", "styrdokument"),
		huh.NewOption("🤝 Avtal (Grundakter)", "avtal"),
	}
	if askSelect("Vad vill du skapa?", huvudKategoriOptions, &dokTyp) != nil {
		return
	}

	var basePath, mdPath, mallText string

	// 3. LOGIK BASERAT PÅ VALD KATEGORI
	switch dokTyp {
	case "protokoll":
		aktuellFörening := cfg.Foreningar[valdOrg]
		organOptions := []huh.Option[string]{}
		for _, o := range aktuellFörening.Organ {
			organOptions = append(organOptions, huh.NewOption(o, o))
		}
		organOptions = append(organOptions, huh.NewOption("➕ Nytt organ...", "_NEW_ORGAN"))
		if askSelect("Vilket organ?", organOptions, &organ) != nil {
			return
		}

		if organ == "_NEW_ORGAN" {
			if askInput("Organets namn (t.ex. festkommitté):", &organ) != nil || organ == "" {
				return
			}
			aktuellFörening.Organ = append(aktuellFörening.Organ, organ)
			cfg.Foreningar[valdOrg] = aktuellFörening
			saveConfig(cwd, cfg)
		}
		if askInput("Datum (ÅÅÅÅ-MM-DD):", &datum) != nil || len(datum) < 4 {
			return
		}

		ar := datum[:4]
		basePath = filepath.Join(cwd, valdOrg, "Årsakter", ar, organ, datum, "källor")
		mdPath = filepath.Join(basePath, "protokoll.md")
		mallText = fmt.Sprintf("---\ntyp: protokoll\ntitle: Protokoll %s\ndatum: %s\ntid: 18:00\nplats: Föreningslokalen\nordforande: Namn Namnsson\nsekreterare: Namn Namnsson\njusterare:\n  - Justerare 1\n---\n\n## Mötets öppnande\n", organ, datum)

	case "styrdokument":
		// SKANNA EFTER BEFINTLIGA KATEGORIER: Leta i Grundakter/Styrdokument/
		underkatPath := filepath.Join(cwd, valdOrg, "Grundakter", "Styrdokument")
		var subKategori string

		kategoriOptions := []huh.Option[string]{}
		entries, err := os.ReadDir(underkatPath)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					kategoriOptions = append(kategoriOptions, huh.NewOption(e.Name(), e.Name()))
				}
			}
		}
		kategoriOptions = append(kategoriOptions, huh.NewOption("➕ Skapa ny kategori...", "_NEW_KAT"))

		if askSelect("Vilken typ av styrdokument?", kategoriOptions, &subKategori) != nil {
			return
		}

		if subKategori == "_NEW_KAT" {
			if askInput("Kategorins namn (t.ex. Policy, Reglemente, Stadgar):", &subKategori) != nil || subKategori == "" {
				return
			}
			// Ersätt ev. mellanslag så att den är säker för mappar
			subKategori = strings.ReplaceAll(subKategori, " ", "_")
		}

		if askInput("Dokumentets/Filens namn (t.ex. IT-policy):", &dokNamn) != nil || dokNamn == "" {
			return
		}
		mappNamn := strings.ReplaceAll(dokNamn, " ", "_")

		// Bygg vägen: Grundakter/Styrdokument/Policy/IT-policy/källor/
		basePath = filepath.Join(underkatPath, subKategori, mappNamn, "källor")
		mdPath = filepath.Join(basePath, "dokument.md")

		// Formatera filen snyggt beroende på kategori, men sätt styrdokument som fall-back!
		ymlKategori := strings.ToLower(subKategori)
		mallText = fmt.Sprintf("---\ntyp: %s\ntitle: %s\nversion: 1.0\nantagen: ÅÅÅÅ-MM-DD av Styrelsen\n---\n\n## 1. Syfte\nSyftet med detta dokument är...\n", ymlKategori, dokNamn)

	case "avtal":
		if askInput("Kort namn på avtalet (t.ex. Hyreskontrakt_Lokal):", &dokNamn) != nil || dokNamn == "" {
			return
		}
		mappNamn := strings.ReplaceAll(dokNamn, " ", "_")
		basePath = filepath.Join(cwd, valdOrg, "Grundakter", "Avtal", mappNamn, "källor")
		mdPath = filepath.Join(basePath, "avtal.md")
		mallText = fmt.Sprintf("---\ntyp: avtal\ntitle: %s\ndatum: %s\nparter:\n  - Föreningen\n  - Motparten AB\n---\n\n## 1. Avtalsobjekt\nDetta avtal avser...\n", dokNamn, time.Now().Format("2006-01-02"))
	}

	// SKAPA MAPPAR OCH FIL
	os.MkdirAll(filepath.Join(basePath, "bilagor"), 0o755)
	f, _ := os.Create(mdPath)
	f.WriteString(mallText)
	f.Close()

	fmt.Printf("\n✅ Succé! Skapade dokument för %s.\n📂 Sökväg: %s\n", valdOrg, mdPath)
	pausePrompt()
}

func runActionFlow(action string) {
	cwd, _ := os.Getwd()
	cfg := loadConfig(cwd)
	var valdOrg string

	orgOptions := []huh.Option[string]{}
	for key := range cfg.Foreningar {
		orgOptions = append(orgOptions, huh.NewOption(key, key))
	}
	if len(orgOptions) == 0 {
		fmt.Println("❌ Inga föreningar inlagda.")
		pausePrompt()
		return
	}

	if askSelect("Vilken förening?", orgOptions, &valdOrg) != nil {
		return
	} // ESC -> Huvudmeny

	// Sök i hela föreningens mapp (Både Årsakter och Grundakter)
	sokvag := filepath.Join(cwd, valdOrg)
	var motenOptions []huh.Option[string]

	filepath.WalkDir(sokvag, func(path string, d os.DirEntry, err error) error {
		// Hitta alla "källor"-mappar
		if d != nil && d.IsDir() && d.Name() == "källor" {
			// För Seal: Filtrera bort Styrdokument eftersom de bara lever i Git och inte ska låsas.
			if action == "seal" && strings.Contains(path, "Styrdokument") {
				return nil
			}

			relPath, _ := filepath.Rel(sokvag, filepath.Dir(path))
			motenOptions = append(motenOptions, huh.NewOption(relPath, path))
		}
		return nil
	})

	if len(motenOptions) == 0 {
		if action == "seal" {
			fmt.Println("❌ Hittade inga dokument som kan förseglas (Styrdokument förseglas inte här).")
		} else {
			fmt.Println("❌ Hittade inga dokument att bygga.")
		}
		pausePrompt()
		return
	}

	var valdKalla string
	if askSelect("Vilket dokument?", motenOptions, &valdKalla) != nil {
		return
	} // ESC

	if action == "build" {
		sigPath := filepath.Join(filepath.Dir(valdKalla), "arkiv", "ATTESTATION.md.sig")
		force := false
		if _, err := os.Stat(sigPath); err == nil {
			var confirm bool
			if askConfirm("⚠️ Arkivet/Avtalet är förseglat! Byggs det om raderas signaturen. Fortsätta?", &confirm) != nil || !confirm {
				fmt.Println("❌ Avbrutet.")
				pausePrompt()
				return
			}
			force = true
		}
		doBuild(valdOrg, valdKalla, force)
		pausePrompt()

	} else if action == "seal" {
		sigPath := filepath.Join(filepath.Dir(valdKalla), "arkiv", "ATTESTATION.md.sig")
		force := false
		if _, err := os.Stat(sigPath); err == nil {
			var confirm bool
			if askConfirm("⚠️ Detta är redan förseglat! Vill du skriva över signaturen?", &confirm) != nil || !confirm {
				fmt.Println("❌ Avbrutet.")
				pausePrompt()
				return
			}
			force = true
		}

		var gpgKey string
		if askInput("Ange GPG E-post/ID:", &gpgKey) != nil || gpgKey == "" {
			return
		}
		doSeal(valdKalla, gpgKey, force)
		pausePrompt()
	}
}
