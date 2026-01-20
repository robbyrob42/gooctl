package cmd

import (
	"fmt"
	"os"

	"github.com/dorkitude/gooctl/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gooctl",
	Short: "A CLI for Google Workspace (Gmail & Calendar)",
	Long: ui.TitleStyle.Render("gooctl") + `
A beautiful CLI for managing Google Workspace.

` + ui.SubtleStyle.Render("Commands:") + `
  auth      Authenticate with Google
  mail      Search and manage emails
  calendar  View and manage calendar events
`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Error_(err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
