// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/15 08:35
// Original filename: src/cmd/completion_commands.go
// Bash/Zsh completion scripts via Cobra.

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:     "completion",
	Example: "gitclt completion bash\n  gitclt completion zsh",
	Short:   "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

Bash:
  $ source <(gitclt completion bash)
  # To persist:
  $ gitclt completion bash | sudo tee /etc/bash_completion.d/gitclt > /dev/null

Zsh:
  $ gitclt completion zsh > ~/.zsh[.completion.d]/_gitclt
  $ echo 'fpath=($HOME/.zsh $fpath)' >> ~/.zshrc
  $ echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
  # Or, for current session:
  $ source <(gitclt completion zsh)
`,
}

var completionBashCmd = &cobra.Command{
	Use:     "bash",
	Example: "source <(gitclt completion bash)",
	Short:   "Generate a Bash completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		// V2 is recommended; writes to stdout
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:     "zsh",
	Example: "source <(gitclt completion zsh)",
	Short:   "Generate a Zsh completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure the script is zsh-compatible
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd, completionZshCmd)
}
