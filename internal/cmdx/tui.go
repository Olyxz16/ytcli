package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI (not yet implemented)",
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}
		app := tui.NewApp(svc)
		if err := app.Run(); err != nil {
			fmt.Println("TUI error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
