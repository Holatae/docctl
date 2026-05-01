package main

import (
	"archive/zip"
	"bufio"
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
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea" // <--- NY!
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed embeds

var embeddedFiles embed.FS

var configMutex sync.Mutex

// Config ============================================
// DATAMODELLER FÖR CONFIG
// ============================================
type Config struct {
	Foreningar map[string]Association `yaml:"foreningar"`
	Settings   Settings               `yaml:"settings"`
}

type Association struct {
	Name      string   `yaml:"namn"`
	OrgNummer string   `yaml:"org_nummer"`
	Body      []string `yaml:"organ"`
}

type Settings struct {
	CreateZIP         bool `yaml:"create_zip"`
	UseOpenTimeStamps bool `yaml:"use_open_time_stamps"`
}

func loadConfig(projRoot string) (Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	var cfg Config

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg = Config{
			Foreningar: map[string]Association{},
			Settings:   Settings{CreateZIP: true, UseOpenTimeStamps: false},
		}
		if err := saveConfigLocked(projRoot, cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	_ = yaml.Unmarshal(data, &cfg)
	return cfg, nil
}

func saveConfig(projRoot string, cfg Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	return saveConfigLocked(projRoot, cfg)
}

func saveConfigLocked(projRoot string, cfg Config) error {
	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("could not make configpath: %w", err)
	}
	data, _ := yaml.Marshal(&cfg)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}
	return nil
}

// ============================================
// HJÄLPFUNKTIONER
// ============================================

