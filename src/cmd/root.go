// gitclt
// src/cmd/root.go

package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitclt",
	Short: "Add a short description here",
	Long: `This tools allows you to create a software directory structure.
This follows my template and allows you to package your software with minimal effort once built`,
}

// buildVersion and buildDate are injected at link time via
// `-ldflags "-X gitclt/cmd.buildVersion=... -X gitclt/cmd.buildDate=..."`
// by every build path (see version-bump.sh and each __*/{Makefile,APKBUILD,
// PKGBUILD,spec}); a plain `go build` leaves them at these fallbacks.
var buildVersion = "dev"
var buildDate = "unknown"

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

	rootCmd.AddCommand(versionCmd)
}
