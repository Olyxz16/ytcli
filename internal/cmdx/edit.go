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

var (
	editSummaryFlag     string
	editDescriptionFlag string
	editStateFlag       string
	editPriorityFlag    string
	editAssigneeFlag    string
)

var editCmd = &cobra.Command{
	Use:   "edit <issue-id>",
	Short: "Edit an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Try local first if DB exists
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					updates := make(map[string]interface{})
					if editSummaryFlag != "" {
						updates["summary"] = editSummaryFlag
					}
					if editDescriptionFlag != "" {
						updates["description"] = editDescriptionFlag
					}
					if editStateFlag != "" {
						updates["state"] = editStateFlag
					}
					if editPriorityFlag != "" {
						updates["priority"] = editPriorityFlag
					}
					if editAssigneeFlag != "" {
						updates["assignee"] = editAssigneeFlag
					}

					if len(updates) == 0 {
						fmt.Fprintln(os.Stderr, "Error: nothing to edit. Use flags.")
						os.Exit(1)
					}

					updated, err := store.UpdateIssue(db, localID, updates)
					if err != nil {
						handleError(err)
					}

					// Queue sync if this issue has a remote mapping
					if updated.SyncStatus == "synced" || updated.SyncStatus == "modified" {
						_ = store.Enqueue(db, "update", "issue", updated.ID, updates)
					}

					if quietFlag {
						fmt.Println(local.FormatID(updated))
						return
					}
					render.LocalIssueDetail(updated, getOutputMode())
					return
				}
			}
		}

		// Fall back to remote API
		svc, _, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		updates := make(map[string]interface{})
		if editSummaryFlag != "" {
			updates["summary"] = editSummaryFlag
		}
		if editDescriptionFlag != "" {
			updates["description"] = editDescriptionFlag
		}

		if len(updates) == 0 {
			fmt.Fprintln(os.Stderr, "Error: nothing to edit. Use -s or -d flags.")
			os.Exit(1)
		}

		updated, err := svc.UpdateIssue(cmd.Context(), args[0], updates)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(updated.ID)
			return
		}

		if err := render.IssueDetail(updated, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	editCmd.Flags().StringVarP(&editSummaryFlag, "summary", "s", "", "New summary")
	editCmd.Flags().StringVarP(&editDescriptionFlag, "description", "d", "", "New description")
	editCmd.Flags().StringVar(&editStateFlag, "state", "", "New state")
	editCmd.Flags().StringVarP(&editPriorityFlag, "priority", "P", "", "New priority")
	editCmd.Flags().StringVarP(&editAssigneeFlag, "assignee", "a", "", "New assignee")

	rootCmd.AddCommand(editCmd)
}
