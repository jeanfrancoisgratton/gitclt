// gitclt
// src/cmd/root.go

package cmd

import (
	"fmt"
	"gitclt/shared"
	"os"
	"runtime"
	"strings"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitclt",
	Short: "forgejo / github / bitbucket service CLI management tool",
	Long: `This tools allows you to manage the usual git server tasks. such as
repository management, branch management, tags, releases, users, orgs, etc`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the software version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(hftx.White("gitclt v" + buildVersion + " (" + buildDate + "), Go version = v" + strings.TrimPrefix(runtime.Version(), "go") + " (" + runtime.GOARCH + ")"))
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.DisableAutoGenTag = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(versionCmd, repoCmd, conn_cfgCmd)
	rootCmd.PersistentFlags().BoolVarP(&shared.QuietOutput, "quiet", "q", false, "Output will be as quiet as possible")
}
