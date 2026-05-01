package cmdx

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
)

var configSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive global configuration setup",
	Run: func(cmd *cobra.Command, args []string) {
		global, err := config.LoadGlobal()
		if err != nil {
			handleError(err)
		}

		var instanceName, instanceURL, token string

		// Instance name
		huh.NewInput().
			Title("Instance name").
			Placeholder("work").
			Value(&instanceName).
			Run()

		if instanceName == "" {
			instanceName = "default"
		}

		// Instance URL
		huh.NewInput().
			Title("YouTrack URL").
			Placeholder("https://company.youtrack.cloud").
			Value(&instanceURL).
			Run()

		if instanceURL == "" {
			fmt.Fprintln(os.Stderr, "URL is required")
			os.Exit(1)
		}
		instanceURL = strings.TrimSuffix(instanceURL, "/")

		// Token
		huh.NewInput().
			Title("Permanent token").
			EchoMode(huh.EchoModePassword).
			Value(&token).
			Run()

		if token == "" {
			fmt.Fprintln(os.Stderr, "Token is required")
			os.Exit(1)
		}

		// Save
		if global.Instances == nil {
			global.Instances = make(map[string]config.InstanceConfig)
		}
		global.Instances[instanceName] = config.InstanceConfig{URL: instanceURL}

		if err := config.SaveGlobal(global); err != nil {
			handleError(err)
		}

		if err := config.SetToken(instanceName, token); err != nil {
			if err := config.SetTokenFile(instanceName, token); err != nil {
				handleError(err)
			}
			fmt.Println("Token saved to fallback credentials file")
		} else {
			fmt.Println("Token saved to keyring")
		}

		fmt.Printf("Instance %q configured successfully.\n", instanceName)
		fmt.Println("To bind this project to this instance, run: ytcli config set instance " + instanceName)
	},
}

func init() {
	configCmd.AddCommand(configSetupCmd)
}