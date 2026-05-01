package main

import (
	"bufio"
	"docctl/internal/app"
	"docctl/internal/assets"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea" // <--- NY!
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

//go:embed internal/assets/embeds

var embeddedFiles embed.FS

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

	if err := app.FirstTimeRun(toolingDir, cwd, embeddedFiles); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

// ============================================
// LOGIK MOTOR (doBuild & doSeal)
// ============================================

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
		err := app.DoBuild(args[1], forceBuild)
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
		err := app.DoSeal(args[0], gpgKey, forceSeal)
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
			err := app.CreateOrgTemplate(cwd, args[1], args[2], args[3], embeddedFiles)
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
		err := app.FirstTimeRun(toolingDir, cwd, assets.Files)
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
		cfg, _ := app.LoadConfig(cwd)
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

			cfg.Foreningar[nyID] = app.Association{Name: nyNamn, OrgNummer: nyOrgNr, Body: []string{"styrelsen", "årsmöte"}}
			if err := app.SaveConfig(cwd, cfg); err != nil {
				fmt.Println("Error occured while saving config")
				pausePrompt()
				return
			}
			if err := app.CreateOrgTemplate(cwd, nyID, nyNamn, nyOrgNr, embeddedFiles); err != nil {
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
			if err := app.SaveConfig(cwd, cfg); err != nil {
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
			if err := app.SaveConfig(cwd, cfg); err != nil {
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
			err := app.SaveConfig(cwd, cfg)
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
	cfg, _ := app.LoadConfig(cwd)
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

		if err := app.CreateOrgTemplate(cwd, nyID, nyNamn, nyOrgNr, embeddedFiles); err != nil {
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
			if err := app.SaveConfig(cwd, cfg); err != nil {
				return err
			}
		}
		if askInput("Datum (ÅÅÅÅ-MM-DD):", &datum) != nil || len(datum) < 4 {
			return nil
		}

		err := app.CreateProtokoll(cwd, valdOrg, organ, datum)
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
	cfg, _ := app.LoadConfig(cwd)
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
		if err := app.DoBuild(valdKalla, force); err != nil {
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
		if err := app.DoSeal(valdKalla, gpgKey, force); err != nil {
			fmt.Println(err)
		}
		pausePrompt()
	}
}
