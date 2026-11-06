// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/helpers.go
// Original timestamp : 2026.11.04 22:00:00

package protections

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
)

// connect opens the client on the selected connection, and decides the repository and its owner (see forge.Client.Target)
func connect(ctx context.Context, repo string) (*forge.Client, string, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", "", ce
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return nil, "", "", ce
	}
	return c, owner, repo, nil
}

// checkRepo makes sure owner/repo exists, so that a 404 further down is about the rule, not the repository
func checkRepo(ctx context.Context, c *forge.Client, owner, repo string) *cerr.CustomError {
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return nil
}

func noSuchRule(owner, repo, pattern string) *cerr.CustomError {
	return &cerr.CustomError{Title: "No such protection rule", Code: http.StatusNotFound,
		Message: fmt.Sprintf("%s/%s has no protection rule %s", owner, repo, pattern)}
}

// getRule fetches the rule of owner/repo addressed by pattern
func getRule(ctx context.Context, c *forge.Client, owner, repo, pattern string) (*gol.BranchProtection, *cerr.CustomError) {
	rule, e := c.BranchProtections().Get(ctx, owner, repo, pattern)
	if gol.IsNotFound(e) {
		return nil, noSuchRule(owner, repo, pattern)
	}
	if e != nil {
		return nil, forge.Error("Unable to get the protection rule "+pattern, e)
	}
	return rule, nil
}

// allowlistOf returns the allowlist behind l, creating an empty one if there is none
func allowlistOf(l **gol.Allowlist) *gol.Allowlist {
	if *l == nil {
		*l = &gol.Allowlist{Users: []string{}, Teams: []string{}}
	}
	return *l
}

// ruleEdits turns the flags explicitly passed into changes to a rule; changed is cmd.Flags().Changed
// create applies them to an empty rule, edit to the server's: the library's Update replaces the whole rule
func ruleEdits(changed func(string) bool) ([]func(*gol.BranchProtection), *cerr.CustomError) {
	if RequiredApprovals < 0 {
		return nil, &cerr.CustomError{Title: "Invalid approval count", Message: "--approvals cannot be negative"}
	}

	var edits []func(*gol.BranchProtection)
	on := func(flag string, edit func(r *gol.BranchProtection)) {
		if changed(flag) {
			edits = append(edits, edit)
		}
	}
	on("approvals", func(r *gol.BranchProtection) { r.RequiredApprovals = RequiredApprovals })
	on("dismiss-stale", func(r *gol.BranchProtection) { r.DismissStaleApprovals = DismissStaleApprovals })
	on("status-check", func(r *gol.BranchProtection) {
		// A blank value removes them all: this is a string array, as a check's name can have commas
		r.RequiredStatusChecks = slices.DeleteFunc(slices.Clone(StatusChecks), func(s string) bool { return strings.TrimSpace(s) == "" })
	})
	on("up-to-date", func(r *gol.BranchProtection) { r.RequireUpToDate = RequireUpToDate })
	on("signed", func(r *gol.BranchProtection) { r.RequireSignedCommits = RequireSignedCommits })
	on("apply-to-admins", func(r *gol.BranchProtection) { r.ApplyToAdmins = ApplyToAdmins })

	on("allow-deletion", func(r *gol.BranchProtection) { r.AllowDeletion = AllowDeletion })
	on("allow-force-push", func(r *gol.BranchProtection) { r.AllowForcePush = AllowForcePush })
	on("linear-history", func(r *gol.BranchProtection) { r.RequireLinearHistory = RequireLinearHistory })
	on("conversation-resolution", func(r *gol.BranchProtection) { r.RequireConversationResolution = RequireConversationResolution })

	on("block-rejected", func(r *gol.BranchProtection) { r.BlockOnRejectedReviews = BlockOnRejectedReviews })
	on("block-review-requests", func(r *gol.BranchProtection) { r.BlockOnOfficialReviewRequests = BlockOnReviewRequests })
	on("protected-files", func(r *gol.BranchProtection) { r.ProtectedFilePatterns = ProtectedFiles })
	on("unprotected-files", func(r *gol.BranchProtection) { r.UnprotectedFilePatterns = UnprotectedFiles })

	// An allowlist's users and teams change independently; --no-push=false and --any-xxx=false ask for nothing
	on("push-users", func(r *gol.BranchProtection) { allowlistOf(&r.PushAllowlist).Users = PushUsers })
	on("push-teams", func(r *gol.BranchProtection) { allowlistOf(&r.PushAllowlist).Teams = PushTeams })
	on("merge-users", func(r *gol.BranchProtection) { allowlistOf(&r.MergeAllowlist).Users = MergeUsers })
	on("merge-teams", func(r *gol.BranchProtection) { allowlistOf(&r.MergeAllowlist).Teams = MergeTeams })
	on("approver-users", func(r *gol.BranchProtection) { allowlistOf(&r.ApprovalsAllowlist).Users = ApproverUsers })
	on("approver-teams", func(r *gol.BranchProtection) { allowlistOf(&r.ApprovalsAllowlist).Teams = ApproverTeams })
	if NoPush {
		on("no-push", func(r *gol.BranchProtection) { r.PushAllowlist = &gol.Allowlist{Users: []string{}, Teams: []string{}} })
	}
	if AnyPush {
		on("any-push", func(r *gol.BranchProtection) { r.PushAllowlist = nil })
	}
	if AnyMerge {
		on("any-merge", func(r *gol.BranchProtection) { r.MergeAllowlist = nil })
	}
	if AnyApprover {
		on("any-approver", func(r *gol.BranchProtection) { r.ApprovalsAllowlist = nil })
	}
	return edits, nil
}

func unique(list []string) []string {
	var out []string
	for _, s := range list {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// confirm asks a yes/no question, unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	answer := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(question)))
	return answer == "y" || answer == "yes"
}
