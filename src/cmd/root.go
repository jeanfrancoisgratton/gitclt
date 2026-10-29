// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/root.go
// Original timestamp : 2026.10.17 23:47:23

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
	Short: "Forgejo / GitHub / Bitbucket service CLI management tool",
	Long: `This tool allows you to manage your usual git server tasks, such as
repository management, branch management, tags, releases, users, orgs, etc`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the software version",
	Args:  cobra.NoArgs,
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
	rootCmd.SilenceUsage = true

	rootCmd.AddCommand(repoCmd, connectionCmd, userCmd, orgCmd, teamCmd, branchCmd, releaseCmd, tagCmd)
	rootCmd.AddCommand(versionCmd, userWhoAmICmd, completionCmd)

	rootCmd.PersistentFlags().StringVarP(&shared.ConnectionName, "connection", "c", "", "connection to use (default: the default connection)")
	rootCmd.PersistentFlags().BoolVarP(&shared.QuietOutput, "quiet", "q", false, "output will be as quiet as possible")
	rootCmd.PersistentFlags().BoolVarP(&shared.DebugMode, "debug", "D", false, "debug mode")
}
