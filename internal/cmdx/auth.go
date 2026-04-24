package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/render"
)

var (
	authTokenFlag string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login [instance]",
	Short: "Authenticate with a YouTrack instance",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		instance := "default"
		if len(args) > 0 {
			instance = args[0]
		}

		global, err := config.LoadGlobal()
		if err != nil {
			handleError(err)
		}

		inst, ok := global.Instances[instance]
		if !ok {
			fmt.Fprintf(os.Stderr, "Instance %q not found in config.\n", instance)
			fmt.Fprintln(os.Stderr, "Add it first with: ytcli config set instances.<name>.url <url>")
			os.Exit(1)
		}

		token := authTokenFlag
		if token == "" {
			fmt.Print("Enter permanent token: ")
			fmt.Scanln(&token)
		}
		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: token is required")
			os.Exit(1)
		}

		// Verify token by calling /users/me
		apiClient := api.NewClient(inst.URL, token)
		ctx := cmd.Context()
		if _, err := apiClient.Me(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: token verification failed: %s\n", err)
		}

		if err := config.SetToken(instance, token); err != nil {
			// Fallback to file
			if err := config.SetTokenFile(instance, token); err != nil {
				handleError(err)
			}
			fmt.Println("Token saved to fallback credentials file (keyring unavailable)")
		} else {
			fmt.Println("Token saved to keyring")
		}
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user",
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		me, err := svc.Me(cmd.Context())
		if err != nil {
			handleError(err)
		}

		if err := render.User(me, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	authLoginCmd.Flags().StringVar(&authTokenFlag, "token", "", "Permanent token (skip interactive prompt)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authWhoamiCmd)
	rootCmd.AddCommand(authCmd)
}
