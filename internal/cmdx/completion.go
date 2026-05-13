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
  $ source <(tkt completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ tkt completion bash > /etc/bash_completion.d/tkt
  # macOS:
  $ tkt completion bash > $(brew --prefix)/etc/bash_completion.d/tkt

Zsh:
  $ source <(tkt completion zsh)
  # To load completions for each session, execute once:
  $ tkt completion zsh > "${fpath[1]}/_tkt"

Fish:
  $ tkt completion fish | source
  # To load completions for each session, execute once:
  $ tkt completion fish > ~/.config/fish/completions/tkt.fish

PowerShell:
  PS> tkt completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> tkt completion powershell > tkt.ps1
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
