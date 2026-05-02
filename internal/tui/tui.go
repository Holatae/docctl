package tui

import (
	"docctl/internal/app"
	"fmt"
	"os"
)

type AppContext struct {
	Cwd string
	Cfg *app.Config
}

func StartTUI() {

	cwd, _ := os.Getwd()
	cfg, _ := app.LoadConfig(cwd)

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
