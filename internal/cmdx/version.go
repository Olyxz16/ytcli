package cmdx

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/render"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]string{
				"version": Version,
				"commit":  Commit,
			})
			return
		}
		if quietFlag {
			fmt.Println(Version)
			return
		}
		fmt.Printf("tkt version %s (commit: %s)\n", Version, Commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}