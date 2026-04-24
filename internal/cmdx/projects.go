package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/render"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		projects, err := svc.ListProjects(cmd.Context())
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			for _, p := range projects {
				fmt.Println(p.ShortName)
			}
			return
		}

		if err := render.ProjectList(projects, getOutputMode()); err != nil {
			handleError(err)
		}
	},
}

var projectCmd = &cobra.Command{
	Use:   "project <id>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		project, err := svc.GetProject(cmd.Context(), args[0])
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(project.ShortName)
			return
		}

		fmt.Printf("%s  %s\n", project.ShortName, project.Name)
		if project.Description != "" {
			fmt.Println(project.Description)
		}
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(projectCmd)
}
