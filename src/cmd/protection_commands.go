// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/protection_commands.go
// Original timestamp : 2026.11.04 22:00:00

package cmd

import (
	"fmt"
	"gitclt/branch_mgmt/protections"

	"github.com/spf13/cobra"
)

// Some settings only exist on some forges: said on the commands having these flags
const protectionForgeNotes = `
Some settings only exist on some forges; the others refuse them:
  GitHub, Bitbucket:  --allow-deletion, --allow-force-push
  GitHub:             --linear-history, --conversation-resolution
  Forgejo, Bitbucket: --merge-users, --merge-teams
  Forgejo:            --approver-users, --approver-teams, --block-rejected, --block-review-requests, --protected-files, --unprotected-files
Bitbucket has no status checks, up-to-date nor signed commits requirement, and always applies its rules to admins: --apply-to-admins is ignored there.
GitHub cannot exempt individual users from a rule, only teams: --push-teams, and not --push-users.
Bitbucket knows its users by display name, nickname, account ID or {UUID}, not by workspace nor e-mail: gitclt user whoami -x shows yours.
On GitHub, --approvals, --dismiss-stale and --conversation-resolution also block direct pushes, whatever the push allowlist says;
and administrators can still push despite --no-push, unless --apply-to-admins is passed.`

var protectionCmd = &cobra.Command{
	Use:     "protection",
	Aliases: []string{"protections", "protect", "prot"},
	Short:   "Branch protection rule commands",
	Long: `A rule protects the branches matching its pattern: a branch name or a glob (main, release/*); rules are addressed by it.
On GitHub, a rule is a branch ruleset; a ruleset targeting several patterns is listed, but cannot be addressed.`,
}

