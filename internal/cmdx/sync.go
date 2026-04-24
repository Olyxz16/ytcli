package cmdx

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
	"github.com/Olyxz16/ytcli/internal/sync"
)

var (
	syncStatusFlag bool
	syncPushFlag   bool
	syncPullFlag   bool
	syncDryRunFlag bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local issues with YouTrack",
	Run: func(cmd *cobra.Command, args []string) {
		if !store.IsInitialized() {
			fmt.Fprintln(os.Stderr, "Error: no local project found. Run: ytcli init")
			os.Exit(1)
		}

		db, err := store.Open()
		if err != nil {
			handleError(err)
		}
		defer db.Close()

		if syncStatusFlag {
			showSyncStatus(db)
			return
		}

		svc, merged, err := buildService()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: no remote configured. Run: ytcli config setup")
			os.Exit(2)
		}

		if syncDryRunFlag {
			fmt.Println("Dry run mode — no changes will be made.")
			fmt.Println()
		}

		mgr := sync.NewManager(db, svc, merged)
		ctx := cmd.Context()
		var result *sync.Result

		if syncPullFlag {
			fmt.Println("Pulling remote issues...")
			result, err = mgr.Pull(ctx)
		} else if syncPushFlag {
			fmt.Println("Pushing local changes...")
			result, err = mgr.Push(ctx)
		} else {
			// Bidirectional: pull then push
			fmt.Println("Pulling remote issues...")
			pullResult, err := mgr.Pull(ctx)
			if err != nil {
				handleError(err)
			}
			fmt.Printf("Pulled %d issues.\n", pullResult.Pulled)
			if pullResult.Failed > 0 {
				for _, e := range pullResult.Errors {
					fmt.Printf("  pull error: %s\n", e)
				}
			}
			fmt.Println()

			fmt.Println("Pushing local changes...")
			result, err = mgr.Push(ctx)
		}

		if err != nil {
			handleError(err)
		}

		fmt.Printf("Pushed %d changes.\n", result.Pushed)
		if result.Failed > 0 {
			fmt.Printf("Failed: %d\n", result.Failed)
			for _, e := range result.Errors {
				fmt.Printf("  push error: %s\n", e)
			}
		}
		if result.Conflicts > 0 {
			fmt.Printf("Conflicts: %d (run 'ytcli sync --status' to see details)\n", result.Conflicts)
		}
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
}

func init() {
	syncCmd.Flags().BoolVar(&syncStatusFlag, "status", false, "Show sync status")
	syncCmd.Flags().BoolVar(&syncPushFlag, "push", false, "Push local changes only")
	syncCmd.Flags().BoolVar(&syncPullFlag, "pull", false, "Pull remote changes only")
	syncCmd.Flags().BoolVar(&syncDryRunFlag, "dry-run", false, "Preview changes without applying")

	rootCmd.AddCommand(syncCmd)
}
