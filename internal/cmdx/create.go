package cmdx

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/render"
)

var (
	createSummaryFlag     string
	createDescriptionFlag string
	createTypeFlag        string
	createPriorityFlag    string
	createAssigneeFlag    string
	createTagsFlag        []string
)

var createCmd = &cobra.Command{
	Use:     "create <project>",
	Short:   "Create a new issue",
	Aliases: []string{"n"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, merged, err := buildService()
		if err != nil {
			handleError(err)
		}

		if createSummaryFlag == "" {
			fmt.Fprint(os.Stderr, "Error: summary is required (-s flag)\n")
			os.Exit(1)
		}

		projectID := args[0]
		if merged.Project != "" && projectID == "" {
			projectID = merged.Project
		}

		desc := createDescriptionFlag
		if desc == "" {
			// Check stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				scanner := bufio.NewScanner(os.Stdin)
				var lines []string
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				desc = strings.Join(lines, "\n")
			}
		}

		issue := model.Issue{
			Summary:     createSummaryFlag,
			Description: desc,
			Project:     &model.Project{ShortName: projectID},
		}

		if createPriorityFlag != "" {
			issue.Priority = createPriorityFlag
		}
		if createAssigneeFlag != "" {
			assignee := createAssigneeFlag
			if assignee == "me" {
				me, err := svc.Me(cmd.Context())
				if err == nil {
					assignee = me.Login
				}
			}
			issue.Assignee = &model.User{Login: assignee}
		}

		if len(createTagsFlag) > 0 {
			var tags []model.Tag
			for _, t := range createTagsFlag {
				tags = append(tags, model.Tag{Name: t})
			}
			issue.Tags = tags
		}

		created, err := svc.CreateIssue(cmd.Context(), issue)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(created.ID)
			return
		}

		if err := render.IssueDetail(created, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	createCmd.Flags().StringVarP(&createSummaryFlag, "summary", "s", "", "Issue summary (required)")
	createCmd.Flags().StringVarP(&createDescriptionFlag, "description", "d", "", "Issue description")
	createCmd.Flags().StringVarP(&createTypeFlag, "type", "t", "", "Issue type")
	createCmd.Flags().StringVarP(&createPriorityFlag, "priority", "P", "", "Priority")
	createCmd.Flags().StringVarP(&createAssigneeFlag, "assignee", "a", "", "Assignee (use 'me' for current user)")
	createCmd.Flags().StringArrayVar(&createTagsFlag, "tag", nil, "Tags (repeatable)")

	rootCmd.AddCommand(createCmd)
}