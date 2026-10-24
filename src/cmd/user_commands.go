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

var userCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"add"},
	Example: "gitclt user create [-c CONNECTION] -e EMAIL_ADDRESS [other flags] <USER>",
	Short:   "Create a user on the server (forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var mustChange *bool // nil: server default (true on Forgejo)
		if cmd.Flags().Changed("must-change-passwd") {
			v := user.CreateUserMustChangePasswd
			mustChange = &v
		}
		if e := user.CreateUser(args[0], mustChange); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"modify"},
	Example: "gitclt user edit [-c CONNECTION] [flags] <USER>\ngitclt user edit --admin=false --location Montreal jfg2",
	Short:   "Edit a user on the server; only the flags passed are changed (forgejo only)",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --admin=false), and an empty string clears a text field (e.g. --website "").`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := user.EditUser(args[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	userCmd.AddCommand(userWhoAmICmd, userGetCmd, userListCmd, userDeleteCmd, userCreateCmd, userEditCmd)

	userWhoAmICmd.Flags().BoolVarP(&user.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userWhoAmICmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "output the list as JSON")
	userGetCmd.Flags().BoolVarP(&user.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userGetCmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "output the list as JSON")
	userListCmd.Flags().BoolVarP(&user.JSONoutput, "json", "j", false, "output the list as JSON")
	userDeleteCmd.Flags().BoolVarP(&user.PurgeUser, "purge", "p", false, "purge all of the user's contents (repos, etc)")
	userDeleteCmd.Flags().BoolVarP(&user.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	userCreateCmd.Flags().StringVarP(&user.CreateUserEmail, "email", "e", "", "user email")
	userCreateCmd.Flags().StringVarP(&user.CreateUserPasswdFile, "password-file", "p", "", "read the password from file")
	userCreateCmd.Flags().StringVarP(&user.CreateUserFullname, "fullname", "f", "", "user full name")
	userCreateCmd.Flags().BoolVarP(&user.CreateUserMustChangePasswd, "must-change-passwd", "m", false, "created user must change password")
	userCreateCmd.Flags().BoolVarP(&user.CreateUserRestrictedUser, "restricted", "r", false, "restricted user")
	userCreateCmd.Flags().BoolVarP(&user.CreateUserSendNotification, "send-notification", "s", false, "send an email notification to the new user")
	userCreateCmd.Flags().VarP(&user.CreateUserVisibility, "visibility", "v", "user visibility (default: server default)")
	_ = userCreateCmd.RegisterFlagCompletionFunc("visibility", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"public", "limited", "private"}, cobra.ShellCompDirectiveNoFileComp
	})
	userCreateCmd.Flags().StringVarP(&user.CreateUserDescription, "description", "d", "", "user description")
	userCreateCmd.Flags().StringVarP(&user.CreateUserWebsite, "website", "w", "", "user web site")
	userCreateCmd.Flags().StringVarP(&user.CreateUserLocation, "location", "l", "", "user location")
	userCreateCmd.Flags().BoolVarP(&user.CreateUserAdmin, "admin", "a", false, "make the user a site administrator")
	_ = userCreateCmd.MarkFlagRequired("email")

	userEditCmd.Flags().StringVarP(&user.EditUserEmail, "email", "e", "", "user email")
	userEditCmd.Flags().StringVarP(&user.EditUserFullname, "fullname", "f", "", "user full name")
	userEditCmd.Flags().StringVarP(&user.EditUserDescription, "description", "d", "", "user description")
	userEditCmd.Flags().StringVarP(&user.EditUserWebsite, "website", "w", "", "user web site")
	userEditCmd.Flags().StringVarP(&user.EditUserLocation, "location", "l", "", "user location")
	userEditCmd.Flags().StringVarP(&user.EditUserPasswdFile, "password-file", "p", "", "read the new password from file")
	userEditCmd.Flags().BoolVarP(&user.EditUserPromptPasswd, "passwd", "P", false, "prompt for a new password")
	userEditCmd.Flags().BoolVarP(&user.EditUserMustChangePasswd, "must-change-passwd", "m", false, "user must change password at next login")
	userEditCmd.Flags().BoolVarP(&user.EditUserAdmin, "admin", "a", false, "site administrator")
	userEditCmd.Flags().BoolVar(&user.EditUserActive, "active", false, "account is active")
	userEditCmd.Flags().BoolVar(&user.EditUserProhibitLogin, "prohibit-login", false, "prevent the user from signing in")
	userEditCmd.Flags().BoolVarP(&user.EditUserRestricted, "restricted", "r", false, "restricted user")
	userEditCmd.Flags().BoolVar(&user.EditUserAllowCreateOrg, "allow-create-org", false, "user may create organizations")
	userEditCmd.Flags().IntVar(&user.EditUserMaxRepos, "max-repos", 0, "maximum number of repositories (-1: instance default)")
	userEditCmd.Flags().VarP(&user.EditUserVisibility, "visibility", "v", "user visibility")
	userEditCmd.MarkFlagsMutuallyExclusive("password-file", "passwd")
	_ = userEditCmd.RegisterFlagCompletionFunc("visibility", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"public", "limited", "private"}, cobra.ShellCompDirectiveNoFileComp
	})

}
