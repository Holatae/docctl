package tui

import (
	"docctl/internal/app"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

type OrgDetails struct {
	ID     string
	Name   string
	Number string
}

func runActionFlow(action string) {
	cwd, _ := os.Getwd()
	cfg, _ := app.LoadConfig(cwd)
	var valdOrg string

	var orgOptions []huh.Option[string]
	for key := range cfg.Foreningar {
		if key == "" {
			continue
		}
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

func runInitFlow() error {
	cwd, _ := os.Getwd()
	cfg, _ := app.LoadConfig(cwd)
	var valdOrg, dokTyp, datum, organ, dokNamn string

	// 1. VÄLJ FÖRENING
	var orgOptions []huh.Option[string]
	for key, f := range cfg.Foreningar {
		if key == "" {
			continue
		}
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

		if err := app.CreateOrganization(cwd, nyID, nyNamn, nyOrgNr); err != nil {
			return err
		}
		valdOrg = nyID

		cfg, _ = app.LoadConfig(cwd)
	}

	// 2. VÄLJ DOKUMENTTYP

	// 2. VÄLJ HUVUDKATEGORI
	huvudKategoriOptions := []huh.Option[string]{
		huh.NewOption("📝 Protokoll (Årsakter)", "protokoll"),
		huh.NewOption("📜 Styrdokument/Policy (Grundakter)", "styrdokument"),
		huh.NewOption("🤝 Andra dokument (Grundakter)", "other"),
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

	case "styrdokument":

		// SKANNA EFTER BEFINTLIGA KATEGORIER: Leta i Grundakter/Styrdokument/
		subcategoryPath := filepath.Join(cwd, valdOrg, "Grundakter", "Styrdokument")
		var subcategory string

		var categoryOptions []huh.Option[string]
		entries, err := os.ReadDir(subcategoryPath)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					categoryOptions = append(categoryOptions, huh.NewOption(e.Name(), e.Name()))
				}
			}
		}
		categoryOptions = append(categoryOptions, huh.NewOption("➕ Skapa ny kategori...", "_NEW_KAT"))

		if askSelect("Vilken typ av styrdokument?", categoryOptions, &subcategory) != nil {
			return nil
		}

		if subcategory == "_NEW_KAT" {
			if askInput("Kategorins namn (t.ex. Policy, Reglemente, Stadgar):", &subcategory) != nil || subcategory == "" {
				return nil
			}
			// Ersätt ev. mellanslag så att den är säker för mappar
			subcategory = strings.ReplaceAll(subcategory, " ", "_")
		}

		if askInput("Dokumentets/Filens namn (t.ex. IT-policy):", &dokNamn) != nil || dokNamn == "" {
			return nil
		}

		err = app.CreateGuidanceDocuments(cwd, valdOrg, subcategory, dokNamn)
		if err != nil {
			return err
		}

	case "other":

		subcategoryPath := filepath.Join(cwd, valdOrg, "Grundakter")

		var categoryOptions []huh.Option[string]
		var subCategory string

		entries, err := os.ReadDir(subcategoryPath)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					if e.Name() == "Styrdokument" {
						continue
					}
					categoryOptions = append(categoryOptions, huh.NewOption(e.Name(), e.Name()))
				}
			}
		}

		categoryOptions = append(categoryOptions, huh.NewOption("➕ Skapa ny kategori...", "_NEW_KAT"))

		if askSelect("Vilken typ av Grundakt?", categoryOptions, &subCategory) != nil {
			return nil
		}
		if subCategory == "_NEW_KAT" {
			if askInput("Kategorins namn (t.ex. Avtal, Motioner, Propositioner)", &subCategory) != nil || subCategory == "" {
				return nil
			}

			subCategory = strings.ReplaceAll(subCategory, " ", "_")

		}

		if askInput("Dokumentets/Filens namn (t.ex Motion angående xx)", &dokNamn) != nil || dokNamn == "" {
			return nil
		}

		err = app.CreateOtherGoverningDocuments(cwd, valdOrg, subCategory, dokNamn)

		if err != nil {
			return err
		}

		safeDocName := strings.ReplaceAll(dokNamn, " ", "_")
		folderName := "avtal"
		basePath = filepath.Join(cwd, valdOrg, "Grundakter", "Avtal", safeDocName, "källor")
		mdPath = filepath.Join(basePath, "avtal.md")

		if err := app.CreateOtherGoverningDocuments(cwd, valdOrg, folderName, safeDocName); err != nil {
			return err
		}

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

func createMainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("action").Title("Huvudmeny").Options(
				huh.NewOption("✨ Skapa dokument / möte (Init)", "init"),
				huh.NewOption("🏗️ Bygg dokument (Build)", "build"),
				huh.NewOption("🔒 Försegla arkiv (Seal)", "seal"),
				huh.NewOption("⚙️ Inställningar", "settings"),
				huh.NewOption("❌ Avsluta", "exit"),
			)))
}

func createSettingsMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("action").Title("Inställningar").Options(
				huh.NewOption("➕ Lägg till ny förening", "ny_org"),
				huh.NewOption("✏️ Redigera befintlig förening", "redigera_org"),
				huh.NewOption("📦 Hantera ZIP-arkivering (AIP)", "toggle_zip"),
				huh.NewOption("🕑 Hantera Opentimestamps", "toggle_ots"),
				huh.NewOption("⬅️ Tillbaka till Huvudmenyn", "back"),
			)))
}

func askForNewOrgDetailsForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Kortnamn/ID (t.ex. SVDK:)").Key("org_id"),
			huh.NewInput().Title("Fullt namn:").Key("org_name"),
			huh.NewInput().Title("Org.Nr:").Key("org_number")))
}

func createChooseOrgForm(orgs map[string]app.Association, title string, allowCreateNew bool) *huh.Form {

	var options []huh.Option[string]

	// 1. Visa alla föreningar
	for _, org := range orgs {
		if org.Name == "" {
			continue
		}
		showText := fmt.Sprintf("(%s) %s", org.Id, org.Name)

		options = append(options, huh.NewOption(showText, org.Id))
	}
	// 2. Ska skapa ny förening visas
	if allowCreateNew {
		options = append(options, huh.NewOption("Skapa ny förening ....", "create_new"))
	}

	// 3. Tillbaka knappen
	options = append(options, huh.NewOption("Tillbaka", "back"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_org").Title(title).Options(options...)))
}

func showSettingsMenu() (string, error) {
	var action string

	options := []huh.Option[string]{
		huh.NewOption("➕ Lägg till ny förening", "ny_org"),
		huh.NewOption("✏️ Redigera befintlig förening", "redigera_org"),
		huh.NewOption("📦 Hantera ZIP-arkivering (AIP)", "toggle_zip"),
		huh.NewOption("🕑 Hantera Opentimestamps", "toggle_ots"),
		huh.NewOption("⬅️ Tillbaka till Huvudmenyn", "back"),
	}

	err := askSelect("⚙️ Inställningar", options, &action)

	return action, err
}

func askForNewOrgDetails() (OrgDetails, error) {
	var orgDetails OrgDetails

	err := runForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Kortnamn/ID (t.ex. SVDK):").Value(&orgDetails.ID),
			huh.NewInput().Title("Fullt namn:").Value(&orgDetails.Name),
			huh.NewInput().Title("Org.Nr:").Value(&orgDetails.Number),
		),
	))

	return orgDetails, err

}

func createOpenTimeStampsForm(currentVal bool) *huh.Form {
	return huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Key("ots").
			Title("Använd OpenTimestamps?").
			Description("Tidsstämplar sealed filer via blockkedjieteknik.").
			Value(new(currentVal))))

}

