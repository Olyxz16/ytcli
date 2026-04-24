package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/render"
)

var commentCmd = &cobra.Command{
	Use:   "comment <issue-id> <text>",
	Short: "Add a comment to an issue",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		issueID := args[0]
		text := args[1]
		if len(args) > 2 {
			for i := 2; i < len(args); i++ {
				text += " " + args[i]
			}
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
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
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
