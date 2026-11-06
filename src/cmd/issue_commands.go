// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/issue_commands.go
// Original timestamp : 2026.11.06 14:00:00

package cmd

import (
	"fmt"
	"gitclt/issue_mgmt/issues"

	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:     "issue",
	Aliases: []string{"issues"},
	Short:   "Issue commands, with their comments, labels and milestones (Forgejo, GitHub)",
	Long: `Issues, on the server; Bitbucket Cloud retired its issue tracker.
<NUMBER> is the issue's number, with or without a leading # (12 or #12).
Issues and pull requests share one number sequence: get, close, reopen, delete and lock refuse a pull request's number; edit and comment accept it,
which is how a pull request gets labels, a milestone or comments.`,
}

var issueListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt issue list [-c CONNECTION] [-o OWNER] [-s open|closed|all] [-l LABEL,...] [-j] [<REPO>]\n" +
		"  gitclt issue ls -s all -l bug",
	Short: "Lists the issues (open ones by default), without the pull requests; -q outputs only their numbers",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := issues.ListIssues(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <NUMBER>",
	Aliases: []string{"info"},
	Example: "  gitclt issue get [-c CONNECTION] [-o OWNER] [--comments] [-j] [<REPO>] <NUMBER>\n" +
		"  gitclt issue get --comments 12",
	Short: "Shows an issue, with its text; --comments adds what was said on it",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := issues.GetIssue(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCreateCmd = &cobra.Command{
	Use:     "create [<REPO>]",
	Aliases: []string{"add", "open"},
	Example: "  gitclt issue create [-c CONNECTION] [-o OWNER] -t TITLE [-b BODY | -F FILE] [-l LABEL,...] [-a USER,...] [-m MILESTONE] [-j] [<REPO>]\n" +
		"  gitclt issue create -t \"Crash on empty list\" -l bug -a alice -m v1.5",
	Short: "Opens an issue; -q outputs only its number",
	Long: `The labels must already exist: see gitclt issue label. --milestone takes a title or an ID.
On a Forgejo server, a label can also be one of the organization's.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if e := issues.CreateIssue(repo, cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueEditCmd = &cobra.Command{
	Use:     "edit [<REPO>] <NUMBER>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt issue edit [-c CONNECTION] [-o OWNER] [flags] [<REPO>] <NUMBER>\n" +
		"  gitclt issue edit --add-label bug --remove-label triage 12\n" +
		"  gitclt issue edit --milestone \"\" 12",
	Short: "Edits an issue; only the flags passed are changed",
	Long: `--label and --assignee replace the whole list, and "" clears it; --add-label and --remove-label change the labels one by one,
and cannot be mixed with --label. --milestone takes a title or an ID, and "" removes it. Use close and reopen for the state.
--add-label and --remove-label are separate requests: if they fail, what came before them stays changed.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.EditIssue(repo, rest[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCloseCmd = &cobra.Command{
	Use: "close [<REPO>] <NUMBER>...",
	Example: "  gitclt issue close [-c CONNECTION] [-o OWNER] [<REPO>] <NUMBER>...\n" +
		"  gitclt issue close 12 13",
	Short: "Closes issues; those already closed are skipped",
	Long:  "Every number is checked before anything changes; the run stops at the first failure, and the issues closed before it stay closed.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, numbers := splitRepoNumber(args, 1)
		if e := issues.CloseIssues(repo, numbers, false); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueReopenCmd = &cobra.Command{
	Use:     "reopen [<REPO>] <NUMBER>...",
	Example: "  gitclt issue reopen [-c CONNECTION] [-o OWNER] [<REPO>] <NUMBER>...",
	Short:   "Reopens issues; those already open are skipped",
	Long:    "Every number is checked before anything changes; the run stops at the first failure, and the issues reopened before it stay open.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, numbers := splitRepoNumber(args, 1)
		if e := issues.CloseIssues(repo, numbers, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <NUMBER>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt issue delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <NUMBER>...",
	Short:   "Deletes issues for good, with their comments, after confirmation (needs repository admin rights)",
	Long: `This cannot be undone; closing an issue is usually what is wanted. Every number is checked before asking, and the question names
the repository and the issues' titles. The run stops at the first failure; the issues deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, numbers := splitRepoNumber(args, 1)
		if e := issues.DeleteIssues(repo, numbers); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLockCmd = &cobra.Command{
	Use:     "lock [<REPO>] <NUMBER>",
	Example: "  gitclt issue lock [-c CONNECTION] [-o OWNER] [-r REASON] [<REPO>] <NUMBER>",
	Short:   "Locks an issue's conversation (Forgejo 17 and later, GitHub)",
	Long: `Forgejo requires a reason, and takes any text. GitHub does not require one, and only takes off-topic, "too heated", resolved or spam.
Locking a locked issue changes nothing.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.LockIssue(repo, rest[0], false); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueUnlockCmd = &cobra.Command{
	Use:     "unlock [<REPO>] <NUMBER>",
	Example: "  gitclt issue unlock [-c CONNECTION] [-o OWNER] [<REPO>] <NUMBER>",
	Short:   "Unlocks an issue's conversation (Forgejo 17 and later, GitHub)",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.LockIssue(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCommentCmd = &cobra.Command{
	Use:     "comment",
	Aliases: []string{"comments"},
	Short:   "Comment commands: what was said on an issue",
}

var issueCommentListCmd = &cobra.Command{
	Use:     "list [<REPO>] <NUMBER>",
	Aliases: []string{"ls"},
	Example: "  gitclt issue comment list [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <NUMBER>",
	Short:   "Lists an issue's comments; -q outputs only their IDs",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := issues.ListComments(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCommentAddCmd = &cobra.Command{
	Use:     "add [<REPO>] <NUMBER>",
	Aliases: []string{"create"},
	Example: "  gitclt issue comment add [-c CONNECTION] [-o OWNER] (-b BODY | -F FILE) [<REPO>] <NUMBER>\n" +
		"  gitclt issue comment add -b \"Fixed in v1.5\" 12",
	Short: "Comments on an issue; -q outputs only the comment's ID",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.AddComment(repo, rest[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCommentEditCmd = &cobra.Command{
	Use:     "edit [<REPO>] <ID>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt issue comment edit [-c CONNECTION] [-o OWNER] (-b BODY | -F FILE) [<REPO>] <ID>",
	Short:   "Replaces a comment's text",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.EditComment(repo, rest[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueCommentDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <ID>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt issue comment delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <ID>...",
	Short:   "Deletes comments, after confirmation",
	Long:    "The run stops at the first failure; the comments deleted before it stay deleted.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, ids := splitRepoNumber(args, 1)
		if e := issues.DeleteComments(repo, ids); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLabelCmd = &cobra.Command{
	Use:     "label",
	Aliases: []string{"labels"},
	Short:   "Label commands: a repository's labels, addressed by name",
	Long: `To put labels on an issue, or take them off, see gitclt issue edit.
Forgejo lets labels share a name, while its API only tells them by name: gitclt does not create such duplicates, and refuses to touch those it meets.`,
}

var issueLabelListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt issue label list [-c CONNECTION] [-o OWNER] [-j] [<REPO>]",
	Short:   "Lists the labels; -q outputs only their names",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := issues.ListLabelsOf(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLabelGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <NAME>",
	Aliases: []string{"info"},
	Example: "  gitclt issue label get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <NAME>",
	Short:   "Shows a label",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := issues.GetLabel(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLabelCreateCmd = &cobra.Command{
	Use:     "create [<REPO>] <NAME>",
	Aliases: []string{"add"},
	Example: "  gitclt issue label create [-c CONNECTION] [-o OWNER] -C COLOR [-d DESCRIPTION] [-j] [<REPO>] <NAME>\n" +
		"  gitclt issue label create -C d73a4a -d \"Something is broken\" bug",
	Short: "Creates a label",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.CreateLabel(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLabelEditCmd = &cobra.Command{
	Use:     "edit [<REPO>] <NAME>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt issue label edit [-c CONNECTION] [-o OWNER] [-n NEW-NAME] [-C COLOR] [-d DESCRIPTION] [<REPO>] <NAME>",
	Short:   "Edits a label; only the flags passed are changed",
	Long:    `--name renames the label, which follows on every issue carrying it; "" clears the description.`,
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.EditLabel(repo, rest[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueLabelDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <NAME>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt issue label delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <NAME>...\n" +
		"  gitclt issue label rm . wontfix duplicate",
	Short: "Deletes labels, after confirmation; they leave every issue carrying them",
	Long:  "Every name is checked before asking. The run stops at the first failure; the labels deleted before it stay deleted.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, names := splitRepo(args, 1)
		if e := issues.DeleteLabels(repo, names); e != nil {
			printTargetsError(e)
		}
	},
}

var issueMilestoneCmd = &cobra.Command{
	Use:     "milestone",
	Aliases: []string{"milestones", "ms"},
	Short:   "Milestone commands: a repository's milestones, addressed by title or ID",
	Long: `A title comes first: a milestone whose title looks like another's ID is found by its title. To set an issue's milestone, see gitclt issue edit.
Forgejo lets milestones share a title: gitclt does not create such duplicates, and asks for the ID when it meets some.`,
}

var issueMilestoneListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt issue milestone list [-c CONNECTION] [-o OWNER] [-s open|closed|all] [-j] [<REPO>]",
	Short:   "Lists the milestones (open ones by default); -q outputs only their titles",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := issues.ListMilestones(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueMilestoneGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <MILESTONE>",
	Aliases: []string{"info"},
	Example: "  gitclt issue milestone get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <MILESTONE>",
	Short:   "Shows a milestone",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := issues.GetMilestone(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueMilestoneCreateCmd = &cobra.Command{
	Use:     "create [<REPO>] <TITLE>",
	Aliases: []string{"add"},
	Example: "  gitclt issue milestone create [-c CONNECTION] [-o OWNER] [-d DESCRIPTION] [--due DATE] [-j] [<REPO>] <TITLE>\n" +
		"  gitclt issue milestone create --due 2026-12-31 v1.5",
	Short: "Creates a milestone; -q outputs only its ID",
	Long:  "--due is a day (YYYY-MM-DD, meaning the end of that day where the command runs) or an RFC 3339 timestamp.",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.CreateMilestone(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueMilestoneEditCmd = &cobra.Command{
	Use:     "edit [<REPO>] <MILESTONE>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt issue milestone edit [-c CONNECTION] [-o OWNER] [-t TITLE] [-d DESCRIPTION] [--due DATE] [-s open|closed] [<REPO>] <MILESTONE>\n" +
		"  gitclt issue milestone edit -s closed v1.5",
	Short: "Edits a milestone; only the flags passed are changed",
	Long:  `--state closes or reopens it; "" clears the description. --due takes the same dates as create; the API cannot remove a due date.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := issues.EditMilestone(repo, rest[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var issueMilestoneDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <MILESTONE>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt issue milestone delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <MILESTONE>...",
	Short:   "Deletes milestones, after confirmation; their issues stay, without a milestone",
	Long:    "Every milestone is checked before asking. The run stops at the first failure; the milestones deleted before it stay deleted.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepoNumber(args, 1)
		if e := issues.DeleteMilestones(repo, rest); e != nil {
			printTargetsError(e)
		}
	},
}

const issueNumberHelp = `A number is never taken for <REPO>, so several need no . (gitclt issue close 12 13);
a repository with a numeric name cannot be designated by this command.`

func init() {
	issueCmd.AddCommand(issueListCmd, issueGetCmd, issueCreateCmd, issueEditCmd, issueCloseCmd, issueReopenCmd, issueDeleteCmd,
		issueLockCmd, issueUnlockCmd, issueCommentCmd, issueLabelCmd, issueMilestoneCmd)
	issueCommentCmd.AddCommand(issueCommentListCmd, issueCommentAddCmd, issueCommentEditCmd, issueCommentDeleteCmd)
	issueLabelCmd.AddCommand(issueLabelListCmd, issueLabelGetCmd, issueLabelCreateCmd, issueLabelEditCmd, issueLabelDeleteCmd)
	issueMilestoneCmd.AddCommand(issueMilestoneListCmd, issueMilestoneGetCmd, issueMilestoneCreateCmd, issueMilestoneEditCmd, issueMilestoneDeleteCmd)

	issueCmd.PersistentFlags().StringVarP(&issues.OwnerOrg, "owner", "o", "", ownerFlagHelp)

	issueListCmd.Flags().StringVarP(&issues.ListState, "state", "s", "open", "open, closed or all")
	issueListCmd.Flags().StringSliceVarP(&issues.ListLabels, "label", "l", nil, "only the issues carrying all these labels (comma-separated, or repeated)")
	issueListCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueGetCmd.Flags().BoolVar(&issues.WithComments, "comments", false, "also show what was said on the issue")
	issueGetCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")

	for _, cmd := range []*cobra.Command{issueCreateCmd, issueEditCmd} {
		cmd.Flags().StringVarP(&issues.IssueTitle, "title", "t", "", "title")
		cmd.Flags().StringVarP(&issues.IssueBody, "body", "b", "", "text (Markdown)")
		cmd.Flags().StringVarP(&issues.IssueBodyFile, "body-file", "F", "", "read the text from this file")
		cmd.Flags().StringSliceVarP(&issues.IssueLabels, "label", "l", nil, "labels, by name (comma-separated, or repeated)")
		cmd.Flags().StringSliceVarP(&issues.IssueAssignees, "assignee", "a", nil, "users it is assigned to (comma-separated, or repeated)")
		cmd.Flags().StringVarP(&issues.IssueMilestone, "milestone", "m", "", "milestone, by title or ID")
		cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	}
	_ = issueCreateCmd.MarkFlagRequired("title")
	issueCreateCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueEditCmd.Flags().StringSliceVar(&issues.IssueAddLabels, "add-label", nil, "labels to add, by name (comma-separated, or repeated)")
	issueEditCmd.Flags().StringSliceVar(&issues.IssueRemoveLabels, "remove-label", nil, "labels to remove, by name (comma-separated, or repeated)")
	issueEditCmd.MarkFlagsMutuallyExclusive("label", "add-label")
	issueEditCmd.MarkFlagsMutuallyExclusive("label", "remove-label")

	issueDeleteCmd.Flags().BoolVarP(&issues.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
	issueLockCmd.Flags().StringVarP(&issues.LockReason, "reason", "r", "", "why it is locked (required on Forgejo)")

	issueCommentListCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	for _, cmd := range []*cobra.Command{issueCommentAddCmd, issueCommentEditCmd} {
		cmd.Flags().StringVarP(&issues.IssueBody, "body", "b", "", "text (Markdown)")
		cmd.Flags().StringVarP(&issues.IssueBodyFile, "body-file", "F", "", "read the text from this file")
		cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	}
	issueCommentDeleteCmd.Flags().BoolVarP(&issues.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	issueLabelListCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueLabelGetCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueLabelCreateCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	for _, cmd := range []*cobra.Command{issueLabelCreateCmd, issueLabelEditCmd} {
		cmd.Flags().StringVarP(&issues.LabelColor, "color", "C", "", "six hexadecimal digits, with or without # (ex: d73a4a)")
		cmd.Flags().StringVarP(&issues.LabelDescription, "description", "d", "", "description")
	}
	_ = issueLabelCreateCmd.MarkFlagRequired("color")
	issueLabelEditCmd.Flags().StringVarP(&issues.LabelName, "name", "n", "", "new name")
	issueLabelDeleteCmd.Flags().BoolVarP(&issues.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	issueMilestoneListCmd.Flags().StringVarP(&issues.MilestoneListState, "state", "s", "open", "open, closed or all")
	issueMilestoneListCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueMilestoneGetCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	issueMilestoneCreateCmd.Flags().BoolVarP(&issues.JSONoutput, "json", "j", false, "output as JSON")
	for _, cmd := range []*cobra.Command{issueMilestoneCreateCmd, issueMilestoneEditCmd} {
		cmd.Flags().StringVarP(&issues.MilestoneDescription, "description", "d", "", "description")
		cmd.Flags().StringVar(&issues.MilestoneDue, "due", "", "due date: YYYY-MM-DD, or an RFC 3339 timestamp")
	}
	issueMilestoneEditCmd.Flags().StringVarP(&issues.MilestoneTitle, "title", "t", "", "new title")
	issueMilestoneEditCmd.Flags().StringVarP(&issues.MilestoneState, "state", "s", "", "open or closed")
	issueMilestoneDeleteCmd.Flags().BoolVarP(&issues.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	addHelp(issueCmd, repoInferenceHelp)
	for _, c := range []*cobra.Command{issueCloseCmd, issueReopenCmd, issueDeleteCmd, issueCommentDeleteCmd, issueMilestoneDeleteCmd} {
		addHelp(c, issueNumberHelp)
	}
}
