package cmdx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/store"
)

var (
	articlesLimitFlag int
	articlesQueryFlag string
)

var articlesCmd = &cobra.Command{
	Use:   "articles [query]",
	Short: "List knowledge base articles",
	Aliases: []string{"kb"},
	Run: func(cmd *cobra.Command, args []string) {
		query := articlesQueryFlag
		if len(args) > 0 {
			query = args[0]
		}

		if store.IsInitialized() {
			localCfg, _, _ := config.LoadLocal()
			if query == "" && localCfg != nil && localCfg.DefaultQuery != "" {
				query = localCfg.DefaultQuery
			}
		}

		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		top := articlesLimitFlag
		if top == 0 {
			top = 50
		}

		articles, err := svc.ListArticles(cmd.Context(), query, top, 0)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			for _, a := range articles {
				fmt.Println(a.IDReadable)
			}
			return
		}

		if err := render.ArticleList(articles, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var articleCmd = &cobra.Command{
	Use:   "show-article <article-id>",
	Short: "Show article details",
	Aliases: []string{"article"},
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		article, err := svc.GetArticle(cmd.Context(), args[0], articleCommentsFlag)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(article.IDReadable)
			return
		}

		if err := render.ArticleDetail(article, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var (
	articleSummaryFlag   string
	articleContentFlag   string
	articleProjectFlag   string
	articleCommentsFlag bool
)

var articleCreateCmd = &cobra.Command{
	Use:   "create-article -s \"Title\" [-d \"Content\"] [--project PROJ]",
	Short: "Create a knowledge base article",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if articleSummaryFlag == "" {
			fmt.Fprintln(os.Stderr, "Error: summary is required (-s flag)")
			os.Exit(1)
		}

		svc, merged, err := buildService()
		if err != nil {
			handleError(err)
		}

		project := articleProjectFlag
		if project == "" {
			project = merged.Project
		}
		if project == "" {
			fmt.Fprintln(os.Stderr, "Error: project is required (--project flag or project in config)")
			os.Exit(1)
		}

		article := model.Article{
			Summary: articleSummaryFlag,
			Content: articleContentFlag,
			Project: &model.Project{ShortName: project},
		}

		created, err := svc.CreateArticle(cmd.Context(), article)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(created.IDReadable)
			return
		}

		if err := render.ArticleDetail(created, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var articleDeleteCmd = &cobra.Command{
	Use:   "delete-article <article-id>",
	Short: "Delete a knowledge base article",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		if err := svc.DeleteArticle(cmd.Context(), args[0]); err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(args[0])
			return
		}
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]interface{}{"id": args[0], "deleted": true})
			return
		}
		fmt.Println("Article deleted")
	},
}

func init() {
	articlesCmd.Flags().IntVar(&articlesLimitFlag, "limit", 50, "Max results (0 = all)")
	articlesCmd.Flags().StringVarP(&articlesQueryFlag, "query", "q", "", "Search query")

	articleCmd.Flags().BoolVar(&articleCommentsFlag, "comments", false, "Show comments")

	articleCreateCmd.Flags().StringVarP(&articleSummaryFlag, "summary", "s", "", "Article title (required)")
	articleCreateCmd.Flags().StringVarP(&articleContentFlag, "content", "d", "", "Article content")
	articleCreateCmd.Flags().StringVar(&articleProjectFlag, "project", "", "Project short name")

	rootCmd.AddCommand(articlesCmd)
	rootCmd.AddCommand(articleCmd)
	rootCmd.AddCommand(articleCreateCmd)
	rootCmd.AddCommand(articleDeleteCmd)
}