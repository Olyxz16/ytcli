package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
)

var (
	syncStatusFlag bool
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
			// Show sync status breakdown
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
			return
		}

		// TODO: implement actual sync in Phase C
		fmt.Println("Sync not yet implemented. Use --status to view current state.")
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncStatusFlag, "status", false, "Show sync status")
	syncCmd.Flags().BoolVar(&syncDryRunFlag, "dry-run", false, "Preview changes without applying")

	rootCmd.AddCommand(syncCmd)
}

