package cmdx

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
)

var (
	listLimitFlag int
	listStateFlag string
)

var listCmd = &cobra.Command{
	Use:     "list [query]",
	Short:   "List local issues",
	Aliases: []string{"ls"},
	Args:    cobra.ArbitraryArgs,
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

		query := strings.Join(args, " ")
		issues, err := store.ListIssues(db, query, listLimitFlag)
		if err != nil {
			handleError(err)
		}

		// Filter by state if requested
		if listStateFlag != "" {
			var filtered []store.LocalIssue
			for _, i := range issues {
				if i.State == listStateFlag {
					filtered = append(filtered, i)
				}
			}
			issues = filtered
		}

		// Load tags and comment counts
		_ = store.LoadIssueTags(db, issues)
		_ = store.LoadIssueComments(db, issues)

		if quietFlag {
			for _, i := range issues {
				fmt.Printf("#%d\n", i.ID)
			}
			return
		}

		if err := render.LocalIssueList(issues, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	listCmd.Flags().IntVar(&listLimitFlag, "limit", 100, "Max results")
	listCmd.Flags().StringVar(&listStateFlag, "state", "", "Filter by state")

	rootCmd.AddCommand(listCmd)
}
