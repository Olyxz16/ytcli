package cmdx

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/local"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
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
		openWeb, _ := cmd.Flags().GetBool("web")
		withComments, _ := cmd.Flags().GetBool("comments")

		// Try local first if DB exists
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					issue, err := store.GetIssue(db, localID)
					if err == nil && issue != nil {
						tags, _ := store.GetIssueTags(db, issue.ID)
						issue.Tags = tags
						issue.Comments, _ = store.CountComments(db, issue.ID)

						if withComments {
							comments, _ := store.ListComments(db, issue.ID)
							// We can't easily embed comments in LocalIssue for rendering
							// So render detail first, then comments separately
							if quietFlag {
								fmt.Println(local.FormatID(issue))
								return
							}
							render.LocalIssueDetail(issue, getOutputMode())
							if len(comments) > 0 {
								fmt.Println()
								render.Header("Comments")
								render.LocalCommentList(comments, getOutputMode())
							}
							return
						}

						if quietFlag {
							fmt.Println(local.FormatID(issue))
							return
						}
						render.LocalIssueDetail(issue, getOutputMode())
						return
					}
				}
			}
		}

		// Fall back to remote API
		svc, merged, err := buildService()
		if err != nil {
			// No remote configured either
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		if openWeb {
			url := fmt.Sprintf("%s/issue/%s", merged.InstanceURL, args[0])
			if err := openBrowser(url); err != nil {
				handleError(err)
			}
			return
		}

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
