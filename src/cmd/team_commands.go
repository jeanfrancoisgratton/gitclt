// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/team_commands.go
// Original timestamp : 2026.10.26 07:31:42

package cmd

import (
	"fmt"
	"gitclt/access_mgmt/teams"
	"strconv"

	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Team management commands",
	Long: `Some of the commands are self-service, but most of the commands here need administrative rights.
In other words: you must have the appropriate access or auth token to perform some of the tasks.
Teams are addressed by their organization and their name (case-insensitive): <ORG> <TEAM>.`,
}

var teamListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt team list [-c CONNECTION] [--json] <ORG>",
	Short:   "Lists the teams of an organization",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := teams.ListTeams(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamGetCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"info"},
	Example: "  gitclt team get [-c CONNECTION] [--json] <ORG> <TEAM>\n" +
		"  gitclt team get [-c CONNECTION] [--json] <TEAM-ID>",
	Short: "Gives information about a team, by name or by ID",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var e error
		if len(args) == 2 {
			if _, ce := teams.TeamGet(args[0], args[1], true); ce != nil {
				e = ce
			}
		} else if id, perr := strconv.ParseInt(args[0], 10, 64); perr != nil {
			e = fmt.Errorf("%q is not a team ID; to get a team by name, use: gitclt team get <ORG> <TEAM>", args[0])
		} else if _, ce := teams.TeamGetByID(id, true); ce != nil {
			e = ce
		}
		if e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"add"},
	Example: "  gitclt team create [-c CONNECTION] [flags] <ORG> <TEAM>\n" +
		"  gitclt team create -p write --all-repos acme devs\n" +
		"  gitclt team create -u code=write,issues=write,wiki=read acme devs",
	Short: "Creates a team in an organization",
	Long: `The team's access is given either with --permission (the same level on every unit),
or with --units (a level per unit: none, read or write), but not both. Neither means read on every unit.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.CreateTeam(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt team edit [-c CONNECTION] [flags] <ORG> <TEAM>\n" +
		"  gitclt team edit --all-repos=false -u code=read acme devs",
	Short: "Edits a team; only the flags passed are changed",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --all-repos=false), and an empty string clears a text field (e.g. --description "").
The access is replaced as a whole by either --permission or --units, not both.
The owner team cannot be renamed, nor have its access changed.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.EditTeam(args[0], args[1], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt team delete [-c CONNECTION] [--yes] <ORG> <TEAM>",
	Short:   "Deletes a team (the owner team cannot be deleted)",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.DeleteTeam(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamMembersCmd = &cobra.Command{
	Use:     "members",
	Example: "  gitclt team members [-c CONNECTION] [--json] <ORG> <TEAM>",
	Short:   "Lists the members of a team",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := teams.ListMembers(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamIsMemberCmd = &cobra.Command{
	Use:     "ismember",
	Example: "  gitclt team ismember [-c CONNECTION] <ORG> <TEAM> <USER>",
	Short:   "Checks whether <USER> is a member of a team",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := teams.IsMemberOfTeam(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamAddMemberCmd = &cobra.Command{
	Use:     "addmember",
	Example: "  gitclt team addmember [-c CONNECTION] <ORG> <TEAM> <USER>",
	Short:   "Adds <USER> to a team, which also makes them a member of the organization",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.AddMemberToTeam(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamRemoveMemberCmd = &cobra.Command{
	Use:     "removemember",
	Aliases: []string{"rmmember"},
	Example: "  gitclt team removemember [-c CONNECTION] [--yes] <ORG> <TEAM> <USER>",
	Short:   "Removes <USER> from a team",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.RemoveMemberFromTeam(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamReposCmd = &cobra.Command{
	Use:     "repos",
	Example: "  gitclt team repos [-c CONNECTION] [--json] <ORG> <TEAM>",
	Short:   "Lists the repositories a team has access to",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := teams.ListRepos(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamAddRepoCmd = &cobra.Command{
	Use:     "addrepo",
	Example: "  gitclt team addrepo [-c CONNECTION] <ORG> <TEAM> <REPO>",
	Short:   "Gives a team access to <REPO>, a repository of the organization",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.AddRepoToTeam(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var teamRemoveRepoCmd = &cobra.Command{
	Use:     "removerepo",
	Aliases: []string{"rmrepo"},
	Example: "  gitclt team removerepo [-c CONNECTION] [--yes] <ORG> <TEAM> <REPO>",
	Short:   "Revokes a team's access to <REPO>; the repository itself is kept",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := teams.RemoveRepoFromTeam(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

// completeTeamPermission offers the values teams.PermissionFlag accepts
func completeTeamPermission(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"read", "write", "admin"}, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	teamCmd.AddCommand(teamListCmd, teamGetCmd, teamCreateCmd, teamEditCmd, teamDeleteCmd,
		teamMembersCmd, teamIsMemberCmd, teamAddMemberCmd, teamRemoveMemberCmd,
		teamReposCmd, teamAddRepoCmd, teamRemoveRepoCmd)

	teamListCmd.Flags().BoolVarP(&teams.JSONoutput, "json", "j", false, "output as JSON")
	teamGetCmd.Flags().BoolVarP(&teams.JSONoutput, "json", "j", false, "output as JSON")
	teamMembersCmd.Flags().BoolVarP(&teams.JSONoutput, "json", "j", false, "output as JSON")
	teamReposCmd.Flags().BoolVarP(&teams.JSONoutput, "json", "j", false, "output as JSON")
	teamDeleteCmd.Flags().BoolVarP(&teams.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
	teamRemoveMemberCmd.Flags().BoolVarP(&teams.AssumeYes, "yes", "y", false, "do not ask for confirmation before removing")
	teamRemoveRepoCmd.Flags().BoolVarP(&teams.AssumeYes, "yes", "y", false, "do not ask for confirmation before removing")

	teamCreateCmd.Flags().StringVarP(&teams.CreateTeamDescription, "description", "d", "", "team description")
	teamCreateCmd.Flags().VarP(&teams.CreateTeamPermission, "permission", "p", "same access level on every unit (default: read)")
	teamCreateCmd.Flags().StringVarP(&teams.CreateTeamUnits, "units", "u", "", "per-unit access, e.g. code=write,issues=read (none|read|write)")
	teamCreateCmd.Flags().BoolVarP(&teams.CreateTeamIncludesAllRepos, "all-repos", "a", false, "the team has access to every repository of the organization")
	teamCreateCmd.Flags().BoolVarP(&teams.CreateTeamCanCreateOrgRepo, "can-create-repos", "r", false, "team members can create repositories in the organization")
	_ = teamCreateCmd.RegisterFlagCompletionFunc("permission", completeTeamPermission)

	teamEditCmd.Flags().StringVarP(&teams.EditTeamName, "name", "n", "", "new team name")
	teamEditCmd.Flags().StringVarP(&teams.EditTeamDescription, "description", "d", "", "team description")
	teamEditCmd.Flags().VarP(&teams.EditTeamPermission, "permission", "p", "same access level on every unit")
	teamEditCmd.Flags().StringVarP(&teams.EditTeamUnits, "units", "u", "", "per-unit access, e.g. code=write,issues=read (none|read|write)")
	teamEditCmd.Flags().BoolVarP(&teams.EditTeamIncludesAllRepos, "all-repos", "a", false, "the team has access to every repository of the organization")
	teamEditCmd.Flags().BoolVarP(&teams.EditTeamCanCreateOrgRepo, "can-create-repos", "r", false, "team members can create repositories in the organization")
	_ = teamEditCmd.RegisterFlagCompletionFunc("permission", completeTeamPermission)
}
