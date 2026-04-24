package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <issue-id>",
	Short: "Delete an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		if err := svc.DeleteIssue(cmd.Context(), args[0]); err != nil {
			handleError(err)
		}
		fmt.Println("Deleted")
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
