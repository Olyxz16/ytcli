package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
)

var (
	editSummaryFlag     string
	editDescriptionFlag string
)

var editCmd = &cobra.Command{
	Use:   "edit <issue-id>",
	Short: "Edit an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		updates := make(map[string]interface{})
		if editSummaryFlag != "" {
			updates["summary"] = editSummaryFlag
		}
		if editDescriptionFlag != "" {
			updates["description"] = editDescriptionFlag
		}

		if len(updates) == 0 {
			fmt.Fprint(os.Stderr, "Error: nothing to edit. Use -s or -d flags.\n")
			os.Exit(1)
		}

		updated, err := svc.UpdateIssue(cmd.Context(), args[0], updates)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(updated.IDReadable)
			return
		}

		if err := render.IssueDetail(updated, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

func init() {
	editCmd.Flags().StringVarP(&editSummaryFlag, "summary", "s", "", "New summary")
	editCmd.Flags().StringVarP(&editDescriptionFlag, "description", "d", "", "New description")

	rootCmd.AddCommand(editCmd)
}
