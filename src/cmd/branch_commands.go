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
	Use:     "branch",
	Aliases: []string{"bra"},
	Short:   "Branch management commands",
}

var branchListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt branch list [-c CONNECTION] [-o OWNER] [-l | -r | -a] [--json] <REPO>\n" +
		"  gitclt branch ls -l",
	Short: "Lists a repository's branches: local, on the server, or both (the default)",
	Long: `The local branches are those of the git working copy in the current directory (or one of its parents);
when a repository is given, that working copy must be a clone of it.
--remote and --all need the repository; --local alone does not. The owner defaults to the connection's owner, else your username.
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

var branchDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt branch delete [-c CONNECTION] [-o OWNER] [-l | -r | -a] [-f] [-y] <REPO> <BRANCH>...\n" +
		"  gitclt branch rm -a myrepo feature/x feature/y",
	Short: "Deletes repository branches: local (the default), on the server, or both",
	Long: `The local branches are those of the git working copy in the current directory (or one of its parents),
which must be a clone of the repository; the git client does the deletion, so it must be installed.
git refuses to delete the checked out branch, or a branch that is not merged (unless --force is passed).
--all (or -l and -r together) deletes on both sides, local first.
The run stops at the first failure; the branches deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := branches.DeleteBranch(args[0], args[1:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchGetCmd = &cobra.Command{
	Use: "get",
	Example: "  gitclt branch get [-c CONNECTION] [-o OWNER] [--json] <REPO> <BRANCH>\n" +
		"  gitclt branch get myrepo feature/x",
	Short: "Shows extended information about a repository branch",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := branches.GetBranch(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchRenameCmd = &cobra.Command{
	Use:     "rename",
	Aliases: []string{"mv"},
	Example: "  gitclt branch rename [-c CONNECTION] [-o OWNER] [-l | -r | -a] <REPO> <BRANCH> <NEW NAME>\n" +
		"  gitclt branch mv -a myrepo dev develop",
	Short: "Renames a repository branch: local (the default), on the server, or both",
	Long: `The local branches are those of the git working copy in the current directory (or one of its parents),
which must be a clone of the repository; the git client does the renaming, so it must be installed.
On the server, the open pull requests and the protection rules follow the branch.
--all (or -l and -r together) renames on both sides, local first,
and the local branch then tracks the server's new name.`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := branches.RenameBranch(args[0], args[1], args[2]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchCompareCmd = &cobra.Command{
	Use:     "compare",
	Aliases: []string{"cmp"},
	Example: "  gitclt branch compare [-c CONNECTION] [-o OWNER] [--json] <REPO> <BASE> <HEAD>\n" +
		"  gitclt branch cmp -o myorg myrepo develop feature/x",
	Short: "Compares two branches on the server: how far ahead of, and behind, the base the head is",
	Long: `Base and head can be branches, tags or commits; the comparison is the server's.
Ahead counts the commits of head that base lacks, behind the reverse.
When head is only ahead, it can be merged into base as a fast-forward.`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := branches.CompareBranches(args[0], args[1], args[2], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	branchCmd.AddCommand(branchListCmd, branchDeleteCmd, branchGetCmd, branchRenameCmd, branchCompareCmd)

	branchCmd.PersistentFlags().StringVarP(&branches.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	branchListCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")
	branchListCmd.Flags().BoolVarP(&branches.LocalOnly, "local", "l", false, "only list the local branches")
	branchListCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "only list the branches on the server")
	branchListCmd.Flags().BoolVarP(&branches.AllBranches, "all", "a", false, "list both the local branches and the server's, overriding -l and -r (default)")
	branchListCmd.Flags().BoolVarP(&branches.SuppressWarnings, "warnings", "w", false, "suppress all warnings")

	branchDeleteCmd.Flags().BoolVarP(&branches.AssumeYes, "yes", "y", false, "confirm deletion")
	branchDeleteCmd.Flags().BoolVarP(&branches.LocalOnly, "local", "l", false, "delete the local branches (default)")
	branchDeleteCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "delete the branches on the server")
	branchDeleteCmd.Flags().BoolVarP(&branches.AllBranches, "all", "a", false, "delete both the local branches and the server's, overriding -l and -r")
	branchDeleteCmd.Flags().BoolVarP(&branches.ForceDelete, "force", "f", false, "delete local branches even if they are not merged")

	branchGetCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")

	branchRenameCmd.Flags().BoolVarP(&branches.LocalOnly, "local", "l", false, "rename the local branch (default)")
	branchRenameCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "rename the branch on the server")
	branchRenameCmd.Flags().BoolVarP(&branches.AllBranches, "all", "a", false, "rename both the local branch and the server's, overriding -l and -r")

	branchCompareCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")
}
