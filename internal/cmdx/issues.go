package cmdx

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
)

var (
	issuesProjectFlag  string
	issuesAssigneeFlag string
	issuesStateFlag    string
	issuesLimitFlag    int
	issuesSortFlag     string
)

var issuesCmd = &cobra.Command{
	Use:     "issues [query]",
	Short:   "List issues",
	Aliases: []string{"i", "search"},
	Args:    cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		svc, merged, err := buildService()
		if err != nil {
			handleError(err)
		}

		query := strings.Join(args, " ")
		if query == "" && merged.DefaultQuery != "" {
			query = merged.DefaultQuery
		}
		if issuesProjectFlag != "" {
			query = fmt.Sprintf("project: {%s} %s", issuesProjectFlag, query)
		}
		if issuesAssigneeFlag != "" {
			assignee := issuesAssigneeFlag
			if assignee == "me" {
				me, err := svc.Me(cmd.Context())
				if err == nil {
					assignee = me.Login
				}
			}
			query = fmt.Sprintf("for: %s %s", assignee, query)
		}
		if issuesStateFlag != "" {
			query = fmt.Sprintf("State: {%s} %s", issuesStateFlag, query)
		}
		if issuesSortFlag != "" {
			query = fmt.Sprintf("%s sort by: %s", query, issuesSortFlag)
		}

		query = strings.TrimSpace(query)

		issues, err := svc.ListIssues(cmd.Context(), query, issuesLimitFlag, 0)
		if err != nil {
			handleError(err)
		}

		if quietFlag && len(issues) > 0 {
			for _, i := range issues {
				fmt.Println(i.IDReadable)
			}
			return
		}

		if err := render.IssueList(issues, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var showCmd = &cobra.Command{
	Use:     "show <issue-id>",
	Short:   "Show issue details",
	Aliases: []string{"s"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, merged, err := buildService()
		if err != nil {
			handleError(err)
		}

		openWeb, _ := cmd.Flags().GetBool("web")
		if openWeb {
			url := fmt.Sprintf("%s/issue/%s", merged.InstanceURL, args[0])
			if err := openBrowser(url); err != nil {
				handleError(err)
			}
			return
		}

		withComments, _ := cmd.Flags().GetBool("comments")
		issue, err := svc.GetIssue(cmd.Context(), args[0], withComments)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(issue.IDReadable)
			return
		}

		if err := render.IssueDetail(issue, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	issuesCmd.Flags().StringVarP(&issuesProjectFlag, "project", "p", "", "Filter by project")
	issuesCmd.Flags().StringVarP(&issuesAssigneeFlag, "assignee", "a", "", "Filter by assignee (use 'me' for current user)")
	issuesCmd.Flags().StringVarP(&issuesStateFlag, "state", "s", "", "Filter by state")
	issuesCmd.Flags().IntVar(&issuesLimitFlag, "limit", 50, "Max results (0 = all)")
	issuesCmd.Flags().StringVar(&issuesSortFlag, "sort", "", "Sort expression")

	showCmd.Flags().Bool("comments", false, "Include comments")
	showCmd.Flags().BoolP("web", "w", false, "Open in browser")

	rootCmd.AddCommand(issuesCmd)
	rootCmd.AddCommand(showCmd)
}
