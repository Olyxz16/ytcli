package cmdx

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/store"
)

var forceStateFlag bool

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

			if !forceStateFlag {
				if err := local.ValidateState(schema, newState); err != nil {
					var valErr *provider.ValidationError
					if !errors.As(err, &valErr) {
						err = &provider.ValidationError{Provider: "local", Message: err.Error()}
					}
					handleError(err)
				}
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

	if !forceStateFlag {
		ctx := context.Background()
		provSchema, err := svc.RemoteProvider().FetchSchema(ctx)
		if err == nil && provSchema != nil && len(provSchema.States) > 0 {
			if err := local.ValidateState(config.LocalSchema{
				States:       provSchema.States,
				Priorities:   provSchema.Priorities,
				DoneStates:   provSchema.DoneStates,
				DefaultState: provSchema.DefaultState,
			}, newState); err != nil {
				handleError(&provider.ValidationError{Provider: svc.RemoteProvider().Name(), Message: err.Error()})
			}
		}
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
	stateCmd.Flags().BoolVar(&forceStateFlag, "force", false, "Bypass state validation")
	doneCmd.Flags().BoolVar(&forceStateFlag, "force", false, "Bypass state validation")
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(doneCmd)
}
