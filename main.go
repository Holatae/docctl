package main

import (
	"docctl/internal/app"
	"docctl/internal/assets"
	"docctl/internal/tui"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// ------------------------------------------
// PAUSFUNKTIONEN (För att du ska hinna se loggarna)
// ------------------------------------------

// ============================================
// FIRST RUN / ONBOARDING
// ============================================

// ============================================
// LOGIK MOTOR (doBuild & doSeal)
// ============================================

// ============================================
// CLI KOMMANDON
// ============================================
var (
	forceBuild bool
	forceSeal  bool
)

var rootCmd = &cobra.Command{
	Use: "docctl",
	Run: func(cmd *cobra.Command, args []string) {
		tui.StartTUI()
	},
}

var buildCmd = &cobra.Command{
	Use:  "build [org] [sökväg]",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := app.DoBuild(args[1], forceBuild)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var sealCmd = &cobra.Command{
	Use:  "seal [sökväg]",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gpgKey, _ := cmd.Flags().GetString("key")
		err := app.DoSeal(args[0], gpgKey, forceSeal)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var orgCmd = &cobra.Command{
	Use:  "org [add] [short name] [long name] [org-number]",
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		if args[0] == "add" {
			err := app.CreateOrganization(cwd, args[1], args[2], args[3])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
	Example: "doctl org add TestOrg \"Test Organization\" 802000-1234",
	Short:   "Adds a new organization",
}

var initCmd = &cobra.Command{
	Use: "init",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		toolingDir := filepath.Join(cwd, ".tooling")
		err := app.FirstTimeRun(toolingDir, cwd, assets.Files)
		if err != nil {
			_ = fmt.Errorf("error occurred while initializing tooling: %v", err)
			os.Exit(1)
		}
	},
	Short: "Use for initialization of directory",
}

func main() {
	sealCmd.Flags().StringP("key", "k", "", "GPG Key")
	_ = sealCmd.MarkFlagRequired("key")
	buildCmd.Flags().BoolVarP(&forceBuild, "force", "f", false, "Tvinga ombyggnad")
	sealCmd.Flags().BoolVarP(&forceSeal, "force", "f", false, "Tvinga omförsegling")

	// Init behövs inte längre via argument nu när TUI ritar upp miljön så bra,
	// men vi behåller rootCmd för gränssnittet.
	rootCmd.AddCommand(buildCmd, sealCmd, initCmd, orgCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
