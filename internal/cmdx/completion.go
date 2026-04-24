package cmdx

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(ytcli completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ ytcli completion bash > /etc/bash_completion.d/ytcli
  # macOS:
  $ ytcli completion bash > $(brew --prefix)/etc/bash_completion.d/ytcli

Zsh:
  $ source <(ytcli completion zsh)
  # To load completions for each session, execute once:
  $ ytcli completion zsh > "${fpath[1]}/_ytcli"

Fish:
  $ ytcli completion fish | source
  # To load completions for each session, execute once:
  $ ytcli completion fish > ~/.config/fish/completions/ytcli.fish

PowerShell:
  PS> ytcli completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> ytcli completion powershell > ytcli.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
