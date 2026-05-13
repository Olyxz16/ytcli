package cmdx

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/store"
)

var openCmd = &cobra.Command{
	Use:     "open <issue-id>",
	Short:   "Open an issue in browser or editor",
	Aliases: []string{"o"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Try local first
		if store.IsInitialized() {
			db, err := store.Open()
			if err == nil {
				defer db.Close()
				localCfg, _, _ := config.LoadLocal()
				localID, err := local.ResolveID(db, args[0], localCfg)
				if err == nil {
					issue, _ := store.GetIssue(db, localID)
					if issue != nil {
if issue.ProviderRef != "" {
							// Open in browser
							_, merged, err := buildService()
							if err == nil {
								url := fmt.Sprintf("%s/issue/%s", merged.ProviderURL, issue.ProviderRef)
								if err := openBrowser(url); err != nil {
									handleError(err)
								}
								return
							}
						}
						// Open in editor
						editor := os.Getenv("EDITOR")
						if editor == "" {
							editor = "vim"
						}
						tmpfile, err := os.CreateTemp("", fmt.Sprintf("tkt-issue-%d-*.md", issue.ID))
						if err != nil {
							handleError(err)
						}
						defer os.Remove(tmpfile.Name())

						content := fmt.Sprintf("# %s\n\nState: %s\nPriority: %s\n\n%s",
							issue.Summary, issue.State, issue.Priority, issue.Description)
						fmt.Fprint(tmpfile, content)
						tmpfile.Close()

						c := exec.Command(editor, tmpfile.Name())
						c.Stdin = os.Stdin
						c.Stdout = os.Stdout
						c.Stderr = os.Stderr
						if err := c.Run(); err != nil {
							handleError(err)
						}
						return
					}
				}
			}
		}

		// Fall back to remote browser
		_, merged, err := buildService()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: issue %q not found locally and no remote configured\n", args[0])
			os.Exit(3)
		}

		url := fmt.Sprintf("%s/issue/%s", merged.ProviderURL, args[0])
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