var protectionListCmd = &cobra.Command{
	Use:     "list <REPO>",
	Aliases: []string{"ls"},
	Example: "  gitclt branch protection list [-c CONNECTION] [-o OWNER] [-j] <REPO>",
	Short:   "Lists the branch protection rules of a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := protections.ListProtections(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var protectionGetCmd = &cobra.Command{
	Use:     "get <REPO> <PATTERN>",
	Aliases: []string{"info"},
	Example: "  gitclt branch protection get [-c CONNECTION] [-o OWNER] [-j] <REPO> <PATTERN>\n" +
		"  gitclt branch prot get myrepo 'release/*'",
	Short: "Shows a branch protection rule",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := protections.GetProtection(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var protectionCreateCmd = &cobra.Command{
	Use:     "create <REPO> <PATTERN>",
	Aliases: []string{"add"},
	Example: "  gitclt branch protection create [-c CONNECTION] [-o OWNER] [flags] <REPO> <PATTERN>\n" +
		"  gitclt branch prot add --approvals 2 --no-push myrepo main\n" +
		"  gitclt branch prot add --status-check ci/build --status-check ci/test --up-to-date myrepo 'release/*'",
	Short: "Creates a branch protection rule",
	Long: `Without any flag, the rule still blocks deleting the matching branches and force-pushing to them.
Direct pushes stay open to anyone with write access, unless --no-push or --push-users / --push-teams say otherwise.` + protectionForgeNotes,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := protections.CreateProtection(args[0], args[1], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var protectionEditCmd = &cobra.Command{
	Use:     "edit <REPO> <PATTERN>",
	Aliases: []string{"update"},
	Example: "  gitclt branch protection edit [-c CONNECTION] [-o OWNER] [flags] <REPO> <PATTERN>\n" +
		"  gitclt branch prot edit --approvals 1 --signed=false myrepo main\n" +
		"  gitclt branch prot edit --any-push --status-check '' myrepo main",
	Short: "Edits a branch protection rule; only the flags passed are changed",
	Long: `Yes/no settings are turned off with =false (--signed=false). A list is replaced by the values passed:
--push-users '' empties it, and --status-check '' removes all status checks. An allowlist left without users nor teams lets nobody through;
--any-push, --any-merge and --any-approver remove the allowlist itself.
A rule's pattern cannot be changed: delete the rule, and create a new one.
On GitHub, a rule is written back as a whole ruleset: the ruleset's own name, its enforcement status (a disabled one becomes active)
and any setting gitclt does not know of are overwritten. Check the rulesets that were not made with gitclt before editing them.` + protectionForgeNotes,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := protections.EditProtection(args[0], args[1], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var protectionDeleteCmd = &cobra.Command{
	Use:     "delete <REPO> <PATTERN>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt branch protection delete [-c CONNECTION] [-o OWNER] [-y] <REPO> <PATTERN>...\n" +
		"  gitclt branch prot rm myrepo main 'release/*'",
	Short: "Deletes branch protection rules",
	Long: `Every rule is checked before the confirmation. The run stops at the first failure; the rules deleted before it stay deleted.
On GitHub, a ruleset's settings that gitclt does not know of go with it.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := protections.DeleteProtections(args[0], args[1:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

// The settings of a rule, for both create and edit
func protectionRuleFlags(c *cobra.Command) {
	c.Flags().IntVar(&protections.RequiredApprovals, "approvals", 0, "approvals required to merge a pull request")
	c.Flags().BoolVar(&protections.DismissStaleApprovals, "dismiss-stale", false, "a new push dismisses the earlier approvals")
	c.Flags().StringArrayVar(&protections.StatusChecks, "status-check", nil, "status check that must pass (repeatable)")
	c.Flags().BoolVar(&protections.RequireUpToDate, "up-to-date", false, "the branch must be up to date with the base before merging (GitHub: needs status checks)")
	c.Flags().BoolVar(&protections.RequireSignedCommits, "signed", false, "commits must be signed")
	c.Flags().BoolVar(&protections.ApplyToAdmins, "apply-to-admins", false, "the rule also applies to repository administrators")

	c.Flags().BoolVar(&protections.NoPush, "no-push", false, "block direct pushes: changes go through pull requests")
	c.Flags().StringSliceVar(&protections.PushUsers, "push-users", nil, "only these users may push directly (repeatable, or comma-separated)")
	c.Flags().StringSliceVar(&protections.PushTeams, "push-teams", nil, "only these teams may push directly (repeatable, or comma-separated)")
	c.MarkFlagsMutuallyExclusive("no-push", "push-users")
	c.MarkFlagsMutuallyExclusive("no-push", "push-teams")

	c.Flags().BoolVar(&protections.AllowDeletion, "allow-deletion", false, "allow deleting the branches (GitHub, Bitbucket)")
	c.Flags().BoolVar(&protections.AllowForcePush, "allow-force-push", false, "allow force pushes (GitHub, Bitbucket)")
	c.Flags().BoolVar(&protections.RequireLinearHistory, "linear-history", false, "forbid merge commits (GitHub)")
	c.Flags().BoolVar(&protections.RequireConversationResolution, "conversation-resolution", false, "review conversations must be resolved before merging (GitHub)")

	c.Flags().StringSliceVar(&protections.MergeUsers, "merge-users", nil, "only these users may merge pull requests (Forgejo, Bitbucket)")
	c.Flags().StringSliceVar(&protections.MergeTeams, "merge-teams", nil, "only these teams may merge pull requests (Forgejo, Bitbucket)")

	c.Flags().StringSliceVar(&protections.ApproverUsers, "approver-users", nil, "only the approvals of these users count (Forgejo)")
	c.Flags().StringSliceVar(&protections.ApproverTeams, "approver-teams", nil, "only the approvals of these teams count (Forgejo)")
	c.Flags().BoolVar(&protections.BlockOnRejectedReviews, "block-rejected", false, "a rejected review blocks the merge (Forgejo)")
	c.Flags().BoolVar(&protections.BlockOnReviewRequests, "block-review-requests", false, "a pending official review request blocks the merge (Forgejo)")
	c.Flags().StringVar(&protections.ProtectedFiles, "protected-files", "", "semicolon-separated globs of files that cannot be changed (Forgejo)")
	c.Flags().StringVar(&protections.UnprotectedFiles, "unprotected-files", "", "semicolon-separated globs of files that do not need approvals (Forgejo)")
}

func init() {
	branchCmd.AddCommand(protectionCmd)
	protectionCmd.AddCommand(protectionListCmd, protectionGetCmd, protectionCreateCmd, protectionEditCmd, protectionDeleteCmd)

	// Takes over branchCmd's own -o, which is tied to the branches package
	protectionCmd.PersistentFlags().StringVarP(&protections.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")

	protectionListCmd.Flags().BoolVarP(&protections.JSONoutput, "json", "j", false, "output as JSON")
	protectionGetCmd.Flags().BoolVarP(&protections.JSONoutput, "json", "j", false, "output as JSON")
	protectionDeleteCmd.Flags().BoolVarP(&protections.AssumeYes, "yes", "y", false, "confirm deletion")

	protectionRuleFlags(protectionCreateCmd)
	protectionRuleFlags(protectionEditCmd)
	protectionEditCmd.Flags().BoolVar(&protections.AnyPush, "any-push", false, "remove the push allowlist: anyone with write access may push")
	protectionEditCmd.Flags().BoolVar(&protections.AnyMerge, "any-merge", false, "remove the merge allowlist: anyone with write access may merge")
	protectionEditCmd.Flags().BoolVar(&protections.AnyApprover, "any-approver", false, "remove the approvers allowlist: the approvals of anyone with write access count")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-push", "no-push")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-push", "push-users")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-push", "push-teams")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-merge", "merge-users")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-merge", "merge-teams")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-approver", "approver-users")
	protectionEditCmd.MarkFlagsMutuallyExclusive("any-approver", "approver-teams")
}
