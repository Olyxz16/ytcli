package cmdx

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/oauth"
	"github.com/Olyxz16/tkt/internal/provider/youtrack"
	"github.com/Olyxz16/tkt/internal/render"
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
	Use:   "login [provider]",
	Short: "Authenticate with a remote provider",
	Long: `Authenticate with a remote provider.

By default, this prompts for a permanent token (recommended for CLI use).
Use --oauth to authenticate via OAuth 2.0 implicit grant instead. OAuth
requires the instance to have a Hub service configured and a client_id.
`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		localCfg, _, _ := config.LoadLocal()
		providerName := ""
		providerURL := ""

		if localCfg != nil && localCfg.Provider.Name != "" {
			providerName = localCfg.Provider.Name
			providerURL = localCfg.Provider.URL
		}
		// Backwards compatibility
		if providerName == "" && localCfg != nil && localCfg.Instance != "" {
			providerName = localCfg.Instance
			providerURL = localCfg.InstanceURL
		}

		if len(args) > 0 {
			providerName = args[0]
		}
		if providerName == "" {
			fmt.Fprintln(os.Stderr, "Error: no provider specified. Provide a provider name, or configure with: tkt config set provider.name <name>")
			os.Exit(1)
		}

		if providerURL == "" {
			fmt.Fprintf(os.Stderr, "Error: no provider URL configured for %q. Run: tkt config set provider.url <url>\n", providerName)
			os.Exit(1)
		}

		var token string

		if authOAuthFlag {
			token = runOAuthFlow(cmd.Context(), providerName, providerURL)
		} else {
			token = runTokenFlow(cmd.Context(), providerName, providerURL)
		}

		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: no token obtained")
			os.Exit(1)
		}

		if err := config.SetToken(providerName, token); err != nil {
			if err := config.SetTokenFile(providerName, token); err != nil {
				handleError(err)
			}
			fmt.Println("Token saved to fallback credentials file (keyring unavailable)")
		} else {
			fmt.Println("Token saved to keyring")
		}
	},
}

func runTokenFlow(ctx context.Context, providerName, providerURL string) string {
	token := authTokenFlag
	if token == "" {
		fmt.Print("Enter permanent token: ")
		fmt.Scanln(&token)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: token is required")
		os.Exit(1)
	}

	// Verify token by pinging the provider
	prov := youtrack.NewProvider(providerURL, token)
	if err := prov.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: token verification failed: %s\n", err)
	}

	return token
}

func runOAuthFlow(ctx context.Context, providerName, providerURL string) string {
	hubURL := oauth.DiscoverHubURL(providerURL)
	fmt.Printf("Auto-discovered Hub URL: %s\n", hubURL)

	// For now, OAuth still requires client_id to be configured per-provider
	fmt.Fprintf(os.Stderr, "OAuth requires a client_id for provider %q.\n", providerName)
	fmt.Fprintln(os.Stderr, "Register this CLI as a service in Hub (Admin > Services) and configure it.")
	os.Exit(1)

	// This path is kept for future implementation with provider-specific OAuth config
	return ""
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