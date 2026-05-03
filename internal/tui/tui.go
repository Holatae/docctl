package tui

import (
	"docctl/internal/app"
	"docctl/internal/assets"
	"fmt"
	"os"
	"path/filepath"

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
	stateChooseOrgForEdit
	stateChooseOrgForDoc
	stateCreateOrganization
	stateFirstRun
)

type mainModel struct {
	state appState
	cwd   string
	cfg   *app.Config

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

			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
			cmds = append(cmds, m.activeForm.Init())

			m.mainMenu = createMainMenuForm()
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
				tea.Quit()
			}
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.state = stateMainMenu
			m.mainMenu = createMainMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
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
	case stateEditZip, stateEditOts, stateChooseOrgForDoc, stateCreateOrganization, stateFirstRun:
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
