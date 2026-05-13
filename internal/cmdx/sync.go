package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/store"
	"github.com/Olyxz16/tkt/internal/sync"
)

var syncDryRunFlag bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Bidirectional sync: pull then push",
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

		svc, merged, err := buildService()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: no remote configured. Run: tkt remote auth")
			os.Exit(2)
		}

		if syncDryRunFlag {
			fmt.Println("Dry run mode — no changes will be made.")
			fmt.Println()
		}

		prov := svc.RemoteProvider()
		mgr := sync.NewManager(db, prov, merged)
		ctx := cmd.Context()

		fmt.Println("Pulling remote issues...")
		pullResult, err := mgr.Pull(ctx)
		if err != nil {
			handleError(err)
		}
		if getOutputMode() != render.OutputJSON && !quietFlag {
			fmt.Printf("Pulled %d issues.\n", pullResult.Pulled)
			if pullResult.Failed > 0 {
				for _, e := range pullResult.Errors {
					fmt.Printf("  pull error: %s\n", e)
				}
			}
			fmt.Println()
		}

		fmt.Println("Pushing local changes...")
		pushResult, err := mgr.Push(ctx)
		if err != nil {
			handleError(err)
		}

		result := &sync.Result{
			Pulled:    pullResult.Pulled,
			Pushed:    pushResult.Pushed,
			Conflicts: pullResult.Conflicts + pushResult.Conflicts,
			Failed:    pullResult.Failed + pushResult.Failed,
			Errors:    append(pullResult.Errors, pushResult.Errors...),
		}

		if getOutputMode() == render.OutputJSON {
			render.JSON(result)
			return
		}
		if quietFlag {
			fmt.Printf("%d %d %d %d\n", result.Pulled, result.Pushed, result.Conflicts, result.Failed)
			return
		}

		fmt.Printf("Pushed %d changes.\n", result.Pushed)
		if result.Failed > 0 {
			fmt.Printf("Failed: %d\n", result.Failed)
			for _, e := range result.Errors {
				fmt.Printf("  error: %s\n", e)
			}
		}
		if result.Conflicts > 0 {
			fmt.Printf("Conflicts: %d (run 'tkt remote status' to see details)\n", result.Conflicts)
		}
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRunFlag, "dry-run", false, "Preview changes without applying")
	rootCmd.AddCommand(syncCmd)
}