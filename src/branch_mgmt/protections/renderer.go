// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/renderer.go
// Original timestamp : 2026.11.04 22:00:00

package protections

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func allowedBlocked(b bool) string {
	if b {
		return "allowed"
	}
	return "blocked"
}

// GitHub lists a ruleset targeting more than one pattern, but it cannot be addressed by a pattern
func patternOf(r *gol.BranchProtection) string {
	if r.Pattern == "" {
		return "(several patterns)"
	}
	return r.Pattern
}

func noneIfEmpty(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// who words an allowlist: no list lets anyone with write access through, an empty one nobody
func who(a *gol.Allowlist) string {
	if a == nil {
		return "write access"
	}
	var parts []string
	if len(a.Users) > 0 {
		parts = append(parts, "users: "+strings.Join(a.Users, ", "))
	}
	if len(a.Teams) > 0 {
		parts = append(parts, "teams: "+strings.Join(a.Teams, ", "))
	}
	if len(parts) == 0 {
		return "nobody"
	}
	return strings.Join(parts, "; ")
}

func renderRuleTable(list []gol.BranchProtection) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Pattern", "Approvals", "Status checks", "Signed", "Admins too", "Direct push", "Deletion", "Force push"})
	for i := range list {
		r := &list[i]
		t.AppendRow(table.Row{patternOf(r), r.RequiredApprovals, strings.Join(r.RequiredStatusChecks, ", "), yesNo(r.RequireSignedCommits),
			yesNo(r.ApplyToAdmins), who(r.PushAllowlist), allowedBlocked(r.AllowDeletion), allowedBlocked(r.AllowForcePush)})
	}
	t.SetStyle(table.StyleColoredBlackOnCyanWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

// renderRule shows a rule, with the settings this forge has
func renderRule(r *gol.BranchProtection, backend gol.Backend) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	line := func(label, value string) { _, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label+":"), value) }

	line("Pattern", patternOf(r))
	if r.ID != 0 {
		line("Ruleset ID", fmt.Sprint(r.ID))
	}
	line("Required approvals", fmt.Sprint(r.RequiredApprovals))
	line("Dismiss stale approvals", yesNo(r.DismissStaleApprovals))
	line("Status checks", noneIfEmpty(strings.Join(r.RequiredStatusChecks, ", ")))
	line("Must be up to date", yesNo(r.RequireUpToDate))
	line("Signed commits", yesNo(r.RequireSignedCommits))
	line("Applies to admins", yesNo(r.ApplyToAdmins))
	line("Direct push", who(r.PushAllowlist))
	line("Deletion", allowedBlocked(r.AllowDeletion))
	line("Force push", allowedBlocked(r.AllowForcePush))

	switch backend {
	case gol.BackendGitHub:
		line("Linear history", yesNo(r.RequireLinearHistory))
		line("Conversation resolution", yesNo(r.RequireConversationResolution))
	case gol.BackendBitbucket:
		line("Merge", who(r.MergeAllowlist))
	case gol.BackendForgejo:
		line("Merge", who(r.MergeAllowlist))
		line("Approvals counted from", who(r.ApprovalsAllowlist))
		line("Blocked by rejected reviews", yesNo(r.BlockOnRejectedReviews))
		line("Blocked by review requests", yesNo(r.BlockOnOfficialReviewRequests))
		line("Protected files", noneIfEmpty(r.ProtectedFilePatterns))
		line("Unprotected files", noneIfEmpty(r.UnprotectedFilePatterns))
	}
	_ = w.Flush()
}
