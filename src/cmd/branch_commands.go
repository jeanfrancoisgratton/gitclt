// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/branch_commands.go
// Original timestamp : 2026.10.26 16:40:12

package cmd

import (
	"fmt"
	"gitclt/branch_mgmt/branches"

	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Branch management commands",
}

var branchListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt branch ls [-c CONNECTION] -o OWNER [-l | -r | -a] [--json] <REPO>\n" +
		"  gitclt branch ls -l",
	Short: "Lists a repository's branches: local, on the server, or both (the default)",
	Long: `The local branches are those of the git working copy in the current directory (or one of its parents);
when a repository is given, that working copy must be a clone of it.
--remote and --all need the owner (-o) and the repository; --local alone needs neither.
--all (or -l and -r together, or neither) lists both sides; it overrides -l and -r.
The branches are then merged by name, and their status tells if both sides point to the same commit.
If the current directory is not a working copy of the repository, --all shows the server's branches only.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := ""
		if len(args) > 0 {
			repo = args[0]
		}
		if _, e := branches.ListBranches(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	branchCmd.AddCommand(branchListCmd)

	branchCmd.PersistentFlags().StringVarP(&branches.OwnerOrg, "owner", "o", "", "user or organization that owns the repository")
	branchListCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")
	branchListCmd.Flags().BoolVarP(&branches.LocalOnly, "local", "l", false, "only list the local branches")
	branchListCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "only list the branches on the server")
	branchListCmd.Flags().BoolVarP(&branches.AllBranches, "all", "a", false, "list both the local branches and the server's, overriding -l and -r (default)")
}
