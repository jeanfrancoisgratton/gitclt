// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/pr_commands.go
// Original timestamp : 2026.10.30 01:00:00

package cmd

import (
	"fmt"
	"gitclt/pr_mgmt/pullrequests"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:     "pr",
	Aliases: []string{"pull", "pullrequest"},
	Short:   "Pull request commands",
	Long: `Pull requests, on the server; their branches must already be pushed.
<NUMBER> is the pull request's number, with or without a leading # (12 or #12).`,
}

var prListCmd = &cobra.Command{
	Use:     "list <REPO>",
	Aliases: []string{"ls"},
	Example: "  gitclt pr list [-c CONNECTION] [-o OWNER] [-s open|closed|all] [-j] <REPO>",
	Short:   "Lists the pull requests of a repository: open ones by default",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := pullrequests.ListPRs(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prGetCmd = &cobra.Command{
	Use:     "get <REPO> <NUMBER>",
	Aliases: []string{"info"},
	Example: "  gitclt pr get [-c CONNECTION] [-o OWNER] [-j] <REPO> <NUMBER>",
	Short:   "Shows a pull request: branches, author, status, mergeability, reviews and description",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := pullrequests.GetPR(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prCreateCmd = &cobra.Command{
	Use:     "create <REPO> <HEAD> [<BASE>]",
	Aliases: []string{"add", "open"},
	Example: "  gitclt pr create [-c CONNECTION] [-o OWNER] [-t TITLE] [-b BODY | -F FILE] [-d] [-r USER,...] <REPO> <HEAD> [<BASE>]\n" +
		"  gitclt pr create -t \"Add the pr commands\" -r alice,bob myrepo feature/pr develop",
	Short: "Opens a pull request, from <HEAD> into <BASE> (default: the default branch)",
	Long: `<HEAD> is the branch with the changes, <BASE> the branch to merge into; both must be on the server (push them first).
The title defaults to the head branch's name. -d, --draft opens it as a draft (on Forgejo, a "WIP: " title prefix).
-r, --reviewers requests the review of these users once the pull request is open.`,
	Args: cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		base := ""
		if len(args) > 2 {
			base = args[2]
		}
		if e := pullrequests.CreatePR(args[0], args[1], base, cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prEditCmd = &cobra.Command{
	Use:     "edit <REPO> <NUMBER>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt pr edit [-c CONNECTION] [-o OWNER] [flags] <REPO> <NUMBER>\n" +
		"  gitclt pr edit --draft=false --base main myrepo 12",
	Short: "Edits a pull request; only the flags passed are changed",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
--draft=false marks a draft ready for review, --draft turns it back into a draft. An empty string clears the body (--body "").`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.EditPR(args[0], args[1], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prMergeCmd = &cobra.Command{
	Use: "merge <REPO> <NUMBER>",
	Example: "  gitclt pr merge [-c CONNECTION] [-o OWNER] [-m METHOD] [-d] [--auto | --cancel-auto] <REPO> <NUMBER>\n" +
		"  gitclt pr merge -m squash -d myrepo 12",
	Short: "Merges a pull request",
	Long: `-m, --method is merge (the default: a merge commit), squash, rebase, rebase-merge (rebase, then a merge commit),
fast-forward-only, or manually-merged (only marks as merged a pull request whose commits are already in the base);
GitHub only offers merge, squash and rebase.
The merge fails if the head branch moved since gitclt checked it. Drafts, closed and conflicting pull requests are refused.
--auto schedules the merge for when the checks pass (not on Bitbucket); --cancel-auto cancels such a schedule.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.MergePR(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prCloseCmd = &cobra.Command{
	Use:     "close <REPO> <NUMBER>...",
	Example: "  gitclt pr close [-c CONNECTION] [-o OWNER] <REPO> <NUMBER>...",
	Short:   "Closes pull requests without merging them",
	Long: `Every pull request is checked first; those already closed or merged are skipped.
On Bitbucket, a closed (declined) pull request cannot be reopened.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.ClosePRs(args[0], args[1:], false); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prReopenCmd = &cobra.Command{
	Use:     "reopen <REPO> <NUMBER>...",
	Example: "  gitclt pr reopen [-c CONNECTION] [-o OWNER] <REPO> <NUMBER>...",
	Short:   "Reopens closed pull requests; not on Bitbucket",
	Long:    `Every pull request is checked first; those already open are skipped, and a merged one is refused.`,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.ClosePRs(args[0], args[1:], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prReviewCmd = &cobra.Command{
	Use: "review <REPO> <NUMBER>",
	Example: "  gitclt pr review [-c CONNECTION] [-o OWNER] (-a | -x | -n) [-m TEXT] <REPO> <NUMBER>\n" +
		"  gitclt -c reviewer pr review -a myrepo 12\n" +
		"  gitclt pr review -x -m \"Please add tests\" myrepo 12",
	Short: "Reviews a pull request, as the connection's user: approve, request changes, or comment",
	Long: `Pass exactly one of -a (--approve), -x (--request-changes), -n (--comment); the last two need a text (-m).
The authors of a pull request cannot approve it, nor request changes on it (Forgejo, GitHub): review it with another
user's connection (-c).`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.ReviewPR(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var prReviewersCmd = &cobra.Command{
	Use: "reviewers <REPO> <NUMBER> <USER>...",
	Example: "  gitclt pr reviewers [-c CONNECTION] [-o OWNER] [--remove] <REPO> <NUMBER> <USER>...\n" +
		"  gitclt pr reviewers myrepo 12 alice bob",
	Short: "Requests the review of users on a pull request (--remove withdraws the requests)",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := pullrequests.SetReviewers(args[0], args[1], args[2:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	prCmd.AddCommand(prListCmd, prGetCmd, prCreateCmd, prEditCmd, prMergeCmd, prCloseCmd, prReopenCmd, prReviewCmd, prReviewersCmd)

	prCmd.PersistentFlags().StringVarP(&pullrequests.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	prListCmd.Flags().StringVarP(&pullrequests.ListState, "state", "s", "open", "open, closed (merged included) or all")
	prListCmd.Flags().BoolVarP(&pullrequests.JSONoutput, "json", "j", false, "output as JSON")
	prGetCmd.Flags().BoolVarP(&pullrequests.JSONoutput, "json", "j", false, "output as JSON")

	prCreateCmd.Flags().StringVarP(&pullrequests.PRTitle, "title", "t", "", "title (default: the head branch's name)")
	prCreateCmd.Flags().StringVarP(&pullrequests.PRBody, "body", "b", "", "description (Markdown)")
	prCreateCmd.Flags().StringVarP(&pullrequests.PRBodyFile, "body-file", "F", "", "read the description from this file")
	prCreateCmd.Flags().BoolVarP(&pullrequests.PRDraft, "draft", "d", false, "open the pull request as a draft")
	prCreateCmd.Flags().StringSliceVarP(&pullrequests.PRReviewers, "reviewers", "r", nil, "users whose review is requested (comma-separated, or repeated)")
	prCreateCmd.MarkFlagsMutuallyExclusive("body", "body-file")

	prEditCmd.Flags().StringVarP(&pullrequests.PRTitle, "title", "t", "", "title")
	prEditCmd.Flags().StringVarP(&pullrequests.PRBody, "body", "b", "", "description (Markdown)")
	prEditCmd.Flags().StringVarP(&pullrequests.PRBodyFile, "body-file", "F", "", "read the description from this file")
	prEditCmd.Flags().StringVar(&pullrequests.PRBase, "base", "", "branch to merge into")
	prEditCmd.Flags().BoolVarP(&pullrequests.PRDraft, "draft", "d", false, "make it a draft (--draft=false marks it ready for review)")
	prEditCmd.MarkFlagsMutuallyExclusive("body", "body-file")

	prMergeCmd.Flags().StringVarP(&pullrequests.MergeMethod, "method", "m", "merge", "merge, squash, rebase, rebase-merge, fast-forward-only or manually-merged")
	prMergeCmd.Flags().StringVar(&pullrequests.MergeTitle, "title", "", "title of the merge or squash commit (default: the server's)")
	prMergeCmd.Flags().StringVar(&pullrequests.MergeMessage, "message", "", "message of the merge or squash commit (default: the server's)")
	prMergeCmd.Flags().BoolVarP(&pullrequests.DeleteBranch, "delete-branch", "d", false, "delete the head branch on the server once merged")
	prMergeCmd.Flags().BoolVar(&pullrequests.AutoMerge, "auto", false, "merge once the checks pass, instead of now")
	prMergeCmd.Flags().BoolVar(&pullrequests.CancelAutoMerge, "cancel-auto", false, "cancel a merge scheduled with --auto")
	prMergeCmd.MarkFlagsMutuallyExclusive("auto", "cancel-auto")

	prReviewCmd.Flags().BoolVarP(&pullrequests.ReviewApprove, "approve", "a", false, "approve the pull request")
	prReviewCmd.Flags().BoolVarP(&pullrequests.ReviewRequestChanges, "request-changes", "x", false, "request changes (needs -m)")
	prReviewCmd.Flags().BoolVarP(&pullrequests.ReviewComment, "comment", "n", false, "comment only (needs -m)")
	prReviewCmd.Flags().StringVarP(&pullrequests.ReviewBody, "message", "m", "", "text of the review")
	prReviewCmd.MarkFlagsMutuallyExclusive("approve", "request-changes", "comment")

	prReviewersCmd.Flags().BoolVar(&pullrequests.RemoveReviewers, "remove", false, "withdraw the review requests instead")
}
