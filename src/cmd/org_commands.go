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

var orgGetCmd = &cobra.Command{
	Use:     "get",
	Example: "gitclt org get [-c CONNECTION] [--json] <ORG-NAME>",
	Short:   "Fetches a given organization's information",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := organizations.OrgGet(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var orgDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Example: "gitclt org delete [-c CONNECTION] [--yes] <ORG-NAME>",
	Short:   "Deletes the named organization",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := organizations.DeleteOrg(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var orgRenameCmd = &cobra.Command{
	Use:     "rename",
	Aliases: []string{"mv"},
	Example: "gitclt org rename [-c CONNECTION] <CURRENT_ORG-NAME> <NEW_ORG-NAME>",
	Short:   "Changes (renames) an organization name",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := organizations.RenameOrg(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var orgCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"add"},
	Example: "gitclt org create [-c CONNECTION] [flags] <ORG-NAME>\ngitclt org create --owner jfg --visibility private acme",
	Short:   "Creates an organization; --owner other than yourself needs admin rights",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := organizations.CreateOrg(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var orgEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"set"},
	Example: "gitclt org edit [-c CONNECTION] [flags] <ORG-NAME>\ngitclt org edit --repo-admin-team-access=false --location Montreal acme",
	Short:   "Edits an organization settings; only the flags passed are changed",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --repo-admin-team-access=false), and an empty string clears a text field (e.g. --website "").`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := organizations.EditOrg(args[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	orgCmd.AddCommand(orgListCmd, orgListAllCmd, orgGetCmd, orgDeleteCmd, orgRenameCmd, orgCreateCmd, orgEditCmd)

	orgListCmd.Flags().BoolVarP(&organizations.JSONoutput, "json", "j", false, "output the list as JSON")
	orgListAllCmd.Flags().BoolVarP(&organizations.JSONoutput, "json", "j", false, "output the list as JSON")
	orgGetCmd.Flags().BoolVarP(&organizations.JSONoutput, "json", "j", false, "output the list as JSON")
	orgDeleteCmd.Flags().BoolVarP(&organizations.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgOwner, "owner", "o", "", "user who will own the organization (default: yourself)")
	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgFullName, "fullname", "f", "", "organization full name")
	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgDescription, "description", "d", "", "organization description")
	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgWebsite, "website", "w", "", "organization web site")
	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgEmail, "email", "e", "", "organization email")
	orgCreateCmd.Flags().StringVarP(&organizations.CreateOrgLocation, "location", "l", "", "organization location")
	orgCreateCmd.Flags().VarP(&organizations.CreateOrgVisibility, "visibility", "v", "organization visibility (default: server default)")
	orgCreateCmd.Flags().BoolVarP(&organizations.CreateOrgRepoAdminChangeTeamAccess, "repo-admin-team-access", "t", false, "repository admins can change team access")

	orgEditCmd.Flags().StringVarP(&organizations.EditOrgFullName, "fullname", "f", "", "organization full name")
	orgEditCmd.Flags().StringVarP(&organizations.EditOrgDescription, "description", "d", "", "organization description")
	orgEditCmd.Flags().StringVarP(&organizations.EditOrgWebsite, "website", "w", "", "organization web site")
	orgEditCmd.Flags().StringVarP(&organizations.EditOrgEmail, "email", "e", "", "organization email")
	orgEditCmd.Flags().StringVarP(&organizations.EditOrgLocation, "location", "l", "", "organization location")
	orgEditCmd.Flags().VarP(&organizations.EditOrgVisibility, "visibility", "v", "organization visibility")
	orgEditCmd.Flags().BoolVarP(&organizations.EditOrgRepoAdminChangeTeamAccess, "repo-admin-team-access", "t", false, "repository admins can change team access")

	for _, c := range []*cobra.Command{orgCreateCmd, orgEditCmd} {
		_ = c.RegisterFlagCompletionFunc("visibility", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"public", "limited", "private"}, cobra.ShellCompDirectiveNoFileComp
		})
	}
}