// runCmd runs a given command in the terminal
func runCmd(name string, args ...string) error {
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

func hashFile(filePath string) (string, error) {
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

func createOrgTemplate(projRoot, orgID, orgNamn, orgNummer string) error {
	cfg, err := loadConfig(projRoot)
	if err != nil {
		return err
	}
	cwd, _ := os.Getwd()

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
			if err := saveConfig(cwd, cfg); err != nil {
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

func createZipArchive(srcDir string, destZip string) error {
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

// ------------------------------------------
// PAUSFUNKTIONEN (För att du ska hinna se loggarna)
// ------------------------------------------
func pausePrompt() {
	fmt.Println("\n[ Tryck på Enter för att återgå till menyn... ]")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ============================================
// FIRST RUN / ONBOARDING
// ============================================

func checkFirstRun() error {
	cwd, _ := os.Getwd()
	toolingDir := filepath.Join(cwd, ".tooling")

	// Om .tooling redan finns, är allt frid och fröjd. Avbryt och starta programmet.
	if _, err := os.Stat(toolingDir); !os.IsNotExist(err) {
		return nil
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

	if err := firstTimeRun(toolingDir, cwd); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

// firstTimeRun creates all relevant folders and files
func firstTimeRun(toolingDir string, cwd string) (err error) {
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
# arkiv/*.pdf
# arkiv/*.docx
`
	if err := os.WriteFile(filepath.Join(cwd, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return fmt.Errorf("could not write .gitignore: %v", err)
	}

	// 4. Initiera Git automatiskt (om git finns installerat)
	if _, err := exec.LookPath("git"); err == nil {
		fmt.Println("   -> Sätter upp versionshantering (git init)...")
		if err := exec.Command("git", "init").Run(); err != nil {
			return fmt.Errorf("could not git init: %v", err)
		}
	}

	fmt.Println("✅ Arbetsutrymme skapat! Du är redo att köra.")
	return nil
}

// ============================================
// LOGIK MOTOR (doBuild & doSeal)
// ============================================

// doBuild generates document in multiple formats (PDF, HTML, DOCX)
// from source files in pathToSources.
// force: forces the rebuilding of the documents even if the archive is Sealed
func doBuild(pathToSources string, force bool) (err error) {
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
	if err := runCmd("pandoc", mdFile, "-t", "typst", "-o", tempTypst, "--template", pandocTemplate, "-V", "org_mall="+relMallPath); err != nil {
		return fmt.Errorf("pandoc misslyckades %w", err)
	}

	pdfOut := filepath.Join(arkivDir, baseName+".pdf")

	if err := runCmd("typst", "compile", "--root", projRoot, "--pdf-standard", "a-2b", tempTypst, pdfOut); err != nil {
		return fmt.Errorf("typst failed to compile: %w", err)
	}

	_ = os.Remove(tempTypst)

	htmlOut := filepath.Join(arkivDir, baseName+".html")
	if err := runCmd("pandoc", mdFile, "-o", htmlOut, "--standalone"); err != nil {
		return fmt.Errorf("pandoc failed to compile (HTML): %w", err)
	}

	docxOut := filepath.Join(arkivDir, baseName+".docx")
	if err := runCmd("pandoc", mdFile, "-o", docxOut); err != nil {
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

func doSeal(kallorPath string, key string, force bool) (err error) {
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

	if _, err := os.Stat(pubKeyPath); !os.IsNotExist(err) {
		err = os.Remove(pubKeyPath)
		if err != nil {
			return fmt.Errorf("could not remove public key: %w", err)
		}
	}

	err = runCmd("gpg", "--armor", "--export", "--output", pubKeyPath, key)
	if err != nil {
		return fmt.Errorf("could not armor signeringsnyckel: %w", err)
	}

	var files []string
	err = filepath.WalkDir(arkivDir, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() && d.Name() != "ATTESTATION.md" && d.Name() != "ATTESTATION.md.sig" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	f, _ := os.Create(manifestPath)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString("# Arkivmanifest\n\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	if _, err := f.WriteString(fmt.Sprintf("**Förseglat:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if _, err := f.WriteString("Undertecknad intygar härmed att nedanstående filer\nuthör det formellt justerade och godkända dokumentet.\n\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if _, err := f.WriteString("### Arkivinnehåll:\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	for _, file := range files {
		relPath, err := filepath.Rel(arkivDir, file)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path")
		}
		hashStr, err := hashFile(file)
		if err != nil {
			return fmt.Errorf("failed to calculate hash")
		}

		line := fmt.Sprintf("- `%s` (SHA-256: `%s`)\n", relPath, hashStr)

		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to file")
		}
	}
	_ = f.Close()

	if _, err := os.Stat(sigPath); !os.IsNotExist(err) {
		err = os.Remove(sigPath)
		if err != nil {
			return fmt.Errorf("could not remove sig file: %w", err)
		}
	}

	err = runCmd("gpg", "--detach-sign", "--armor", "--local-user", key, "--output", sigPath, manifestPath)
	if err != nil {
		return fmt.Errorf("could not armor signeringsnyckel: %w", err)
	}

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
		cfg, err := loadConfig(projRoot)
		if err != nil {
			return err
		}
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

	cfg, err := loadConfig(projRoot)
	if err != nil {
		return err
	}

	// Only for OpenTimeStamps
	if cfg.Settings.UseOpenTimeStamps {
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
	}

	fmt.Println("✅ Arkivet är låst och GPG-signerat.")

	return nil
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
		err := doBuild(args[1], forceBuild)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var sealCmd = &cobra.Command{
	Use:  "seal [sökväg]",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gpgKey, _ := cmd.Flags().GetString("key")
		err := doSeal(args[0], gpgKey, forceSeal)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var orgCmd = &cobra.Command{
	Use:  "org [add] [short name] [long name] [org-number]",
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		if args[0] == "add" {
			err := createOrgTemplate(cwd, args[1], args[2], args[3])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
	Example: "doctl org add TestOrg \"Test Organization\" 802000-1234",
	Short:   "Adds a new organization",
}

var initCmd = &cobra.Command{
	Use: "init",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		toolingDir := filepath.Join(cwd, ".tooling")
		err := firstTimeRun(toolingDir, cwd)
		if err != nil {
			_ = fmt.Errorf("error occurred while initializing tooling: %v", err)
			os.Exit(1)
		}
	},
	Short: "Use for initialization of directory",
}

func main() {
	sealCmd.Flags().StringP("key", "k", "", "GPG Key")
	_ = sealCmd.MarkFlagRequired("key")
	buildCmd.Flags().BoolVarP(&forceBuild, "force", "f", false, "Tvinga ombyggnad")
	sealCmd.Flags().BoolVarP(&forceSeal, "force", "f", false, "Tvinga omförsegling")

	// Init behövs inte längre via argument nu när TUI ritar upp miljön så bra,
	// men vi behåller rootCmd för gränssnittet.
	rootCmd.AddCommand(buildCmd, sealCmd, initCmd, orgCmd)
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
	_ = checkFirstRun()

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
			_ = runInitFlow()
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
		cfg, _ := loadConfig(cwd)
		var valdAction string

		options := []huh.Option[string]{
			huh.NewOption("➕ Lägg till ny förening", "ny_org"),
			huh.NewOption("✏️ Redigera befintlig förening", "redigera_org"),
			huh.NewOption("📦 Hantera ZIP-arkivering (AIP)", "toggle_zip"),
			huh.NewOption("🕑 Hantera Opentimestamps", "toggle_ots"),
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

			cfg.Foreningar[nyID] = Association{Name: nyNamn, OrgNummer: nyOrgNr, Body: []string{"styrelsen", "årsmöte"}}
			if err := saveConfig(cwd, cfg); err != nil {
				fmt.Println("Error occured while saving config")
				pausePrompt()
				return
			}
			if err := createOrgTemplate(cwd, nyID, nyNamn, nyOrgNr); err != nil {
				fmt.Println("Error occured while creating template")
				pausePrompt()
				return
			}

			fmt.Println("✅ Förening sparad! Du hittar den nu i menyerna.")
			pausePrompt() // <---- Skaparen får en chans att läsa detta innan menyn tar över skärmen igen

		case "redigera_org":
			var orgOptions []huh.Option[string]
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
			nyNamn := f.Name
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
			f.Name = nyNamn
			f.OrgNummer = nyOrgNr
			cfg.Foreningar[valdOrg] = f
			if err := saveConfig(cwd, cfg); err != nil {
				fmt.Printf("Error occurred while saving config %v\n", err)
			}
			fmt.Println("✅ Ändringarna sparade!")
			pausePrompt()

		case "toggle_zip":
			sysZip := cfg.Settings.CreateZIP
			if askConfirm(fmt.Sprintf("Skapa automatiskt ZIP-arkiv? (Nu: %v)", sysZip), &sysZip) != nil {
				continue
			}

			cfg.Settings.CreateZIP = sysZip
			if err := saveConfig(cwd, cfg); err != nil {
				fmt.Printf("Error occurred while saving config %v\n", err)
			}
			fmt.Println("✅ Inställningen sparad!")
			pausePrompt()

		case "toggle_ots":
			sysOts := cfg.Settings.UseOpenTimeStamps
			if askConfirm(fmt.Sprintf("Skapa timestamp med OpenTimeStamp (Nu: %v)", sysOts), &sysOts) != nil {
				continue
			}

			cfg.Settings.UseOpenTimeStamps = sysOts
			err := saveConfig(cwd, cfg)
			if err != nil {
				fmt.Println("KUNDE INTE SPARA INSTÄLLNINGEN")
				pausePrompt()
			} else {
				fmt.Println("✅ Inställningen sparad!")
				pausePrompt()
			}

		}
	}
}

func createProtokoll(cwd string, orgId string, body string, date string) error {
	cfg, _ := loadConfig(cwd)

	currentOrg := cfg.Foreningar[orgId]

	// Check if body exists otherwise create new

	if !slices.Contains(currentOrg.Body, body) {
		currentOrg.Body = append(currentOrg.Body, body)
		cfg.Foreningar[orgId] = currentOrg
		if err := saveConfig(cwd, cfg); err != nil {
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

func createGuidanceDocuments(cwd string, orgId string, subcategory string, docName string) error {

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

//func createOtherGoverningDocument

func runInitFlow() error {
	cwd, _ := os.Getwd()
	cfg, _ := loadConfig(cwd)
	var valdOrg, dokTyp, datum, organ, dokNamn string

	// 1. VÄLJ FÖRENING
	var orgOptions []huh.Option[string]
	for key, f := range cfg.Foreningar {
		orgOptions = append(orgOptions, huh.NewOption(fmt.Sprintf("%s (%s)", key, f.Name), key))
	}
	orgOptions = append(orgOptions, huh.NewOption("➕ Lägg till ny förening...", "_NEW_ORG"))

	if askSelect("Vilken förening?", orgOptions, &valdOrg) != nil {
		return nil
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
			return nil
		}

		if err := createOrgTemplate(cwd, nyID, nyNamn, nyOrgNr); err != nil {
			return err
		}
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
		return nil
	}

	var basePath, mdPath string

	// 3. LOGIK BASERAT PÅ VALD KATEGORI
	switch dokTyp {
	case "protokoll":
		currentAssociation := cfg.Foreningar[valdOrg]
		var organOptions []huh.Option[string]
		for _, o := range currentAssociation.Body {
			organOptions = append(organOptions, huh.NewOption(o, o))
		}
		organOptions = append(organOptions, huh.NewOption("➕ Nytt organ...", "_NEW_ORGAN"))
		if askSelect("Vilket organ?", organOptions, &organ) != nil {
			return nil
		}

		if organ == "_NEW_ORGAN" {
			if askInput("Organets namn (t.ex. festkommitté):", &organ) != nil || organ == "" {
				return nil
			}
			currentAssociation.Body = append(currentAssociation.Body, organ)
			cfg.Foreningar[valdOrg] = currentAssociation
			if err := saveConfig(cwd, cfg); err != nil {
				return err
			}
		}
		if askInput("Datum (ÅÅÅÅ-MM-DD):", &datum) != nil || len(datum) < 4 {
			return nil
		}

		err := createProtokoll(cwd, valdOrg, organ, datum)
		if err != nil {
			return err
		}

		//ar := datum[:4]
		//basePath = filepath.Join(cwd, valdOrg, "Årsakter", ar, organ, datum, "källor")
		//mdPath = filepath.Join(basePath, "protokoll.md")
		//mallText = fmt.Sprintf("---\ntyp: protokoll\ntitle: Protokoll %s\ndatum: %s\ntid: 18:00\nplats: Föreningslokalen\nordforande: Namn Namnsson\nsekreterare: Namn Namnsson\njusterare:\n  - Justerare 1\n---\n\n## Mötets öppnande\n", organ, datum)

	case "styrdokument":

		// SKANNA EFTER BEFINTLIGA KATEGORIER: Leta i Grundakter/Styrdokument/
		underkatPath := filepath.Join(cwd, valdOrg, "Grundakter", "Styrdokument")
		var subKategori string

		var kategoriOptions []huh.Option[string]
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
			return nil
		}

		if subKategori == "_NEW_KAT" {
			if askInput("Kategorins namn (t.ex. Policy, Reglemente, Stadgar):", &subKategori) != nil || subKategori == "" {
				return nil
			}
			// Ersätt ev. mellanslag så att den är säker för mappar
			subKategori = strings.ReplaceAll(subKategori, " ", "_")
		}

		if askInput("Dokumentets/Filens namn (t.ex. IT-policy):", &dokNamn) != nil || dokNamn == "" {
			return nil
		}

		err = createGuidanceDocuments(cwd, valdOrg, subKategori, dokNamn)
		if err != nil {
			return err
		}

	/*	mappNamn := strings.ReplaceAll(dokNamn, " ", "_")

		// Bygg vägen: Grundakter/Styrdokument/Policy/IT-policy/källor/
		basePath = filepath.Join(underkatPath, subKategori, mappNamn, "källor")
		mdPath = filepath.Join(basePath, "dokument.md")

		// Formatera filen snyggt beroende på kategori, men sätt styrdokument som fall-back!
		ymlKategori := strings.ToLower(subKategori)
		mallText = fmt.Sprintf("---\ntyp: %s\ntitle: %s\nversion: 1.0\nantagen: ÅÅÅÅ-MM-DD av Styrelsen\n---\n\n## 1. Syfte\nSyftet med detta dokument är...\n", ymlKategori, dokNamn)
	*/
	case "avtal":
		if askInput("Kort namn på avtalet (t.ex. Hyreskontrakt_Lokal):", &dokNamn) != nil || dokNamn == "" {
			return nil
		}
		mappNamn := strings.ReplaceAll(dokNamn, " ", "_")
		basePath = filepath.Join(cwd, valdOrg, "Grundakter", "Avtal", mappNamn, "källor")
		mdPath = filepath.Join(basePath, "avtal.md")
		//mallText = fmt.Sprintf("---\ntyp: avtal\ntitle: %s\ndatum: %s\nparter:\n  - Föreningen\n  - Motparten AB\n---\n\n## 1. Avtalsobjekt\nDetta avtal avser...\n", dokNamn, time.Now().Format("2006-01-02"))
	}

	// SKAPA MAPPAR OCH FIL
	//os.MkdirAll(filepath.Join(basePath, "bilagor"), 0o755)
	//f, _ := os.Create(mdPath)
	//f.WriteString(mallText)
	//f.Close()

	fmt.Printf("\n✅ Succé! Skapade dokument för %s.\n📂 Sökväg: %s\n", valdOrg, mdPath)
	pausePrompt()
	return nil
}

func runActionFlow(action string) {
	cwd, _ := os.Getwd()
	cfg, _ := loadConfig(cwd)
	var valdOrg string

	var orgOptions []huh.Option[string]
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

	_ = filepath.WalkDir(sokvag, func(path string, d os.DirEntry, err error) error {
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
		if err := doBuild(valdKalla, force); err != nil {
			fmt.Printf("Failed to build %v\n", err)
		}
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
		if err := doSeal(valdKalla, gpgKey, force); err != nil {
			fmt.Println(err)
		}
		pausePrompt()
	}
}
