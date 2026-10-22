// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/user_commands.go
// Original timestamp : 2026.10.22 01:33:41

package cmd

import (
	"gitclt/user"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long: `Some of the commands are self-service, but most of the commands here need administrative rights.
In other words: you must have the appropriate access or auth token to perform some of the tasks.`,
}

var userWhoAmICmd = &cobra.Command{
	Use:     "whoami",
	Example: "gitclt user [-x] [<USERNAME>]",
	Short:   "Gives information about the current user (if no username is provided)",
	Long: `The -x flag would give extended information.
If a username is provided administrative rights are required.`,
	Run: func(cmd *cobra.Command, args []string) {
		//if _, e := repositories.ListRepositories(true); e != nil {
		//	fmt.Println(e.Error())
		//}
	},
}

func init() {
	userCmd.AddCommand(userWhoAmICmd)

	userWhoAmICmd.Flags().BoolVarP(&user.ExtendedInfo, "extended", "x", false, "provide extended user information")
}