func chooseBodyForm(org app.Association) *huh.Form {
	var options []huh.Option[string]

	for _, o := range org.Body {
		options = append(options, huh.NewOption(o, o))
	}
	options = append(options, huh.NewOption("Skapa nytt organ ....", "create_new"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_body").Title("Vilket organ?").Options(options...)))
}

func chooseGoverningDocumentTypeForm(cwd string, org app.Association) *huh.Form {
	// SKANNA EFTER BEFINTLIGA KATEGORIER: Leta i Grundakter/Styrdokument/
	subcategoryPath := filepath.Join(cwd, org.Id, "Grundakter", "Styrdokument")

	var categoryOptions []huh.Option[string]
	entries, err := os.ReadDir(subcategoryPath)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				categoryOptions = append(categoryOptions, huh.NewOption(e.Name(), e.Name()))
			}
		}
	}
	categoryOptions = append(categoryOptions, huh.NewOption("➕ Skapa ny kategori...", "create_new"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_document_type").Title("Vilken typ av styrdokument?").Options(categoryOptions...)))

}

func genericInputForm(title string, key string) *huh.Form {
	return huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Key(key)))
}

func chooseOtherDocumentTypeForm(cwd string, org app.Association) *huh.Form {
	// SKANNA EFTER BEFINTLIGA KATEGORIER: Leta i Grundakter/
	subcategoryPath := filepath.Join(cwd, org.Id, "Grundakter")

	var categoryOptions []huh.Option[string]
	entries, err := os.ReadDir(subcategoryPath)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				if e.Name() == "Styrdokument" {
					continue
				}
				categoryOptions = append(categoryOptions, huh.NewOption(e.Name(), e.Name()))
			}
		}
	}
	categoryOptions = append(categoryOptions, huh.NewOption("➕ Skapa ny kategori...", "create_new"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_document_type").Title("Vilken typ av Grundakt?").Options(categoryOptions...)))
}

func chooseDocumentTypeForm() *huh.Form {

	var options []huh.Option[string]
	options = append(options, huh.NewOption("📝 Protokoll (Årsakter)", "protokoll"))
	options = append(options, huh.NewOption("📜 Styrdokument/Policy (Grundakter)", "styrdokument"))
	options = append(options, huh.NewOption("🤝 Andra dokument (Grundakter)", "other"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("document_type").Title("Vad vill du skapa?").Options(options...)))
}

func createFirstRunForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("action").
				Title("Tom mapp upptäckt!").
				Description("Det verkar inte finnas något arkiv här.\nVill du initiera ett nytt Föreningsarkiv i denna mapp?").
				Affirmative("Ja, bygg arkivet!").
				Negative("Nej, avbryt")))
}

func createZipForm(currentVal bool) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("zip").
				Title("Skapa automatiskt ZIP-arkiv?").
				Description("Paketerar alla filer i en zip-fil när bygget är klart.").
				Value(new(currentVal)))).WithTheme(huh.ThemeBase16())
}

func askForAssociationForm(context *AppContext) (string, error) {
	var orgOptions []huh.Option[string]
	for key := range context.Cfg.Foreningar {
		if key == "" {
			continue
		}
		orgOptions = append(orgOptions, huh.NewOption(key, key))
	}
	if len(orgOptions) == 0 {
		return "", fmt.Errorf("no association found")
	}

	var valdOrg string
	if err := askSelect("Vilken förening?", orgOptions, &valdOrg); err != nil {
		return "", err
	}

	return valdOrg, nil

}

func askForNewDetailsForAssociationForm(org OrgDetails) (OrgDetails, error) {
	err := runForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Fullt namn:").Value(&org.Name),
			huh.NewInput().Title("Org.Nr:").Value(&org.Number),
		),
	))

	return org, err
}

func askToggleSetting(title string, currentValue bool) (bool, error) {
	newValue := currentValue
	prompt := fmt.Sprintf("%s (Nu: %v)", title, newValue)

	if err := askConfirm(prompt, &newValue); err != nil {
		return currentValue, err
	}

	return newValue, nil
}
