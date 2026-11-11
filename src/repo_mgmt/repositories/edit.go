// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/repositories/edit.go
// Original timestamp : 2026.10.29 15:10:00

package repositories

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"slices"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditRepo only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
// Bitbucket cannot archive a repository, nor make it a template, and has no features nor settings: the library refuses those flags there,
// as it refuses a feature or setting the forge lacks, before any change
func EditRepo(reponame string, changed func(string) bool) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	ero, ce := editOptionsFromFlags(changed)
	if ce != nil {
		return ce
	}
	if ero == (gol.EditRepoOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt repo edit --help"}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	owner, reponame, ce := c.Target(ctx, OwnerOrg, reponame)
	if ce != nil {
		return ce
	}

	r, e := c.Repos().Edit(ctx, owner, reponame, ero)
	if e != nil {
		// The server answered success but kept its own value for some of it (Forgejo with Actions turned off for the
		// instance, GitHub's wiki or auto-merge on a free private repository): the rest went through
		if r != nil && errors.Is(e, gol.ErrNotSupported) {
			fmt.Println(hftx.WarningSign("Repository " + hftx.Brown(owner+"/"+r.Name) + " was updated, but: " + e.Error()))
			return nil
		}
		return forge.Error(fmt.Sprintf("Unable to edit the repository %s/%s", owner, reponame), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated repository " + hftx.Brown(owner+"/"+r.Name)))
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) (gol.EditRepoOptions, *cerr.CustomError) {
	var ero gol.EditRepoOptions

	strFlags := []struct {
		name string
		val  string
		dst  **string
	}{
		{"name", EditRepoName, &ero.Name},
		{"description", EditRepoDescription, &ero.Description},
		{"website", EditRepoWebsite, &ero.Website},
		{"default-branch", EditRepoDefaultBranch, &ero.DefaultBranch},
		{"external-wiki", EditExternalWiki, &ero.Wiki.ExternalURL},
	}
	for _, f := range strFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	boolFlags := []struct {
		name string
		val  bool
		dst  **bool
	}{
		{"archived", EditRepoArchived, &ero.Archived},
		{"template", EditRepoTemplate, &ero.Template},

		{"actions", EditActions, &ero.Features.Actions},
		{"issues", EditIssues, &ero.Features.Issues},
		{"wiki", EditWiki, &ero.Features.Wiki},
		{"pull-requests", EditPullRequests, &ero.Features.PullRequests},
		{"projects", EditProjects, &ero.Features.Projects},
		{"releases", EditReleases, &ero.Features.Releases},
		{"packages", EditPackages, &ero.Features.Packages},
		{"discussions", EditDiscussions, &ero.Features.Discussions},

		{"merge-commit", EditMergeCommit, &ero.Merge.MergeCommit},
		{"rebase", EditRebase, &ero.Merge.Rebase},
		{"rebase-merge", EditRebaseMerge, &ero.Merge.RebaseMerge},
		{"squash", EditSquash, &ero.Merge.Squash},
		{"fast-forward-only", EditFastForwardOnly, &ero.Merge.FastForwardOnly},
		{"delete-branch", EditDeleteBranch, &ero.Merge.DeleteBranch},
		{"auto-merge", EditAutoMerge, &ero.Merge.AutoMerge},
		{"update-branch", EditUpdateBranch, &ero.Merge.UpdateBranch},
		{"rebase-update", EditRebaseUpdate, &ero.Merge.RebaseUpdate},

		{"time-tracking", EditTimeTracking, &ero.IssueTracker.TimeTracking},
		{"time-tracking-contributors-only", EditTimeTrackingContributorsOnly, &ero.IssueTracker.TimeTrackingContributorsOnly},
		{"dependencies", EditDependencies, &ero.IssueTracker.Dependencies},
		{"wiki-editable-by-all", EditWikiEditableByAll, &ero.Wiki.EditableByAll},

		{"actions-approve-prs", EditActionsApprovePRs, &ero.Actions.ApprovePullRequests},
	}
	for _, f := range boolFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	if changed("public") {
		ero.Visibility = gol.Ptr(gol.VisibilityPrivate)
		if EditRepoPublic {
			ero.Visibility = gol.Ptr(gol.VisibilityPublic)
		}
	}

	if changed("default-merge") {
		// manually-merged marks a pull request, it is no merge method a repository can propose
		m := gol.MergeMethod(EditDefaultMerge)
		if !m.Valid() || m == gol.MergeMethodManuallyMerged {
			return ero, badValue("default-merge", EditDefaultMerge, gol.MergeMethodMerge, gol.MergeMethodRebase,
				gol.MergeMethodRebaseMerge, gol.MergeMethodSquash, gol.MergeMethodFastForwardOnly)
		}
		ero.Merge.DefaultMethod = &m
	}
	if changed("actions-allowed") {
		a := gol.ActionsAllowed(EditActionsAllowed)
		if !slices.Contains([]gol.ActionsAllowed{gol.ActionsAllowedAll, gol.ActionsAllowedLocalOnly, gol.ActionsAllowedSelected}, a) {
			return ero, badValue("actions-allowed", EditActionsAllowed, gol.ActionsAllowedAll, gol.ActionsAllowedLocalOnly, gol.ActionsAllowedSelected)
		}
		ero.Actions.Allowed = &a
	}
	if changed("actions-token") {
		p := gol.ActionsTokenPermission(EditActionsToken)
		if p != gol.ActionsTokenRead && p != gol.ActionsTokenWrite {
			return ero, badValue("actions-token", EditActionsToken, gol.ActionsTokenRead, gol.ActionsTokenWrite)
		}
		ero.Actions.TokenPermission = &p
	}

	// The tracker's format, style and pattern are only sent with its URL: alone, the library would read
	// the zero URL as a return to the built-in tracker
	if !changed("external-tracker") {
		for _, f := range []string{"tracker-format", "tracker-style", "tracker-pattern"} {
			if changed(f) {
				return ero, &cerr.CustomError{Title: "Incomplete external tracker",
					Message: "--" + f + " needs --external-tracker URL"}
			}
		}
		return ero, nil
	}
	ext := gol.ExternalTracker{URL: EditExternalTracker, Format: EditTrackerFormat,
		Style: gol.ExternalIssueStyle(EditTrackerStyle), Pattern: EditTrackerPattern}
	if ext.Style != "" && !slices.Contains([]gol.ExternalIssueStyle{gol.ExternalIssueNumeric, gol.ExternalIssueAlphanumeric, gol.ExternalIssueRegexp}, ext.Style) {
		return ero, badValue("tracker-style", EditTrackerStyle, gol.ExternalIssueNumeric, gol.ExternalIssueAlphanumeric, gol.ExternalIssueRegexp)
	}
	if ext.Pattern != "" && ext.Style != gol.ExternalIssueRegexp {
		return ero, &cerr.CustomError{Title: "Invalid external tracker", Message: "--tracker-pattern needs --tracker-style regexp"}
	}
	ero.IssueTracker.External = &ext

	return ero, nil
}

// badValue refuses a flag's value, listing the accepted ones
func badValue[T ~string](flag, val string, accepted ...T) *cerr.CustomError {
	list := ""
	for i, a := range accepted {
		if i > 0 {
			list += ", "
		}
		list += string(a)
	}
	return &cerr.CustomError{Title: "Invalid value", Message: fmt.Sprintf("--%s %q: expected one of %s", flag, val, list)}
}
