package tui

import (
	"bufio"
	"docctl/internal/app"
	"docctl/internal/assets"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func pausePrompt() {
	fmt.Println("\n[ Tryck på Enter för att återgå till menyn... ]")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

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

	if err := app.FirstTimeRun(toolingDir, cwd, assets.Files); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
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

func runForm(f *huh.Form) error {
	km := huh.NewDefaultKeyMap()
	km.Quit.SetKeys("esc", "ctrl+c")

	// tea.WithAltScreen() är the holy grail här. Den ritar formuläret på en ren skärm,
	// och återskapar din terminalhistorkik perfekt när den stängs!
	return f.WithKeyMap(km).WithProgramOptions(tea.WithAltScreen()).Run()
}
