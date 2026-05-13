package cmdx

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/store"
)

var commentCmd = &cobra.Command{
	Use:   "comment <issue-id> <text>",
	Short: "Add a comment to an issue",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		issueID := args[0]
		text := strings.Join(args[1:], " ")

		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, issueID, localCfg)
				if err == nil {
					comment, err := store.CreateComment(db, localID, text)
					if err != nil {
						handleError(err)
					}

					issue, _ := store.GetIssue(db, localID)
					if issue != nil && (issue.SyncStatus == "synced" || issue.SyncStatus == "modified") {
						_ = store.Enqueue(db, "comment", "comment", comment.ID, map[string]interface{}{"text": text})
					}

					if quietFlag {
						fmt.Println(comment.ID)
						return
					}
					render.LocalCommentList([]store.LocalComment{*comment}, getOutputMode())
					return
				}
			}
		}

		// Fall back to remote API
		svc, _, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", issueID)
			os.Exit(3)
		}

		comment, err := svc.AddComment(cmd.Context(), issueID, text)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(comment.ID)
			return
		}

		if err := render.CommentList([]model.Comment{*comment}, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var commentsCmd = &cobra.Command{
	Use:   "comments <issue-id>",
	Short: "List comments on an issue",
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
					comments, err := store.ListComments(db, localID)
					if err != nil {
						handleError(err)
					}

					if quietFlag {
						for _, c := range comments {
							fmt.Println(c.ID)
						}
						return
					}

					if err := render.LocalCommentList(comments, getOutputMode()); err != nil {
						handleError(err)
					}
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

		comments, err := svc.ListComments(cmd.Context(), args[0])
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			for _, c := range comments {
				fmt.Println(c.ID)
			}
			return
		}

		if err := render.CommentList(comments, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(commentsCmd)
}
