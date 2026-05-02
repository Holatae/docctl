package tui

import (
	"fmt"
)

func StartTUI() {
	_ = checkFirstRun()

	for {

		action, err := mainMenu()

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
