package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/render"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags",
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		tags, err := svc.ListTags(cmd.Context())
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			for _, t := range tags {
				fmt.Println(t.Name)
			}
			return
		}

		if getOutputMode() == render.OutputJSON {
			render.JSON(tags)
			return
		}

		for _, t := range tags {
			fmt.Println(t.Name)
		}
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "tag-add <issue-id> <tag-name>",
	Short: "Add a tag to an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		if err := svc.AddTagToIssue(cmd.Context(), args[0], model.Tag{Name: args[1]}); err != nil {
			handleError(err)
		}
		fmt.Println("Tag added")
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "tag-remove <issue-id> <tag-id>",
	Short: "Remove a tag from an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		if err := svc.RemoveTagFromIssue(cmd.Context(), args[0], args[1]); err != nil {
			handleError(err)
		}
		fmt.Println("Tag removed")
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(tagAddCmd)
	rootCmd.AddCommand(tagRemoveCmd)
}
