// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/merge.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// MergePR merges a pull request of owner/repo with --method, provided its head did not move since it was checked
// --auto schedules the merge for when the checks pass instead; --cancel-auto cancels such a schedule
// Drafts, closed pull requests and conflicting ones are refused before asking the server
func MergePR(repo, number string) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}
	method := gol.MergeMethod(strings.ToLower(strings.TrimSpace(MergeMethod)))
	if !method.Valid() {
		return &cerr.CustomError{Title: "Invalid merge method",
			Message: MergeMethod + " is not one of merge, squash, rebase, rebase-merge, fast-forward-only, manually-merged"}
	}

	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if _, ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	pr, ce := getPR(ctx, c, owner, repo, nums[0])
	if ce != nil {
		return ce
	}

	if CancelAutoMerge {
		if e := c.PullRequests().CancelAutoMerge(ctx, owner, repo, pr.Number); e != nil {
			return forge.Error(fmt.Sprintf("Unable to cancel the automatic merge of pull request #%d", pr.Number), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Cancelled the automatic merge of pull request #%d", pr.Number)))
		}
		return nil
	}

	switch {
	case pr.Merged:
		return &cerr.CustomError{Title: "Already merged", Message: fmt.Sprintf("pull request #%d is already merged", pr.Number)}
	case pr.State != gol.PRStateOpen:
		return &cerr.CustomError{Title: "Pull request closed", Message: fmt.Sprintf("pull request #%d is closed; reopen it first", pr.Number)}
	case pr.Draft:
		return &cerr.CustomError{Title: "Draft pull request",
			Message: fmt.Sprintf("pull request #%d is a draft; mark it ready first (gitclt pr edit -o %s --draft=false %s %d)", pr.Number, owner, repo, pr.Number)}
	case pr.Mergeable != nil && !*pr.Mergeable && !AutoMerge:
		return &cerr.CustomError{Title: "Conflicts", Message: fmt.Sprintf("pull request #%d has conflicts with %s; resolve them first", pr.Number, pr.Base.Ref)}
	}

	opts := gol.MergeOptions{Method: method, Title: MergeTitle, Message: MergeMessage, HeadSHA: pr.Head.SHA,
		DeleteBranch: DeleteBranch, WhenChecksSucceed: AutoMerge}
	if e := c.PullRequests().Merge(ctx, owner, repo, pr.Number, opts); e != nil {
		return forge.Error(fmt.Sprintf("Unable to merge pull request #%d", pr.Number), e)
	}
	if shared.QuietOutput {
		return nil
	}
	switch {
	case AutoMerge:
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Pull request #%d will be merged (%s) once its checks pass", pr.Number, method)))
	default:
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Merged pull request #%d (%s) into %s", pr.Number, method, pr.Base.Ref)))
		if DeleteBranch {
			fmt.Println(hftx.EnabledSign("Deleted branch " + hftx.Brown(pr.Head.Ref)))
		}
	}
	return nil
}
