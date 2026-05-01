package cmdx

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/oauth"
	"github.com/Olyxz16/ytcli/internal/render"
)

var (
	authTokenFlag string
	authOAuthFlag bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login [instance]",
	Short: "Authenticate with a YouTrack instance",
	Long: `Authenticate with a YouTrack instance.

By default, this prompts for a permanent token (recommended for CLI use).
Use --oauth to authenticate via OAuth 2.0 implicit grant instead. OAuth
requires the instance to have a Hub service configured and a client_id.
`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		global, err := config.LoadGlobal()
		if err != nil {
			handleError(err)
		}

		instance := global.DefaultInstance
		if instance == "" && len(global.Instances) == 1 {
			for name := range global.Instances {
				instance = name
			}
		}
		localCfg, _, _ := config.LoadLocal()
		if localCfg != nil && localCfg.Instance != "" {
			instance = localCfg.Instance
		}
		if len(args) > 0 {
			instance = args[0]
		}
		if instance == "" {
			instance = "default"
		}

		inst, ok := global.Instances[instance]
		if !ok {
			fmt.Fprintf(os.Stderr, "Instance %q not found in config.\n", instance)
			fmt.Fprintln(os.Stderr, "Add it first with: ytcli config set instances.<name>.url <url>")
			os.Exit(1)
		}

		var token string

		if authOAuthFlag {
			token = runOAuthFlow(cmd.Context(), instance, inst)
		} else {
			token = runTokenFlow(cmd.Context(), instance, inst)
		}

		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: no token obtained")
			os.Exit(1)
		}

		if err := config.SetToken(instance, token); err != nil {
			if err := config.SetTokenFile(instance, token); err != nil {
				handleError(err)
			}
			fmt.Println("Token saved to fallback credentials file (keyring unavailable)")
		} else {
			fmt.Println("Token saved to keyring")
		}
	},
}

func runTokenFlow(ctx context.Context, instance string, inst config.InstanceConfig) string {
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
	if _, err := apiClient.Me(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: token verification failed: %s\n", err)
	}

	return token
}

func runOAuthFlow(ctx context.Context, instance string, inst config.InstanceConfig) string {
	hubURL := inst.HubURL
	if hubURL == "" {
		hubURL = oauth.DiscoverHubURL(inst.URL)
		fmt.Printf("Auto-discovered Hub URL: %s\n", hubURL)
	}

	clientID := inst.ClientID
	if clientID == "" {
		fmt.Fprintf(os.Stderr, "Error: OAuth requires a client_id for instance %q.\n", instance)
		fmt.Fprintln(os.Stderr, "Register this CLI as a service in Hub (Admin > Services) and set the ID:")
		fmt.Fprintf(os.Stderr, "  ytcli config set instances.%s.client_id <id>\n", instance)
		os.Exit(1)
	}

	scope := inst.Scope
	if scope == "" {
		// Try to discover YouTrack service ID from Hub
		token, err := config.GetToken(instance)
		if err == nil && token != "" {
			services, err := oauth.ListServices(ctx, hubURL, token)
			if err == nil {
				scope = oauth.FindYouTrackServiceID(services)
			}
		}
		if scope == "" {
			fmt.Fprintf(os.Stderr, "Error: OAuth requires a scope (YouTrack service ID) for instance %q.\n", instance)
			fmt.Fprintln(os.Stderr, "Set it with:")
			fmt.Fprintf(os.Stderr, "  ytcli config set instances.%s.scope <youtrack-service-id>\n", instance)
			fmt.Fprintln(os.Stderr, "Or authenticate with a permanent token first so we can auto-discover it:")
			fmt.Fprintln(os.Stderr, "  ytcli auth login --token <token>")
			os.Exit(1)
		}
		fmt.Printf("Auto-discovered YouTrack service ID: %s\n", scope)
	}

	result, err := oauth.ImplicitFlow(ctx, oauth.FlowConfig{
		HubURL:   hubURL,
		ClientID: clientID,
		Scope:    scope,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "OAuth flow failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("OAuth token obtained (expires in %d seconds).\n", result.ExpiresIn)
	return result.AccessToken
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
	authLoginCmd.Flags().BoolVar(&authOAuthFlag, "oauth", false, "Use OAuth 2.0 implicit grant instead of permanent token")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authWhoamiCmd)
	rootCmd.AddCommand(authCmd)
}
