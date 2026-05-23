package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"docctl/internal/app"
	"docctl/internal/assets"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type AppContext struct {
	Cwd string
	Cfg *app.Config
}

type appState int

const (
	stateMainMenu appState = iota
	stateSettingsMenu
	stateEditZip
	stateEditOts
	stateChooseOrgForBuild
	stateChooseOrgForSeal
	stateChooseDocumentToSeal
	stateAskForPGPKey
	stateChooseOrgForDoc
	stateCreateOrganization
	stateFirstRun
	stateCreateBody
	stateChooseBodyForm
	stateChooseDocumentTypeForm
	stateAskDateForm
	stateChooseGoverningDocumentTypeForm
	stateCreateNewGoverningDocumentTypeForm
	stateAskGoverningDocumentNameForm
	stateChooseOtherDocumentType
	stateChooseOtherDocumentCategoryNameForm
	stateCreateOtherDocumentTypeForm
	stateChooseDocumentToBuild
	stateAskIfUserWantToNukeSealedDocument
	stateAskIfUserWantToNukeSealedDocumentForSealedDocument
	stateConfirmTemplateUpdate
)

type mainModel struct {
	state appState
	cwd   string
	cfg   *app.Config

	selectedOrg             app.Association
	selectedDocumentType    string
	selectedBody            string
	date                    string
	force                   bool
	selectedDocumentToBuild string
	selectedDocumentToSeal  string

	selectedGoverningDocType string
	selectedGoverningDocName string
	grundaktCategory         string

	activeForm *huh.Form

	// Formulär
	mainMenu     *huh.Form
	settingsMenu *huh.Form
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

func (m mainModel) updateMainMenu(keyMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	form, cmd := m.mainMenu.Update(keyMsg)
	if f, ok := form.(*huh.Form); ok {
		m.mainMenu = f
	}
	cmds = append(cmds, cmd)

	// Blev menyn klar? (tryckte användaren enter?)
	if m.mainMenu.State == huh.StateCompleted {
		action := m.mainMenu.GetString("action")

		switch action {
		case "settings":
			m.state = stateSettingsMenu
			m.settingsMenu = createSettingsMenuForm()

			cmds = append(cmds, m.settingsMenu.Init())
			m.mainMenu = createMainMenuForm()

		case "init":
			m.state = stateChooseOrgForDoc
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
			cmds = append(cmds, m.activeForm.Init())

			m.mainMenu = createMainMenuForm()

		case "build":
			m.state = stateChooseOrgForBuild
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", false)
			cmds = append(cmds, m.activeForm.Init())

		case "seal":
			m.state = stateChooseOrgForSeal
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", false)
			cmds = append(cmds, m.activeForm.Init())
		case "exit":
			return m, tea.Quit
		}

	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateSettingsMenu(keyMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	form, cmd := m.settingsMenu.Update(keyMsg)
	if f, ok := form.(*huh.Form); ok {
		m.settingsMenu = f
	}
	cmds = append(cmds, cmd)

	if keyMsg, ok := keyMsg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.state = stateMainMenu
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.settingsMenu.State == huh.StateCompleted {
		action := m.settingsMenu.GetString("action")
		switch action {
		case "toggle_zip":
			m.activeForm = createZipForm(m.cfg.Settings.CreateZIP)
			m.state = stateEditZip
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.activeForm.Init())
		case "toggle_ots":
			m.activeForm = createOpenTimeStampsForm(m.cfg.Settings.UseOpenTimeStamps)
			m.state = stateEditOts
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.activeForm.Init())
		case "update_templates":
			m.activeForm = askConfirmTemplateUpdateForm()
			m.state = stateConfirmTemplateUpdate
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.activeForm.Init())
		case "back":
			m.state = stateMainMenu
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

// Update is the function that actually switches the form
func (m mainModel) Update(keyMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := keyMsg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateMainMenu:
		return m.updateMainMenu(keyMsg)
	case stateSettingsMenu:
		return m.updateSettingsMenu(keyMsg)

	case stateChooseOrgForBuild:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			selectedOrgId := m.activeForm.GetString("selected_org")
			*m.cfg, _ = app.LoadConfig(m.cwd)

			m.selectedOrg = m.cfg.Foreningar[selectedOrgId]

			m.state = stateChooseDocumentToBuild
			m.activeForm = listAllDocumentsFromAssociation(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())

		}

	case stateChooseOrgForSeal:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			selectedOrgId := m.activeForm.GetString("selected_org")
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.selectedOrg = m.cfg.Foreningar[selectedOrgId]
			m.state = stateChooseDocumentToSeal
			m.activeForm = listAllBuildDocumentsFromAssociation(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())
		}

	case stateChooseDocumentToSeal:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.force {
			m.state = stateAskForPGPKey
			m.activeForm = genericInputForm("Ange GPG E-post/ID:", "pgp")
			cmds = append(cmds, m.activeForm.Init())
		}

		if m.activeForm.State == huh.StateCompleted {
			m.selectedDocumentToSeal = m.activeForm.GetString("document")

			switch m.selectedDocumentToSeal {
			case "back":
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			default:
				sigPath := filepath.Join(filepath.Dir(m.selectedDocumentToBuild), "arkiv", "ATTESTATION.md.sig")

				if _, err := os.Stat(sigPath); err == nil {
					m.state = stateAskIfUserWantToNukeSealedDocumentForSealedDocument
					m.activeForm = askGenericYesOrNowForm("⚠️ Arkivet/Avtalet är förseglat!", " Seals det om raderas signaturen. Fortsätta?", "action")
					cmds = append(cmds, m.activeForm.Init())
				}

				m.state = stateAskForPGPKey
				m.activeForm = genericInputForm("Ange GPG E-post/ID:", "pgp")
				cmds = append(cmds, m.activeForm.Init())

			}
			// Check if it is already sealed
		}

	case stateAskIfUserWantToNukeSealedDocumentForSealedDocument:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			m.force = m.activeForm.GetBool("action")

			if m.force == false {
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			}
			// TODO DETTA KAN VARA HELT FEL
			if m.force == true {
				m.state = stateChooseDocumentToSeal
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			}
		}

	case stateAskForPGPKey:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			pgpKey := m.activeForm.GetString("pgp")
			err := app.DoSeal(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToSeal, "källor"), pgpKey, m.force)
			if err != nil {
				fmt.Println(err)
			}
			m.force = false

			m.state = stateMainMenu
			m.mainMenu = createMainMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
		}
	case stateChooseDocumentToBuild:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.force {
			app.DoBuild(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToBuild, "källor"), m.force)
			m.force = false

			m.state = stateMainMenu
			m.mainMenu = createSettingsMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
		}

		if m.activeForm.State == huh.StateCompleted {
			m.selectedDocumentToBuild = m.activeForm.GetString("document")
			switch m.selectedDocumentToBuild {
			case "back":
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			default:
				sigPath := filepath.Join(filepath.Dir(m.selectedDocumentToBuild), "arkiv", "ATTESTATION.md.sig")

				if _, err := os.Stat(sigPath); err == nil {
					m.state = stateAskIfUserWantToNukeSealedDocument
					m.activeForm = askGenericYesOrNowForm("⚠️ Arkivet/Avtalet är förseglat!", " Byggs det om raderas signaturen. Fortsätta?", "action")
					cmds = append(cmds, m.activeForm.Init())
				}
				app.DoBuild(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToBuild, "källor"), m.force)
				m.force = false

				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			}
		}

	case stateAskIfUserWantToNukeSealedDocument:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			m.force = m.activeForm.GetBool("action")

			if m.force == false {
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			}
			// TODO DETTA KAN VARA HELT FEL
			if m.force == true {
				m.state = stateChooseDocumentToBuild
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			}
		}
	case stateEditZip:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			m.cfg.Settings.CreateZIP = m.activeForm.GetBool("zip")
			if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}

			m.settingsMenu = createSettingsMenuForm()
			m.state = stateSettingsMenu
			cmds = append(cmds, m.settingsMenu.Init())
		}

	case stateEditOts:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			m.cfg.Settings.UseOpenTimeStamps = m.activeForm.GetBool("ots")
			if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}
			m.settingsMenu = createSettingsMenuForm()
			m.state = stateSettingsMenu
			cmds = append(cmds, m.settingsMenu.Init())
		}

	case stateChooseOrgForDoc:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if keyMsg, ok := keyMsg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEsc {
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
				return m, tea.Batch(cmds...)
			}
		}

		if m.activeForm.State == huh.StateCompleted {
			action := m.activeForm.GetString("selected_org")
			switch action {
			case "create_new":
				m.activeForm = askForNewOrgDetailsForm()
				m.state = stateCreateOrganization
				cmds = append(cmds, m.activeForm.Init())
			case "back":
				m.state = stateMainMenu
				m.mainMenu = createMainMenuForm()
				cmds = append(cmds, m.mainMenu.Init())
			default:
				m.selectedOrg = m.cfg.Foreningar[action]
				m.state = stateChooseDocumentTypeForm
				m.activeForm = chooseDocumentTypeForm()
				cmds = append(cmds, m.activeForm.Init())
			}
		}

	case stateCreateOrganization:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			newOrg := app.Association{
				Id:        m.activeForm.GetString("org_id"),
				Name:      m.activeForm.GetString("org_name"),
				OrgNummer: m.activeForm.GetString("org_number"),
			}

			err := app.CreateOrganization(m.cwd, newOrg.Id, newOrg.Name, newOrg.OrgNummer)
			if err != nil {
				return nil, nil
			}

			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
			m.state = stateChooseOrgForDoc
			cmds = append(cmds, m.activeForm.Init())
		}

	case stateFirstRun:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {

			// If true: create
			// If false: EXIT
			if m.activeForm.GetBool("action") {
				toolingDir := filepath.Join(m.cwd, ".tooling")

				if err := app.FirstTimeRun(toolingDir, m.cwd, assets.Files); err != nil {
					panic("Couldn't create things")
				}

			} else {
				return m, tea.Quit
			}
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.state = stateMainMenu
			m.mainMenu = createMainMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
		}

	case stateChooseDocumentTypeForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if keyMsg, ok := keyMsg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEsc {
				m.state = stateChooseOrgForDoc
				*m.cfg, _ = app.LoadConfig(m.cwd)
				m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
				cmds = append(cmds, m.activeForm.Init())
				return m, tea.Batch(cmds...)
			}
		}

		if m.activeForm.State == huh.StateCompleted {
			documentType := m.activeForm.GetString("document_type")
			switch documentType {
			case "protokoll":
				m.selectedDocumentType = documentType

				m.activeForm = chooseBodyForm(m.selectedOrg)
				m.state = stateChooseBodyForm
				cmds = append(cmds, m.activeForm.Init())
			case "styrdokument":
				m.selectedDocumentType = documentType

				m.state = stateChooseGoverningDocumentTypeForm
				m.activeForm = chooseGoverningDocumentTypeForm(m.cwd, m.selectedOrg)
				cmds = append(cmds, m.activeForm.Init())
			case "other":
				m.selectedDocumentType = documentType
				m.state = stateChooseOtherDocumentType
				m.activeForm = chooseOtherDocumentTypeForm(m.cwd, m.selectedOrg)
				cmds = append(cmds, m.activeForm.Init())

			}
		}
	case stateChooseOtherDocumentType:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			selectedDocumentType := m.activeForm.GetString("selected_document_type")
			switch selectedDocumentType {
			case "create_new":
				m.state = stateChooseOtherDocumentCategoryNameForm
				m.activeForm = genericInputForm("Kategorins namn (t.ex. Avtal, Motioner, Propositioner)", "name")
				cmds = append(cmds, m.activeForm.Init())

			default:
				m.grundaktCategory = selectedDocumentType
				m.state = stateCreateOtherDocumentTypeForm
				m.activeForm = genericInputForm("Dokumentets/Filens namn (t.ex Motion angående xx)", "name")
				cmds = append(cmds, m.activeForm.Init())
			}

		}
	case stateChooseOtherDocumentCategoryNameForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			name := m.activeForm.GetString("name")

			_ = os.MkdirAll(filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", m.grundaktCategory, name), os.ModePerm)
			m.state = stateChooseOtherDocumentType
			m.activeForm = chooseOtherDocumentTypeForm(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())

		}

	case stateCreateOtherDocumentTypeForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			name := m.activeForm.GetString("name")
			_ = app.CreateOtherGoverningDocuments(m.cwd, m.selectedOrg.Id, m.grundaktCategory, name)

			m.state = stateMainMenu
			m.mainMenu = createMainMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
		}
	case stateChooseGoverningDocumentTypeForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			selectedDocumentType := m.activeForm.GetString("selected_document_type")
			switch selectedDocumentType {
			case "create_new":
				// SKAPA NY STYRDOKUMENT
				m.state = stateCreateNewGoverningDocumentTypeForm
				m.activeForm = genericInputForm("Kategorins namn (t.ex. Policy, Reglemente, Stadgar):", "name")
				cmds = append(cmds, m.activeForm.Init())

			default:
				m.selectedGoverningDocType = selectedDocumentType
				m.state = stateAskGoverningDocumentNameForm
				m.activeForm = genericInputForm("Dokumentets/Filens namn (t.ex. IT-policy):", "name")
				cmds = append(cmds, m.activeForm.Init())
			}
		}
	case stateCreateNewGoverningDocumentTypeForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			name := m.activeForm.GetString("name")
			err := os.MkdirAll(filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", "Styrdokument", name), os.ModePerm)
			if err != nil {
				panic(err)
			}
			m.state = stateChooseGoverningDocumentTypeForm
			m.activeForm = chooseGoverningDocumentTypeForm(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())

		}
	case stateAskGoverningDocumentNameForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			name := m.activeForm.GetString("name")
			_ = app.CreateGuidanceDocuments(m.cwd, m.selectedOrg.Id, m.selectedGoverningDocType, name)

			m.mainMenu = createMainMenuForm()
			m.state = stateMainMenu
			cmds = append(cmds, m.mainMenu.Init())
		}
	case stateChooseBodyForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if keyMsg, ok := keyMsg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEsc {
				m.state = stateChooseDocumentTypeForm
				m.activeForm = chooseDocumentTypeForm()
				cmds = append(cmds, m.activeForm.Init())
				return m, tea.Batch(cmds...)
			}
		}

		if m.activeForm.State == huh.StateCompleted {
			body := m.activeForm.GetString("selected_body")
			switch body {

			case "create_new":
				m.state = stateCreateBody
				m.activeForm = genericInputForm("Organets namn (t.ex. utbildningsutskott)", "name")
				cmds = append(cmds, m.activeForm.Init())
			default:
				m.selectedBody = body
				m.state = stateAskDateForm
				m.activeForm = genericInputForm("Datum (ÅÅÅÅ-MM-DD):", "date")
				cmds = append(cmds, m.activeForm.Init())
			}
		}
	case stateCreateBody:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)

		if m.activeForm.State == huh.StateCompleted {
			body := m.activeForm.GetString("name")
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.selectedOrg.Body = append(m.selectedOrg.Body, body)
			m.cfg.Foreningar[m.selectedOrg.Id] = m.selectedOrg
			_ = app.SaveConfig(m.cwd, *m.cfg)

			m.state = stateChooseBodyForm
			m.activeForm = chooseBodyForm(m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())
		}
	case stateAskDateForm:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			date := m.activeForm.GetString("date")
			m.date = date

			_ = app.CreateProtokoll(m.cwd, m.selectedOrg.Id, m.selectedBody, m.date)

			m.state = stateMainMenu
			m.mainMenu = createMainMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
		}
	case stateConfirmTemplateUpdate:
		form, cmd := m.activeForm.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		cmds = append(cmds, cmd)
		if m.activeForm.State == huh.StateCompleted {
			confirm := m.activeForm.GetBool("confirm")
			if confirm {
				_ = app.UpdateTemplates(m.cwd, assets.Files)
			}
			m.settingsMenu = createSettingsMenuForm()
			m.state = stateSettingsMenu
			cmds = append(cmds, m.settingsMenu.Init())
		}
	default:
		panic("unhandled default case")
	}
	return m, tea.Batch(cmds...)
}

