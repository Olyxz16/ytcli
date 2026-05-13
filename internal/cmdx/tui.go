package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/service"
	"github.com/Olyxz16/tkt/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		localCfg, _, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		privateCfg, _, err := config.LoadLocalPrivate()
		if err != nil {
			handleError(err)
		}
		merged, err := config.Resolve(localCfg, privateCfg)
		if err != nil {
			handleError(err)
		}

		var svc *service.Service
		providerErr := ""
		if merged.ProviderURL != "" {
			svc, err = service.NewService(merged)
			if err != nil {
				providerErr = err.Error()
				svc = nil
			}
		}

		app := tui.NewApp(svc, merged, providerErr)
		if err := app.Run(); err != nil {
			fmt.Println("TUI error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
