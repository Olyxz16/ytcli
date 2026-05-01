package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/local"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags",
	Run: func(cmd *cobra.Command, args []string) {
		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				tags, err := store.ListTags(db)
				if err != nil {
					handleError(err)
				}

				if quietFlag {
					for _, t := range tags {
						fmt.Println(t)
					}
					return
				}

				if getOutputMode() == render.OutputJSON {
					render.JSON(tags)
					return
				}

				for _, t := range tags {
					fmt.Println(t)
				}
				return
			}
		}

		// Fall back to remote
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
		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					if err := store.AddIssueTag(db, localID, args[1]); err != nil {
						handleError(err)
					}
					issue, _ := store.GetIssue(db, localID)
					if issue != nil && (issue.SyncStatus == "synced" || issue.SyncStatus == "modified") {
						_ = store.Enqueue(db, "tag", "issue", issue.ID, map[string]interface{}{"tag": args[1]})
					}
					if quietFlag {
						fmt.Println(args[0])
						return
					}
					if getOutputMode() == render.OutputJSON {
						render.JSON(map[string]interface{}{"issue": args[0], "tag": args[1], "added": true})
						return
					}
					fmt.Println("Tag added")
					return
				}
			}
		}

		// Fall back to remote
		svc, _, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		if err := svc.AddTagToIssue(cmd.Context(), args[0], model.Tag{Name: args[1]}); err != nil {
			handleError(err)
		}
		if quietFlag {
			fmt.Println(args[0])
			return
		}
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]interface{}{"issue": args[0], "tag": args[1], "added": true})
			return
		}
		fmt.Println("Tag added")
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "tag-remove <issue-id> <tag-name>",
	Short: "Remove a tag from an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					if err := store.RemoveIssueTag(db, localID, args[1]); err != nil {
						handleError(err)
					}
					issue, _ := store.GetIssue(db, localID)
					if issue != nil && (issue.SyncStatus == "synced" || issue.SyncStatus == "modified") {
						_ = store.Enqueue(db, "untag", "issue", issue.ID, map[string]interface{}{"tag": args[1]})
					}
					if quietFlag {
						fmt.Println(args[0])
						return
					}
					if getOutputMode() == render.OutputJSON {
						render.JSON(map[string]interface{}{"issue": args[0], "tag": args[1], "removed": true})
						return
					}
					fmt.Println("Tag removed")
					return
				}
			}
		}

		// Fall back to remote
		svc, _, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		if err := svc.RemoveTagFromIssue(cmd.Context(), args[0], args[1]); err != nil {
			handleError(err)
		}
		if quietFlag {
			fmt.Println(args[0])
			return
		}
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]interface{}{"issue": args[0], "tag": args[1], "removed": true})
			return
		}
		fmt.Println("Tag removed")
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(tagAddCmd)
	rootCmd.AddCommand(tagRemoveCmd)
}
