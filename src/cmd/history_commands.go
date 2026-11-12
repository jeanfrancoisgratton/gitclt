// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/history_commands.go
// Original timestamp : 2026.10.30 15:10:00

package cmd

import (
	"fmt"
	"gitclt/history"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:     "history",
	Aliases: []string{"log", "commits"},
	Short:   "Commit history commands",
	Long: `A repository's commits, read from the server like git log would: no local clone is needed,
and only the commits that were pushed are seen.`,
}

var historyListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt history list [-c CONNECTION] [-o OWNER] [-r REF] [-p PATH] [-n COUNT] [--since DATE] [--until DATE] [-j] [<REPO>]\n" +
		"  gitclt history ls -r develop -p src/history -n 10 myrepo",
	Short: "Lists the commits of a repository, newest first",
	Long: `This is the server's history: commits that are not pushed yet are not in it.
The commits are those reachable from --ref (a branch, tag or commit). Without --ref, from a clone of the repository,
the checked out branch is listed, if the server has it; otherwise, the default branch.
From a clone, a notice says when your local branch has commits the server does not, or is behind it.
--path keeps the commits that touched this file or directory.
--since and --until take a day (YYYY-MM-DD, local time; --until includes the whole day) or an RFC 3339 timestamp;
only GitHub can filter by date. -n 0 lists the whole history. Merge commits are marked with M.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := history.List(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var historyGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <REF>",
	Aliases: []string{"show", "info"},
	Example: "  gitclt history get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <REF>\n" +
		"  gitclt history show myrepo d0fe552",
	Short: "Shows a commit: author, parents, full message and changed files",
	Long: `<REF> is a commit SHA (full or abbreviated), a branch or a tag; the last two show their head commit.
Forgejo reports no per-file line counts, only the commit's totals, and does not detect renames.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := history.Get(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	historyCmd.AddCommand(historyListCmd, historyGetCmd)

	historyCmd.PersistentFlags().StringVarP(&history.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	historyListCmd.Flags().StringVarP(&history.ListRef, "ref", "r", "", "branch, tag or commit to start from (default: the checked out branch, from a clone; else the default branch)")
	historyListCmd.Flags().StringVarP(&history.ListPath, "path", "p", "", "only the commits that touched this file or directory")
	historyListCmd.Flags().IntVarP(&history.ListLimit, "limit", "n", 30, "maximum number of commits to list (0: all)")
	historyListCmd.Flags().StringVar(&history.ListSince, "since", "", "only the commits authored on or after this date (GitHub only)")
	historyListCmd.Flags().StringVar(&history.ListUntil, "until", "", "only the commits authored on or before this date (GitHub only)")
	historyListCmd.Flags().BoolVarP(&history.JSONoutput, "json", "j", false, "output as JSON")
	historyGetCmd.Flags().BoolVarP(&history.JSONoutput, "json", "j", false, "output as JSON")

	addHelp(historyCmd, repoInferenceHelp)
}
