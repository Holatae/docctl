package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func StartTUI() {
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
