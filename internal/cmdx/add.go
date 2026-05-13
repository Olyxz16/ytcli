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
	addSummaryFlag     string
	addDescriptionFlag string
	addStateFlag       string
	addPriorityFlag    string
	addAssigneeFlag    string
	addTagsFlag        []string
)

var addCmd = &cobra.Command{
	Use:   "add [summary]",
	Short: "Create a new local issue",
	Args:  cobra.MaximumNArgs(1),
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

		localCfg, _, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		schema := local.GetSchema(localCfg)

		summary := addSummaryFlag
		if len(args) > 0 {
			summary = args[0]
		}
		if summary == "" {
			fmt.Fprintln(os.Stderr, "Error: summary is required")
			os.Exit(1)
		}

		state := addStateFlag
		if state == "" {
			state = schema.DefaultState
		}
		if err := local.ValidateState(schema, state); err != nil {
			handleError(err)
		}

		priority := addPriorityFlag
		if priority == "" {
			priority = schema.DefaultPriority
		}
		if err := local.ValidatePriority(schema, priority); err != nil {
			handleError(err)
		}

		issue, err := store.CreateIssue(db, summary, addDescriptionFlag, state, priority, addAssigneeFlag)
		if err != nil {
			handleError(err)
		}

		for _, tag := range addTagsFlag {
			_ = store.AddIssueTag(db, issue.ID, tag)
		}

		// Reload with tags
		issue, _ = store.GetIssue(db, issue.ID)
		tags, _ := store.GetIssueTags(db, issue.ID)
		issue.Tags = tags

		if quietFlag {
			fmt.Printf("#%d\n", issue.ID)
			return
		}

		if err := render.LocalIssueDetail(issue, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	addCmd.Flags().StringVarP(&addSummaryFlag, "summary", "s", "", "Issue summary")
	addCmd.Flags().StringVarP(&addDescriptionFlag, "description", "d", "", "Issue description")
	addCmd.Flags().StringVar(&addStateFlag, "state", "", "Initial state")
	addCmd.Flags().StringVarP(&addPriorityFlag, "priority", "P", "", "Priority")
	addCmd.Flags().StringVarP(&addAssigneeFlag, "assignee", "a", "", "Assignee")
	addCmd.Flags().StringArrayVar(&addTagsFlag, "tag", nil, "Tags (repeatable)")

	rootCmd.AddCommand(addCmd)
}