// View is a function that switches the modelView depending on the state.
func (m mainModel) View() string {
	switch m.state {
	case stateMainMenu:
		return m.mainMenu.View()
	case stateSettingsMenu:
		return m.settingsMenu.View()
	case stateEditZip, stateEditOts, stateChooseOrgForDoc, stateCreateOrganization, stateFirstRun,
		stateChooseDocumentTypeForm, stateChooseBodyForm, stateCreateBody, stateAskDateForm,
		stateChooseGoverningDocumentTypeForm, stateCreateNewGoverningDocumentTypeForm, stateAskGoverningDocumentNameForm,
		stateChooseOtherDocumentType, stateChooseOtherDocumentCategoryNameForm, stateCreateOtherDocumentTypeForm,
		stateChooseOrgForBuild, stateChooseDocumentToBuild, stateAskIfUserWantToNukeSealedDocument,
		stateChooseOrgForSeal, stateChooseDocumentToSeal, stateAskForPGPKey, stateAskIfUserWantToNukeSealedDocumentForSealedDocument,
		stateConfirmTemplateUpdate:
		if m.activeForm == nil {
			return "Laddar..."
		}
		return m.activeForm.View()
	default:
		return "Eeeh något gick fel"
	}
}

func StartTUI() error {
	cwd, _ := os.Getwd()
	cfg := app.Config{}

	initialModel := mainModel{
		state:        stateMainMenu,
		cwd:          cwd,
		cfg:          &cfg,
		mainMenu:     createMainMenuForm(),
		settingsMenu: createSettingsMenuForm(),
		activeForm:   createFirstRunForm(),
	}

	if isFirstRun() {
		initialModel.state = stateFirstRun
	} else {
		cfg, _ = app.LoadConfig(cwd)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func runSettingsFlow(context *AppContext) {
	action, err := showSettingsMenu()
	if err != nil || action == "exit" {
	}

	switch action {
	case "ny_org":
		// Ev kolla erros här
		orgDetails, _ := askForNewOrgDetails()

		err := app.CreateOrganization(context.Cwd, orgDetails.ID, orgDetails.Name, orgDetails.Number)
		if err != nil {
			fmt.Println(err)
		}

		context.Cfg.Foreningar[orgDetails.ID] = app.Association{Name: orgDetails.Name, OrgNummer: orgDetails.Number, Body: []string{"styrelsen", "årsmöte"}}

		// TODO Change SaveConfig to tage &Cfg
		if err := app.SaveConfig(context.Cwd, *context.Cfg); err != nil {
			fmt.Println("Error occured while saving config")
			pausePrompt()
			return
		}
	case "redigera_org":
		selectedOrg, err := askForAssociationForm(context)
		if err != nil {
			fmt.Println(err)
		}

		originalOrg := context.Cfg.Foreningar[selectedOrg]

		detailsToEdit := OrgDetails{
			ID:     selectedOrg,
			Name:   originalOrg.Name,
			Number: originalOrg.OrgNummer,
		}

		updatedDetails, err := askForNewDetailsForAssociationForm(detailsToEdit)
		if err != nil {
			fmt.Println(err)
		}

		originalOrg.Name = updatedDetails.Name
		originalOrg.OrgNummer = updatedDetails.Number

		context.Cfg.Foreningar[selectedOrg] = originalOrg

		_ = app.SaveConfig(context.Cwd, *context.Cfg)
	case "toggle_zip":
		newValue, err := askToggleSetting("Skapa automatiskt ZIP-arkiv?", context.Cfg.Settings.CreateZIP)
		if err != nil {
			fmt.Println(err)
		}

		context.Cfg.Settings.CreateZIP = newValue

		if err := app.SaveConfig(context.Cwd, *context.Cfg); err != nil {
			fmt.Println("Error occured while saving config")
		} else {
			fmt.Println("✅ Inställningen sparad!")
		}
	case "toggle_ots":
		newValue, err := askToggleSetting("Använd OpenTimeStamps?", context.Cfg.Settings.UseOpenTimeStamps)
		if err != nil {
			fmt.Println(err)
		}

		context.Cfg.Settings.UseOpenTimeStamps = newValue

		if err := app.SaveConfig(context.Cwd, *context.Cfg); err != nil {
			fmt.Println("Error occured while saving config")
		} else {
			fmt.Println("✅ Inställningen sparad!")
		}
	}
}
