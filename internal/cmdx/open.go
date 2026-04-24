package cmdx

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:     "open <issue-id>",
	Short:   "Open an issue in the browser",
	Aliases: []string{"o"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, merged, err := buildService()
		if err != nil {
			handleError(err)
		}

		issueID := args[0]
		url := fmt.Sprintf("%s/issue/%s", merged.InstanceURL, issueID)
		if err := openBrowser(url); err != nil {
			handleError(err)
		}
	},
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
		// Fallback if xdg-open is not available
		if _, err := exec.LookPath(cmd); err != nil {
			for _, b := range []string{"google-chrome", "chromium", "firefox", "opera", "brave"} {
				if _, err := exec.LookPath(b); err == nil {
					cmd = b
					args = []string{url}
					break
				}
			}
		}
	}

	return exec.Command(cmd, args...).Start()
}

func init() {
	rootCmd.AddCommand(openCmd)
}
