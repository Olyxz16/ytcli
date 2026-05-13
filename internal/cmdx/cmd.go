package cmdx

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/render"
)

var (
	cmdCommentFlag string
	cmdSilentFlag  bool
)

var cmdCmd = &cobra.Command{
	Use:     "cmd <issue-id> <command>",
	Short:   "Apply a YouTrack command to an issue",
	Aliases: []string{"c"},
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		issueID := args[0]
		command := strings.Join(args[1:], " ")

		result, err := svc.ExecuteCommand(cmd.Context(), command, []string{issueID}, cmdSilentFlag)
		if err != nil {
			handleError(err)
		}

		if cmdCommentFlag != "" {
			_, _ = svc.AddComment(cmd.Context(), issueID, cmdCommentFlag)
		}

		if quietFlag {
			fmt.Println(result.Query)
			return
		}

		if err := render.CommandResult(result, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	cmdCmd.Flags().StringVarP(&cmdCommentFlag, "comment", "c", "", "Add a comment alongside the command")
	cmdCmd.Flags().BoolVar(&cmdSilentFlag, "silent", false, "Apply without sending notifications")

	rootCmd.AddCommand(cmdCmd)
}
