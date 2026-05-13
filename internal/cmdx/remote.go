package cmdx

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/oauth"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/provider/youtrack"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/store"
	"github.com/Olyxz16/tkt/internal/sync"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Interact with the remote provider",
	Long:  `Authentication, sync, and status for the remote issue tracker.`,
}

// ── remote auth ──────────────────────────────────────────────────────────────

var (
	remoteAuthToken string
	remoteAuthOAuth bool
)

var remoteAuthCmd = &cobra.Command{
	Use:   "auth [provider]",
	Short: "Authenticate with the remote provider",
	Long: `Authenticate with a remote provider using a permanent token (default) or OAuth.

Without arguments, uses the provider configured in .tktrc.yml.
With an argument, authenticates the named provider directly.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		localCfg, _, _ := config.LoadLocal()
		providerName := ""
		providerURL := ""

		if localCfg != nil && localCfg.Provider.Name != "" {
			providerName = localCfg.Provider.Name
			providerURL = localCfg.Provider.URL
		}
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
		if remoteAuthOAuth {
			token = runRemoteOAuth(cmd.Context(), providerName, providerURL)
		} else {
			token = runRemoteToken(cmd.Context(), providerName, providerURL)
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

func runRemoteToken(ctx context.Context, providerName, providerURL string) string {
	token := remoteAuthToken
	if token == "" {
		fmt.Print("Enter permanent token: ")
		fmt.Scanln(&token)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: token is required")
		os.Exit(1)
	}
	prov := youtrack.NewProvider(providerURL, token)
	if err := prov.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: token verification failed: %s\n", err)
	}
	return token
}

func runRemoteOAuth(ctx context.Context, providerName, providerURL string) string {
	hubURL := oauth.DiscoverHubURL(providerURL)
	fmt.Printf("Auto-discovered Hub URL: %s\n", hubURL)
	fmt.Fprintf(os.Stderr, "OAuth requires a client_id for provider %q.\n", providerName)
	fmt.Fprintln(os.Stderr, "Register this CLI as a service in Hub (Admin > Services) and configure it.")
	os.Exit(1)
	return ""
}

// ── remote whoami ────────────────────────────────────────────────────────────

var remoteWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current remote user",
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

// ── remote pull ──────────────────────────────────────────────────────────────

var remotePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote issues into the local database",
	Run: func(cmd *cobra.Command, args []string) {
		db, prov, cfg := requireDBAndProvider()
		defer db.Close()

		mgr := sync.NewManager(db, prov, cfg)
		result, err := mgr.Pull(cmd.Context())
		if err != nil {
			handleError(err)
		}
		renderSyncResult(result, "pull")
	},
}

// ── remote push ──────────────────────────────────────────────────────────────

var remotePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local pending changes to the remote provider",
	Run: func(cmd *cobra.Command, args []string) {
		db, prov, cfg := requireDBAndProvider()
		defer db.Close()

		mgr := sync.NewManager(db, prov, cfg)
		result, err := mgr.Push(cmd.Context())
		if err != nil {
			handleError(err)
		}
		renderSyncResult(result, "push")
	},
}

// ── remote status ────────────────────────────────────────────────────────────

var remoteStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status and queue summary",
	Run: func(cmd *cobra.Command, args []string) {
		if !store.IsInitialized() {
			fmt.Fprintln(os.Stderr, "Error: no local project found. Run: tkt init")
			os.Exit(1)
		}
		db, err := store.Open()
		if err != nil {
			handleError(err)
		}
		defer db.Close()
		showSyncStatus(db)
	},
}

func showSyncStatus(db *sql.DB) {
	rows, err := db.Query(`
		SELECT sync_status, COUNT(*) FROM issues GROUP BY sync_status
		UNION ALL
		SELECT 'total', COUNT(*) FROM issues`)
	if err != nil {
		handleError(err)
	}
	defer rows.Close()

	if getOutputMode() == render.OutputJSON {
		statusMap := make(map[string]int)
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			statusMap[status] = count
		}
		pending, failed, completed, _ := store.QueueCount(db)
		conflictCount, _ := store.ConflictCount(db)
		render.JSON(map[string]interface{}{
			"statuses": statusMap,
			"queue": map[string]int{
				"pending":   pending,
				"failed":    failed,
				"completed": completed,
			},
			"conflicts": conflictCount,
		})
		return
	}

	fmt.Println(render.Header("Sync Status"))
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		fmt.Printf("  %-12s %d\n", status+":", count)
	}

	pending, failed, completed, err := store.QueueCount(db)
	if err == nil {
		fmt.Println()
		fmt.Println(render.Header("Sync Queue"))
		fmt.Printf("  pending:   %d\n", pending)
		fmt.Printf("  failed:    %d\n", failed)
		fmt.Printf("  completed: %d\n", completed)
	}

	conflictCount, err := store.ConflictCount(db)
	if err == nil && conflictCount > 0 {
		fmt.Println()
		fmt.Printf("  conflicts: %d\n", conflictCount)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func requireDBAndProvider() (*sql.DB, provider.RemoteProvider, *config.MergedConfig) {
	if !store.IsInitialized() {
		fmt.Fprintln(os.Stderr, "Error: no local project found. Run: tkt init")
		os.Exit(1)
	}
	db, err := store.Open()
	if err != nil {
		handleError(err)
	}

	svc, merged, err := buildService()
	if err != nil {
		db.Close()
		fmt.Fprintln(os.Stderr, "Error: no remote configured. Run: tkt config setup")
		os.Exit(2)
	}
	return db, svc.RemoteProvider(), merged
}

func renderSyncResult(result *sync.Result, operation string) {
	if getOutputMode() == render.OutputJSON {
		render.JSON(result)
		return
	}
	if quietFlag {
		fmt.Printf("%d %d %d %d\n", result.Pulled, result.Pushed, result.Conflicts, result.Failed)
		return
	}

	switch operation {
	case "pull":
		fmt.Printf("Pulled %d issues.\n", result.Pulled)
	case "push":
		fmt.Printf("Pushed %d changes.\n", result.Pushed)
	}
	if result.Failed > 0 {
		fmt.Printf("Failed: %d\n", result.Failed)
		for _, e := range result.Errors {
			fmt.Printf("  error: %s\n", e)
		}
	}
	if result.Conflicts > 0 {
		fmt.Printf("Conflicts: %d (run 'tkt remote status' to see details)\n", result.Conflicts)
	}
}

func init() {
	remoteAuthCmd.Flags().StringVar(&remoteAuthToken, "token", "", "Permanent token (skip interactive prompt)")
	remoteAuthCmd.Flags().BoolVar(&remoteAuthOAuth, "oauth", false, "Use OAuth 2.0 implicit grant")

	remoteCmd.AddCommand(remoteAuthCmd)
	remoteCmd.AddCommand(remoteWhoamiCmd)
	remoteCmd.AddCommand(remotePullCmd)
	remoteCmd.AddCommand(remotePushCmd)
	remoteCmd.AddCommand(remoteStatusCmd)
	rootCmd.AddCommand(remoteCmd)
}