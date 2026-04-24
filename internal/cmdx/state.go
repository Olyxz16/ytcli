package cmdx

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/local"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
)

var stateCmd = &cobra.Command{
	Use:   "state <issue-id> <state>",
	Short: "Change the state of an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		changeState(args[0], args[1])
	},
}

var doneCmd = &cobra.Command{
	Use:   "done <issue-id>",
	Short: "Mark an issue as done",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		localCfg, _, _ := config.LoadLocal()
		schema := local.GetSchema(localCfg)
		doneState := "Done"
		if len(schema.DoneStates) > 0 {
			doneState = schema.DoneStates[0]
		}
		changeState(args[0], doneState)
	},
}

func changeState(inputID, newState string) {
	// Try local first
	if store.IsInitialized() {
		db, err := store.Open()
		if err == nil {
			defer db.Close()
			localCfg, _, _ := config.LoadLocal()
			schema := local.GetSchema(localCfg)

			if err := local.ValidateState(schema, newState); err != nil {
				handleError(err)
			}

			localID, err := local.ResolveID(db, inputID, localCfg)
			if err == nil {
				updated, err := store.UpdateIssue(db, localID, map[string]interface{}{"state": newState})
				if err != nil {
					handleError(err)
				}

				if updated.SyncStatus == "synced" || updated.SyncStatus == "modified" {
					_ = store.Enqueue(db, "state", "issue", updated.ID, map[string]interface{}{"state": newState})
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

	// Fall back to remote command
	svc, _, err := buildService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", inputID)
		os.Exit(3)
	}

	ctx := context.Background()
	result, err := svc.ExecuteCommand(ctx, fmt.Sprintf("State: %s", newState), []string{inputID}, false)
	if err != nil {
		handleError(err)
	}

	if quietFlag {
		fmt.Println(result.Query)
		return
	}
	render.CommandResult(result, getOutputMode())
}

func init() {
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(doneCmd)
}
