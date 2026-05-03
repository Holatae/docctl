package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func pausePrompt() {
	fmt.Println("\n[ Tryck på Enter för att återgå till menyn... ]")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func isFirstRun() bool {
	cwd, _ := os.Getwd()
	toolingDir := filepath.Join(cwd, ".tooling")

	if _, err := os.Stat(toolingDir); os.IsNotExist(err) {
		return true
	}
	return false
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
