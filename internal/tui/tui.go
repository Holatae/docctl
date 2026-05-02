package tui

import (
	"docctl/internal/app"
	"fmt"
	"os"

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
	stateSystemSettings
	stateEditZip
	stateEditOts
)

type mainModel struct {
	state appState
	cwd   string
	cfg   *app.Config

	// Formulär
	mainMenu          *huh.Form
	settingsMenu      *huh.Form
	activeSettingForm *huh.Form
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

			cmds = append(cmds, m.settingsMenu.Init())

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

			cmds = append(cmds, m.settingsMenu.Init())

			m.settingsMenu = createSettingsMenuForm()
		}
	}
	if m.settingsMenu.State == huh.StateCompleted {
		action := m.settingsMenu.GetString("action")
		switch action {
		case "toggle_zip":
			m.state = stateEditZip
			m.activeSettingForm = createZipForm(m.cfg.Settings.CreateZIP)

			cmds = append(cmds, m.activeSettingForm.Init())

			m.settingsMenu = createSettingsMenuForm()
		case "toggle_ots":
			m.state = stateEditOts
			m.activeSettingForm = createOpenTimeStampsForm(m.cfg.Settings.UseOpenTimeStamps)

			cmds = append(cmds, m.activeSettingForm.Init())

			m.settingsMenu = createSettingsMenuForm()
		case "back":
			m.state = stateMainMenu

			cmds = append(cmds, m.mainMenu.Init())

			m.settingsMenu = createSettingsMenuForm()
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) Update(keyMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Check for ctrl + c
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
		form, cmd := m.activeSettingForm.Update(keyMsg)

		if f, ok := form.(*huh.Form); ok {
			m.activeSettingForm = f
		}

		cmds = append(cmds, cmd)

		if m.activeSettingForm.State == huh.StateCompleted {
			m.cfg.Settings.CreateZIP = m.activeSettingForm.GetBool("zip")

			if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
				fmt.Println("ERROR DELUX")
			}
			m.state = stateSettingsMenu
			cmds = append(cmds, m.settingsMenu.Init())
			m.settingsMenu = createSettingsMenuForm()

			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

	case stateEditOts:
		form, cmd := m.activeSettingForm.Update(keyMsg)

		if f, ok := form.(*huh.Form); ok {
			m.activeSettingForm = f
		}

		cmds = append(cmds, cmd)

		if m.activeSettingForm.State == huh.StateCompleted {
			m.cfg.Settings.UseOpenTimeStamps = m.activeSettingForm.GetBool("ots")

			if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
				fmt.Println("ERROR DELUX")
			}
			m.state = stateSettingsMenu

			cmds = append(cmds, m.settingsMenu.Init())
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

	default:
		panic("unhandled default case")
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) View() string {
	switch m.state {
	case stateMainMenu:
		return m.mainMenu.View()
	case stateSettingsMenu:
		return m.settingsMenu.View()
	case stateEditZip:
		return m.activeSettingForm.View()
	case stateEditOts:
		return m.activeSettingForm.View()
	default:
		return "Eeeh något gick fel"
	}
}

func StartTUI() error {

	cwd, _ := os.Getwd()
	cfg, _ := app.LoadConfig(cwd)

	initialModel := mainModel{
		state:        stateMainMenu,
		cwd:          cwd,
		cfg:          &cfg,
		mainMenu:     createMainMenuForm(),
		settingsMenu: createSettingsMenuForm(),
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil

	/*
		context := &AppContext{
			Cwd: cwd,
			Cfg: &cfg,
		}

		_ = checkFirstRun()

		for {

			action, err := showMainMenu()

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
				runSettingsFlow(context)
			}


		}

	*/
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
