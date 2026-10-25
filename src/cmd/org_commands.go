// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/org_commands.go
// Original timestamp : 2026.10.24 19:41:51

package cmd

import (
	"fmt"
	"gitclt/organizations"

	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organization"},
	Short:   "Organization management commands",
	Long: `Some of the commands are self-service, but most of the commands here need administrative rights.
In other words: you must have the appropriate access or auth token to perform some of the tasks.`,
}

var orgListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "gitclt org list [-c CONNECTION] [--json] <USER>",
	Short:   "Lists all organizations on the forge that <USER> can access (needs admin rights)",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := ""
		if len(args) > 0 {
			username = args[0]
		}
		if _, e := organizations.ListOrgs(username, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var orgListAllCmd = &cobra.Command{
	Use:     "listall",
	Example: "gitclt org listall [-c CONNECTION] [--json]",
	Short:   "Lists all organizations on the forge (needs admin rights)",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := organizations.ListAllOrgs(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	orgCmd.AddCommand(orgListCmd, orgListAllCmd)

	orgListCmd.Flags().BoolVarP(&organizations.JSONoutput, "json", "j", false, "output the list as JSON")
}
