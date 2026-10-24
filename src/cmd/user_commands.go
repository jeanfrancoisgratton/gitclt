// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/user_commands.go
// Original timestamp : 2026.10.22 01:33:41

package cmd

import (
	"fmt"
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
	Aliases: []string{"who"},
	Example: "gitclt user whoami [--json]",
	Short:   "Gives information about the current user (if no username is provided)",
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := user.WhoAmI(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userGetCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"info"},
	Example: "gitclt user get [-c CONNECTION] [--extended] [--json] <USERNAME>",
	Short:   "Gives information about a user (needs admin rights, Forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := user.UserGet(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "gitclt user list [-c CONNECTION] [--json]",
	Short:   "Lists all users on the forge (needs admin rights, Forgejo only)",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := user.ListUsers(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Example: "gitclt user delete [-c CONNECTION] <USER> [--purge] [--yes]",
	Short:   "Delete a user from the server (forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := user.DeleteUser(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	userCmd.AddCommand(userWhoAmICmd, userGetCmd, userListCmd, userDeleteCmd)

	userWhoAmICmd.Flags().BoolVarP(&user.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userWhoAmICmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "Output the list as JSON")
	userGetCmd.Flags().BoolVarP(&user.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userGetCmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "Output the list as JSON")
	userListCmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "Output the list as JSON")
	userDeleteCmd.Flags().BoolVarP(&user.PurgeUser, "purge", "p", false, "Purge all of the user's contents (repos, etc)")
	userDeleteCmd.Flags().BoolVarP(&user.AssumeYes, "yes", "y", false, "Do not ask for confirmation before deleting")

}
