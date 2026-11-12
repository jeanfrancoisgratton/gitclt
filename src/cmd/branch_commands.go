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
	Long: `list and get also show the branches of the local clone, read straight from its .git directory;
delete and rename only act on the server. gitclt never writes to a local clone, and never needs the git client.`,
}

var branchListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt branch list [-c CONNECTION] [-o OWNER] [-l | -r | -a] [--json] [<REPO>]\n" +
		"  gitclt branch ls -l\n" +
		"  gitclt branch ls   # from inside a clone",
	Short: "Lists a repository's branches: local, on the server, or both (the default)",
	Long: `The local branches are those of the git working copy in the current directory (or one of its parents);
when a repository is given, that working copy must be a clone of it.
Without a repository, it is the one the current directory is a clone of; --local alone needs none, and reads the working copy.
The owner defaults to the clone's, for its repository; else the connection's owner, else your username.
--all (or -l and -r together, or neither) lists both sides; it overrides -l and -r.
The branches are then merged by name, and their status tells if both sides point to the same commit.
If the current directory is not a working copy of the repository, --all shows the server's branches only.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := branches.ListBranches(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchDeleteCmd = &cobra.Command{
	Use:     "delete -r [<REPO>] <BRANCH>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt branch delete [-c CONNECTION] [-o OWNER] -r [-y] [<REPO>] <BRANCH>...\n" +
		"  gitclt branch rm -r myrepo feature/x feature/y",
	Short: "Deletes repository branches on the server (local branches are left to git)",
	Long: `This deletes branches on the server only: gitclt never deletes a local branch, nor needs the git client.
For a local branch, use git branch -d (or -D).
-r (--remote) is required: before 1.8.0, this command deleted local branches by default, and a bare command must not
now delete on the server by mistake.
The run stops at the first failure; the branches deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, names := splitRepo(args, 1)
		if e := branches.DeleteBranch(repo, names); e != nil {
			printTargetsError(e)
		}
	},
}

var branchGetCmd = &cobra.Command{
	Use: "get [<REPO>] <BRANCH>",
	Example: "  gitclt branch get [-c CONNECTION] [-o OWNER] [--json] [<REPO>] <BRANCH>\n" +
		"  gitclt branch get myrepo feature/x",
	Short: "Shows extended information about a repository branch",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := branches.GetBranch(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchRenameCmd = &cobra.Command{
	Use:     "rename -r [<REPO>] <BRANCH> <NEW-NAME>",
	Aliases: []string{"mv"},
	Example: "  gitclt branch rename [-c CONNECTION] [-o OWNER] -r [<REPO>] <BRANCH> <NEW-NAME>\n" +
		"  gitclt branch mv -r myrepo dev develop",
	Short: "Renames a repository branch on the server (local branches are left to git)",
	Long: `This renames a branch on the server only: gitclt never renames a local branch, nor needs the git client.
For a local branch, use git branch -m; a clone tracking the old name can then follow with
git branch -u origin/NEW-NAME (after a git fetch).
-r (--remote) is required: before 1.8.0, this command renamed the local branch by default, and a bare command must not
now rename on the server by mistake.
The open pull requests and the protection rules follow the branch.`,
	Args: cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 2)
		if e := branches.RenameBranch(repo, rest[0], rest[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var branchCompareCmd = &cobra.Command{
	Use:     "compare [<REPO>] <BASE> <HEAD>",
	Aliases: []string{"cmp"},
	Example: "  gitclt branch compare [-c CONNECTION] [-o OWNER] [--json] [<REPO>] <BASE> <HEAD>\n" +
		"  gitclt branch cmp -o myorg myrepo develop feature/x",
	Short: "Compares two branches on the server: how far ahead of, and behind, the base the head is",
	Long: `Base and head can be branches, tags or commits; the comparison is the server's.
Ahead counts the commits of head that base lacks, behind the reverse.
When head is only ahead, it can be merged into base as a fast-forward.`,
	Args: cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 2)
		if _, e := branches.CompareBranches(repo, rest[0], rest[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	branchCmd.AddCommand(branchListCmd, branchDeleteCmd, branchGetCmd, branchRenameCmd, branchCompareCmd)

	branchCmd.PersistentFlags().StringVarP(&branches.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	branchListCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")
	branchListCmd.Flags().BoolVarP(&branches.LocalOnly, "local", "l", false, "only list the local branches")
	branchListCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "only list the branches on the server")
	branchListCmd.Flags().BoolVarP(&branches.AllBranches, "all", "a", false, "list both the local branches and the server's, overriding -l and -r (default)")
	branchListCmd.Flags().BoolVarP(&branches.SuppressWarnings, "warnings", "w", false, "suppress all warnings")

	branchDeleteCmd.Flags().BoolVarP(&branches.AssumeYes, "yes", "y", false, "confirm deletion")
	branchDeleteCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "delete the branches on the server (required: local branches are left to git)")

	branchGetCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")

	branchRenameCmd.Flags().BoolVarP(&branches.RemoteOnly, "remote", "r", false, "rename the branch on the server (required: local branches are left to git)")

	branchCompareCmd.Flags().BoolVarP(&branches.JSONoutput, "json", "j", false, "output as JSON")

	addHelp(branchCmd, repoInferenceHelp)
	addHelp(branchDeleteCmd, confirmNamesRepoHelp)
}
