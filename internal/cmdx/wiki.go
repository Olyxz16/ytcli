package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/wiki"
)

var (
	wikiPullFlag   bool
	wikiPushFlag   bool
	wikiStatusFlag bool
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Sync wiki articles with local directory",
	Long: `Sync YouTrack knowledge base articles with a local wiki directory.

The wiki directory must be configured with:
  ytcli config set wiki_dir <path>

Use --pull to download articles from YouTrack to local files.
Use --push to upload local edits to existing YouTrack articles.
Use --status to check if local files are up to date with remote.

Pull is idempotent: files are only overwritten if the remote version is newer.
Push only updates existing articles — it never creates new ones.
To create a new article, use: ytcli create-article`,
	Run: func(cmd *cobra.Command, args []string) {
		localCfg, _, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		if localCfg.WikiDir == "" {
			fmt.Fprintln(os.Stderr, "Error: wiki_dir not configured. Run: ytcli config set wiki_dir <path>")
			os.Exit(1)
		}

		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		ctx := cmd.Context()
		mode := getOutputMode()

		if wikiStatusFlag {
			result, err := wiki.Status(ctx, svc, localCfg.WikiDir)
			if err != nil {
				handleError(err)
			}
			if mode == render.OutputJSON || quietFlag {
				render.JSON(result)
				return
			}
			if result.NeedsPull {
				fmt.Println("Remote articles are newer — pull recommended")
			} else {
				fmt.Println("Local wiki is up to date")
			}
			fmt.Printf("Wiki dir: %s\n", result.WikiDir)
			return
		}

		if wikiPullFlag {
			result, err := wiki.Pull(ctx, svc, localCfg.WikiDir)
			if err != nil {
				handleError(err)
			}
			if mode == render.OutputJSON {
				render.JSON(result)
				return
			}
			fmt.Printf("Pulled %d articles, skipped %d (up to date)\n", result.Pulled, result.Skipped)
			if len(result.Errors) > 0 {
				for _, e := range result.Errors {
					fmt.Printf("  error: %s\n", e)
				}
			}
			return
		}

		if wikiPushFlag {
			result, err := wiki.Push(ctx, svc, localCfg.WikiDir)
			if err != nil {
				handleError(err)
			}
			if mode == render.OutputJSON {
				render.JSON(result)
				return
			}
			fmt.Printf("Pushed %d articles, skipped %d (no remote ID)\n", result.Pushed, result.Skipped)
			if len(result.Errors) > 0 {
				for _, e := range result.Errors {
					fmt.Printf("  error: %s\n", e)
				}
			}
			return
		}

		fmt.Fprintln(os.Stderr, "Error: specify --pull, --push, or --status")
		os.Exit(1)
	},
}

func init() {
	wikiCmd.Flags().BoolVar(&wikiPullFlag, "pull", false, "Pull articles from YouTrack to local directory")
	wikiCmd.Flags().BoolVar(&wikiPushFlag, "push", false, "Push local edits to existing YouTrack articles")
	wikiCmd.Flags().BoolVar(&wikiStatusFlag, "status", false, "Check if local wiki is up to date")

	rootCmd.AddCommand(wikiCmd)
}