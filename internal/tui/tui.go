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
)

type mainModel struct {
	state appState
	cwd   string
	cfg   *app.Config

	// Formulär
	mainMenu     *huh.Form
	settingsMenu *huh.Form
}

func (m mainModel) Init() tea.Cmd {
	return nil
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
		form, cmd := m.mainMenu.Update(keyMsg)
		if f, ok := form.(*huh.Form); ok {
			m.mainMenu = f
		}
		cmds = append(cmds, cmd)

		// Blev menyn klar? (tryckte användaren enter?)
		if m.mainMenu.State == huh.StateCompleted {
			action := m.mainMenu.GetString("action")

			if action == "settings" {
				m.state = stateSettingsMenu
			}
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
	default:
		return "Eeeh något gick fel"
	}
}

func createMainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("action"). // Sätter en nyckel vi kan hämta med GetString()
				Title("Huvudmeny").
				Options(
					huh.NewOption("⚙️ Inställningar", "settings"),
					huh.NewOption("❌ Avsluta", "exit"),
				),
		),
	)
}

func StartTUI() error {

	cwd, _ := os.Getwd()
	cfg, _ := app.LoadConfig(cwd)

	initialModel := mainModel{
		state:    stateMainMenu,
		cwd:      cwd,
		cfg:      &cfg,
		mainMenu: createMainMenuForm(),
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
