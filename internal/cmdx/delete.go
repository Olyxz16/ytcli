package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/store"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <issue-id>",
	Short: "Delete an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					issue, _ := store.GetIssue(db, localID)
					if issue != nil && issue.ProviderRef != "" {
						_ = store.Enqueue(db, "delete", "issue", issue.ID, map[string]interface{}{})
					}
					if err := store.DeleteIssue(db, localID); err != nil {
						handleError(err)
					}
					if quietFlag {
						fmt.Println(args[0])
						return
					}
					if getOutputMode() == render.OutputJSON {
						render.JSON(map[string]interface{}{"id": args[0], "deleted": true, "local": true})
						return
					}
					fmt.Println("Deleted locally")
					return
				}
			}
		}

		// Fall back to remote
		svc, _, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		if err := svc.DeleteIssue(cmd.Context(), args[0]); err != nil {
			handleError(err)
		}
		if quietFlag {
			fmt.Println(args[0])
			return
		}
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]interface{}{"id": args[0], "deleted": true})
			return
		}
		fmt.Println("Deleted")
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
