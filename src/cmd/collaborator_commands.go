// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/collaborator_commands.go
// Original timestamp : 2026.11.06 18:00:00

package cmd

import (
	"fmt"
	"gitclt/repo_mgmt/collaborators"

	"github.com/spf13/cobra"
)

var collaboratorCmd = &cobra.Command{
	Use:     "collaborator",
	Aliases: []string{"collaborators", "collab", "collabs"},
	Short:   "Repository collaborator commands: the users with their own access to a repository",
	Long: `These need admin access to the repository; access through a team is managed with gitclt team.
On Bitbucket, a user is a nickname of a member of the workspace, an account ID or a {UUID}: to add someone from outside the workspace, use one of the last two.`,
}

var collaboratorListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt collaborator list [-c CONNECTION] [-o OWNER] [-j] [<REPO>]",
	Short:   "Lists the collaborators; -q outputs only their logins",
	Long: `Those with access through a team are not listed. GitHub and Bitbucket usually include the owner, Forgejo does not.
On Forgejo, permissions stay empty unless you are an admin of the repository; a pending GitHub invitation is not listed: see invitations.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := collaborators.List(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var collaboratorGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <USER>",
	Aliases: []string{"info"},
	Example: "  gitclt collaborator get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <USER>",
	Short:   "Shows a collaborator's access; anybody else is an error",
	Long:    "On GitHub, this also finds those with access through the organization.",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := collaborators.Get(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var collaboratorAddCmd = &cobra.Command{
	Use:     "add [<REPO>] <USER>",
	Aliases: []string{"create", "set", "update"},
	Example: "  gitclt collaborator add [-c CONNECTION] [-o OWNER] [-p PERMISSION] [<REPO>] <USER>\n" +
		"  gitclt collaborator add -p read alice",
	Short: "Gives a user access to a repository, or changes that of an existing collaborator",
	Long: `-p, --permission is read, write (the default) or admin; GitHub also has triage and maintain.
GitHub usually invites a new collaborator, who has no access until they accept: the command says so, and see invitations.
GitHub ignores the permission on a repository owned by a user, as opposed to an organization.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := collaborators.Add(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var collaboratorRemoveCmd = &cobra.Command{
	Use:     "remove [<REPO>] <USER>...",
	Aliases: []string{"rm", "delete", "del"},
	Example: "  gitclt collaborator remove [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <USER>...\n" +
		"  gitclt collaborator rm . alice bob",
	Short: "Takes users' access to a repository away, after confirmation",
	Long: `Every user is checked before asking, as the forges take removing a stranger for a success; on GitHub, this also withdraws a pending invitation.
The run stops at the first failure; the users removed before it stay removed.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, users := splitRepo(args, 1)
		if e := collaborators.Remove(repo, users); e != nil {
			printTargetsError(e)
		}
	},
}

var collaboratorInvitationsCmd = &cobra.Command{
	Use:     "invitations [<REPO>]",
	Aliases: []string{"invites"},
	Example: "  gitclt collaborator invitations [-c CONNECTION] [-o OWNER] [-j] [<REPO>]",
	Short:   "Lists who was invited and has not accepted yet (GitHub only)",
	Long:    "The other forges give access at once. To withdraw an invitation, see gitclt collaborator remove.",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := collaborators.Invitations(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	collaboratorCmd.AddCommand(collaboratorListCmd, collaboratorGetCmd, collaboratorAddCmd, collaboratorRemoveCmd, collaboratorInvitationsCmd)
	collaboratorCmd.PersistentFlags().StringVarP(&collaborators.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	collaboratorListCmd.Flags().BoolVarP(&collaborators.JSONoutput, "json", "j", false, "output as JSON")
	collaboratorGetCmd.Flags().BoolVarP(&collaborators.JSONoutput, "json", "j", false, "output as JSON")
	collaboratorInvitationsCmd.Flags().BoolVarP(&collaborators.JSONoutput, "json", "j", false, "output as JSON")
	collaboratorAddCmd.Flags().StringVarP(&collaborators.Permission, "permission", "p", "write", "read, write or admin; GitHub also has triage and maintain")
	_ = collaboratorAddCmd.RegisterFlagCompletionFunc("permission", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"read", "write", "admin", "triage", "maintain"}, cobra.ShellCompDirectiveNoFileComp
	})
	collaboratorRemoveCmd.Flags().BoolVarP(&collaborators.AssumeYes, "yes", "y", false, "do not ask for confirmation before removing")
	addHelp(collaboratorCmd, repoInferenceHelp)
}
