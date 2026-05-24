package tui

import (
	"docctl/internal/app"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

func createMainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("action").Title("Huvudmeny").Options(
				huh.NewOption("✨ Skapa dokument / möte (Init)", "init"),
				huh.NewOption("🏗️ Bygg dokument (Build)", "build"),
				huh.NewOption("🔒 Försegla arkiv (Seal)", "seal"),
				huh.NewOption("⚙️ Inställningar", "settings"),
				huh.NewOption("❌ Avsluta", "exit"),
			),
		),
	)
}

func createSettingsMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("action").Title("Inställningar").Options(
				huh.NewOption("➕ Lägg till ny förening", "ny_org"),
				huh.NewOption("✏️ Redigera befintlig förening", "redigera_org"),
				huh.NewOption("📦 Hantera ZIP-arkivering (AIP)", "toggle_zip"),
				huh.NewOption("🕑 Hantera Opentimestamps", "toggle_ots"),
				huh.NewOption("🔄 Uppdatera standardmallar", "update_templates"),
				huh.NewOption("⬅️ Tillbaka till Huvudmenyn", "back"),
			),
		),
	)
}

func askForNewOrgDetailsForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Kortnamn/ID (t.ex. SVDK):").Key("org_id"),
			huh.NewInput().Title("Fullt namn:").Key("org_name"),
			huh.NewInput().Title("Org.Nr:").Key("org_number"),
		),
	)
}

func editOrgDetailsForm(name *string, number *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Fullt namn:").Key("org_name").Value(name),
			huh.NewInput().Title("Org.Nr:").Key("org_number").Value(number),
		),
	)
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

func askConfirmTemplateUpdateForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("confirm").
				Title("🔄 Uppdatera standardmallar?").
				Description("Detta kommer att skriva över ALLA standardmallar i .tooling/mallar/ med de inbyggda originalmallarna. Eventuella egna ändringar i dessa filer kommer att gå förlorade.").
				Affirmative("Ja, skriv över mallarna!").
				Negative("Nej, avbryt"))).WithTheme(huh.ThemeBase16())
}

func createConfirmForm(key string, title string, description string, currentVal bool) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key(key).
				Title(title).
				Description(description).
				Value(&currentVal),
		),
	).WithTheme(huh.ThemeBase16())
}

func chooseBodyForm(org app.Association) *huh.Form {
	var options []huh.Option[string]

	for _, o := range org.Body {
		options = append(options, huh.NewOption(o, o))
	}
	options = append(options, huh.NewOption("Skapa nytt organ ....", "create_new"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_body").Title("Vilket organ?").Options(options...),
		),
	)
}

func chooseSubcategoryForm(cwd string, org app.Association, pathParts []string, title string, excludeDirs ...string) *huh.Form {
	elements := append([]string{cwd, org.Id}, pathParts...)
	subcategoryPath := filepath.Join(elements...)

	var categoryOptions []huh.Option[string]
	entries, err := os.ReadDir(subcategoryPath)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				excluded := false
				for _, excl := range excludeDirs {
					if e.Name() == excl {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
				categoryOptions = append(categoryOptions, huh.NewOption(e.Name(), e.Name()))
			}
		}
	}
	categoryOptions = append(categoryOptions, huh.NewOption("➕ Skapa ny kategori...", "create_new"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("selected_document_type").Title(title).Options(categoryOptions...),
		),
	)
}

func chooseDocumentTypeForm() *huh.Form {
	var options []huh.Option[string]
	options = append(options, huh.NewOption("📝 Protokoll (Årsakter)", "protokoll"))
	options = append(options, huh.NewOption("📜 Styrdokument/Policy (Grundakter)", "styrdokument"))
	options = append(options, huh.NewOption("🤝 Andra dokument (Grundakter)", "other"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("document_type").Title("Vad vill du skapa?").Options(options...),
		),
	)
}

func createFirstRunForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("action").
				Title("Tom mapp upptäckt!").
				Description("Det verkar inte finnas något arkiv här.\nVill du initiera ett nytt Föreningsarkiv i denna mapp?").
				Affirmative("Ja, bygg arkivet!").
				Negative("Nej, avbryt"),
		),
	)
}

func genericInputForm(title string, key string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(title).Key(key),
		),
	)
}

func listAllDocumentsFromAssociation(cwd string, org app.Association) *huh.Form {
	searchPath := filepath.Join(cwd, org.Id)
	var docsOptions []huh.Option[string]

	_ = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() && d.Name() == "källor" {
			relPath, _ := filepath.Rel(searchPath, filepath.Dir(path))
			docsOptions = append(docsOptions, huh.NewOption(relPath, relPath))
		}
		return nil
	})

	if len(docsOptions) == 0 {
		docsOptions = append(docsOptions, huh.NewOption("Gå tillbaka", "back"))
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Inga dokument hittades").Options(docsOptions...).Key("document"),
			),
		)
	}
	docsOptions = append(docsOptions, huh.NewOption("Gå tillbaka", "back"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Vilket dokument?").Options(docsOptions...).Key("document"),
		),
	)
}

func listAllBuildDocumentsFromAssociation(cwd string, org app.Association) *huh.Form {
	searchPath := filepath.Join(cwd, org.Id)
	var docsOptions []huh.Option[string]

	_ = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && d.Name() == "arkiv" {
			pdfFiles, globErr := filepath.Glob(filepath.Join(path, "*.pdf"))
			if globErr == nil && len(pdfFiles) > 0 {
				relPath, _ := filepath.Rel(searchPath, filepath.Dir(path))
				docsOptions = append(docsOptions, huh.NewOption(relPath, relPath))
			}
			return filepath.SkipDir
		}

		return nil
	})

	if len(docsOptions) == 0 {
		docsOptions = append(docsOptions, huh.NewOption("Gå tillbaka", "back"))
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Inga dokument hittades").Options(docsOptions...).Key("document"),
			),
		)
	}
	docsOptions = append(docsOptions, huh.NewOption("Gå tillbaka", "back"))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Vilket dokument?").Options(docsOptions...).Key("document"),
		),
	)
}

func askGenericYesOrNowForm(title string, description string, key string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key(key).
				Title(title).
				Description(description),
		),
	).WithTheme(huh.ThemeBase16())
}
