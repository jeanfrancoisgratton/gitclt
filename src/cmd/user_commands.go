// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/user_commands.go
// Original timestamp : 2026.10.22 01:33:41

package cmd

import (
	"fmt"
	"gitclt/access_mgmt/users"

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
	Example: "  gitclt user whoami [-c CONNECTION] [--extended] [--json]",
	Short:   "Gives information about the current user",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.WhoAmI(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userGetCmd = &cobra.Command{
	Use:     "get <USER>",
	Aliases: []string{"info"},
	Example: "  gitclt user get [-c CONNECTION] [--extended] [--json] <USER>",
	Short:   "Gives information about a user (needs admin rights, Forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.UserGet(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt user list [-c CONNECTION] [--json]",
	Short:   "Lists all users on the forge (needs admin rights, Forgejo only)",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.ListUsers(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userDeleteCmd = &cobra.Command{
	Use:     "delete <USER>",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt user delete [-c CONNECTION] [--purge] [--yes] <USER>",
	Short:   "Deletes a user (needs admin rights, Forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.DeleteUser(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userWipeCmd = &cobra.Command{
	Use:     "wipe <USER>",
	Example: "  gitclt user wipe [-c CONNECTION] [--purge] [--dry-run] [--yes] <USER>",
	Short:   "Removes a user from all of their organizations and teams, then deletes them (needs admin rights, Forgejo only)",
	Long: `Everything is checked before anything is changed; the wipe stops, and changes nothing, if:
  - the connection's user is not a site administrator, or is the user to wipe
  - the user is the last owner of an organization that still has other members or repositories
  - the user owns repositories, and --purge was not passed
An organization whose only member is the user, and which owns no repositories, is deleted as well.
The plan is shown, then confirmed once (--yes skips the confirmation, --dry-run stops after the plan).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.WipeUser(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userCreateCmd = &cobra.Command{
	Use:     "create <USER>",
	Aliases: []string{"add"},
	Example: "  gitclt user create [-c CONNECTION] -e EMAIL [flags] <USER>",
	Short:   "Creates a user (needs admin rights, Forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var mustChange *bool // nil: server default (true on Forgejo)
		if cmd.Flags().Changed("must-change-passwd") {
			v := users.CreateUserMustChangePasswd
			mustChange = &v
		}
		if e := users.CreateUser(args[0], mustChange); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userEditCmd = &cobra.Command{
	Use:     "edit <USER>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt user edit [-c CONNECTION] [flags] <USER>\n" +
		"  gitclt user edit --admin=false --location Montreal jfg2",
	Short: "Edits a user; only the flags passed are changed (needs admin rights, Forgejo only)",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --admin=false), and an empty string clears a text field (e.g. --website "").`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.EditUser(args[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userEmailCmd = &cobra.Command{
	Use:     "email",
	Aliases: []string{"emails"},
	Short:   "Manages your own e-mail addresses",
	Long:    "Acts on the connection's user, and needs no admin rights. Bitbucket can only list the addresses.",
}

var userEmailListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt user email list [-c CONNECTION] [--json]",
	Short:   "Lists your e-mail addresses",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.ListEmails(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userEmailAddCmd = &cobra.Command{
	Use:     "add <ADDRESS>...",
	Aliases: []string{"create"},
	Example: "  gitclt user email add [-c CONNECTION] [--json] <ADDRESS>...",
	Short:   "Adds e-mail addresses to your account (GitHub sends each a verification e-mail; Forgejo does if the server requires it)",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.AddEmails(args); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userEmailDeleteCmd = &cobra.Command{
	Use:     "delete <ADDRESS>...",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt user email delete [-c CONNECTION] [--yes] <ADDRESS>...",
	Short:   "Removes e-mail addresses from your account; the primary one cannot be removed",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.DeleteEmails(args); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userSSHKeyCmd = &cobra.Command{
	Use:     "sshkey",
	Aliases: []string{"sshkeys", "ssh"},
	Short:   "Manages your own public SSH keys",
	Long:    "Acts on the connection's user, and needs no admin rights.",
}

var userSSHKeyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt user sshkey list [-c CONNECTION] [--json]",
	Short:   "Lists your public SSH keys",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.ListSSHKeys(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userSSHKeyGetCmd = &cobra.Command{
	Use:     "get <TITLE>",
	Aliases: []string{"info"},
	Example: "  gitclt user sshkey get [-c CONNECTION] [--json] <TITLE>\n" +
		"  gitclt user sshkey get --id 42",
	Short: "Gives information about one of your public SSH keys",
	Args:  sshKeyTargetArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := users.GetSSHKey(sshKeyTarget(args), true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userSSHKeyAddCmd = &cobra.Command{
	Use:     "add <TITLE> <PUBLIC-KEY-FILE>",
	Aliases: []string{"create"},
	Example: "  gitclt user sshkey add [-c CONNECTION] [--json] <TITLE> <PUBLIC-KEY-FILE>\n" +
		"  gitclt user sshkey add laptop ~/.ssh/id_ed25519.pub",
	Short: "Adds a public SSH key to your account",
	Long:  "The file must hold one public key; anything else, such as a private key, is refused before contacting the server.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.AddSSHKey(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userSSHKeyDeleteCmd = &cobra.Command{
	Use:     "delete <TITLE>",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt user sshkey delete [-c CONNECTION] [--yes] <TITLE>\n" +
		"  gitclt user sshkey delete --id 42",
	Short: "Removes a public SSH key from your account",
	Args:  sshKeyTargetArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.DeleteSSHKey(sshKeyTarget(args)); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userAvatarCmd = &cobra.Command{
	Use:   "avatar",
	Short: "Manages your own avatar (Forgejo only)",
	Long:  "Acts on the connection's user, and needs no admin rights. GitHub and Bitbucket have no API for this.",
}

var userAvatarSetCmd = &cobra.Command{
	Use:     "set <IMAGE-FILE>",
	Aliases: []string{"upload"},
	Example: "  gitclt user avatar set [-c CONNECTION] <IMAGE-FILE>",
	Short:   "Uploads an image (at most 10 MiB and 4096x4096) as your avatar (Forgejo only)",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.SetAvatar(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var userAvatarDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt user avatar delete [-c CONNECTION] [--yes]",
	Short:   "Restores the default avatar (Forgejo only)",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if e := users.DeleteAvatar(); e != nil {
			fmt.Println(e.Error())
		}
	},
}

// A key is designated either by its title, or with --id, as titles are not always unique
func sshKeyTargetArgs(cmd *cobra.Command, args []string) error {
	if !cmd.Flags().Changed("id") {
		return cobra.ExactArgs(1)(cmd, args)
	}
	if len(args) != 0 {
		return fmt.Errorf("pass either the key's title, or --id")
	}
	if users.SSHKeyID <= 0 {
		return fmt.Errorf("--id must be a positive number")
	}
	return nil
}

func sshKeyTarget(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func init() {
	userCmd.AddCommand(userWhoAmICmd, userGetCmd, userListCmd, userDeleteCmd, userWipeCmd, userCreateCmd, userEditCmd,
		userEmailCmd, userSSHKeyCmd, userAvatarCmd)
	userEmailCmd.AddCommand(userEmailListCmd, userEmailAddCmd, userEmailDeleteCmd)
	userSSHKeyCmd.AddCommand(userSSHKeyListCmd, userSSHKeyGetCmd, userSSHKeyAddCmd, userSSHKeyDeleteCmd)
	userAvatarCmd.AddCommand(userAvatarSetCmd, userAvatarDeleteCmd)

	userEmailListCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userEmailAddCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userEmailDeleteCmd.Flags().BoolVarP(&users.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	userSSHKeyListCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userSSHKeyGetCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userSSHKeyGetCmd.Flags().Int64Var(&users.SSHKeyID, "id", 0, "designate the key by its ID instead of its title")
	userSSHKeyAddCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userSSHKeyDeleteCmd.Flags().Int64Var(&users.SSHKeyID, "id", 0, "designate the key by its ID instead of its title")
	userSSHKeyDeleteCmd.Flags().BoolVarP(&users.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	userAvatarDeleteCmd.Flags().BoolVarP(&users.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	userWhoAmICmd.Flags().BoolVarP(&users.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userWhoAmICmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userGetCmd.Flags().BoolVarP(&users.ExtendedInfo, "extended", "x", false, "provide extended user information")
	userGetCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userListCmd.Flags().BoolVarP(&users.JSONoutput, "json", "j", false, "output as JSON")
	userDeleteCmd.Flags().BoolVarP(&users.PurgeUser, "purge", "p", false, "purge all of the user's contents (repos, etc)")
	userDeleteCmd.Flags().BoolVarP(&users.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
	userWipeCmd.Flags().BoolVarP(&users.PurgeUser, "purge", "p", false, "also delete the repositories (and other content) the user owns")
	userWipeCmd.Flags().BoolVarP(&users.AssumeYes, "yes", "y", false, "do not ask for confirmation before wiping")
	userWipeCmd.Flags().BoolVarP(&users.WipeDryRun, "dry-run", "n", false, "show what the wipe would do, and stop")

	userCreateCmd.Flags().StringVarP(&users.CreateUserEmail, "email", "e", "", "user email")
	userCreateCmd.Flags().StringVarP(&users.CreateUserPasswdFile, "password-file", "p", "", "read the password from file")
	userCreateCmd.Flags().StringVarP(&users.CreateUserFullName, "fullname", "f", "", "user full name")
	userCreateCmd.Flags().BoolVarP(&users.CreateUserMustChangePasswd, "must-change-passwd", "m", false, "created user must change password")
	userCreateCmd.Flags().BoolVarP(&users.CreateUserRestricted, "restricted", "r", false, "restricted user")
	userCreateCmd.Flags().BoolVarP(&users.CreateUserSendNotification, "send-notification", "s", false, "send an email notification to the new user")
	userCreateCmd.Flags().VarP(&users.CreateUserVisibility, "visibility", "v", "user visibility (default: server default)")
	userCreateCmd.Flags().StringVarP(&users.CreateUserDescription, "description", "d", "", "user description")
	userCreateCmd.Flags().StringVarP(&users.CreateUserWebsite, "website", "w", "", "user web site")
	userCreateCmd.Flags().StringVarP(&users.CreateUserLocation, "location", "l", "", "user location")
	userCreateCmd.Flags().BoolVarP(&users.CreateUserAdmin, "admin", "a", false, "make the user a site administrator")
	_ = userCreateCmd.RegisterFlagCompletionFunc("visibility", completeVisibility)
	_ = userCreateCmd.MarkFlagRequired("email")

	userEditCmd.Flags().StringVarP(&users.EditUserEmail, "email", "e", "", "user email")
	userEditCmd.Flags().StringVarP(&users.EditUserFullName, "fullname", "f", "", "user full name")
	userEditCmd.Flags().StringVarP(&users.EditUserDescription, "description", "d", "", "user description")
	userEditCmd.Flags().StringVarP(&users.EditUserWebsite, "website", "w", "", "user web site")
	userEditCmd.Flags().StringVarP(&users.EditUserLocation, "location", "l", "", "user location")
	userEditCmd.Flags().StringVarP(&users.EditUserPasswdFile, "password-file", "p", "", "read the new password from file")
	userEditCmd.Flags().BoolVarP(&users.EditUserPromptPasswd, "passwd", "P", false, "prompt for a new password")
	userEditCmd.Flags().BoolVarP(&users.EditUserMustChangePasswd, "must-change-passwd", "m", false, "user must change password at next login")
	userEditCmd.Flags().BoolVarP(&users.EditUserAdmin, "admin", "a", false, "site administrator")
	userEditCmd.Flags().BoolVar(&users.EditUserActive, "active", false, "account is active")
	userEditCmd.Flags().BoolVar(&users.EditUserProhibitLogin, "prohibit-login", false, "prevent the user from signing in")
	userEditCmd.Flags().BoolVarP(&users.EditUserRestricted, "restricted", "r", false, "restricted user")
	userEditCmd.Flags().BoolVar(&users.EditUserAllowCreateOrg, "allow-create-org", false, "user may create organizations")
	userEditCmd.Flags().IntVar(&users.EditUserMaxRepos, "max-repos", 0, "maximum number of repositories (-1: instance default)")
	userEditCmd.Flags().VarP(&users.EditUserVisibility, "visibility", "v", "user visibility")
	_ = userEditCmd.RegisterFlagCompletionFunc("visibility", completeVisibility)
	userEditCmd.MarkFlagsMutuallyExclusive("password-file", "passwd")
}
