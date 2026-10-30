// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/review.go
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

// ReviewPR submits a review of a pull request of owner/repo, as the connection's user: an approval, a change
// request or a comment (exactly one); the last two need a text (-m). Authors cannot approve or request changes
// on their own pull requests (Forgejo, GitHub)
func ReviewPR(repo, number string) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}

	var event gol.ReviewEvent
	n := 0
	for _, v := range []struct {
		set bool
		ev  gol.ReviewEvent
	}{{ReviewApprove, gol.ReviewEventApprove}, {ReviewRequestChanges, gol.ReviewEventRequestChanges}, {ReviewComment, gol.ReviewEventComment}} {
		if v.set {
			event, n = v.ev, n+1
		}
	}
	if n != 1 {
		return &cerr.CustomError{Title: "Invalid flags", Message: "pass exactly one of --approve, --request-changes, --comment"}
	}
	if event != gol.ReviewEventApprove && strings.TrimSpace(ReviewBody) == "" {
		return &cerr.CustomError{Title: "Missing text", Message: "--request-changes and --comment need a text (-m)"}
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
	if pr.State != gol.PRStateOpen {
		return &cerr.CustomError{Title: "Pull request closed", Message: fmt.Sprintf("pull request #%d is %s", pr.Number, status(*pr))}
	}

	r, e := c.PullRequests().CreateReview(ctx, owner, repo, pr.Number, gol.CreateReviewOptions{Event: event, Body: ReviewBody})
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to review pull request #%d", pr.Number), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Reviewed pull request #%d: %s", pr.Number, strings.ReplaceAll(string(r.State), "_", " "))))
	}
	return nil
}

// SetReviewers requests the review of users on a pull request of owner/repo, or with --remove, withdraws the requests
func SetReviewers(repo, number string, users []string) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if _, ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	if _, ce = getPR(ctx, c, owner, repo, nums[0]); ce != nil {
		return ce
	}

	call, verb := c.PullRequests().RequestReviewers, "Requested the review of"
	if RemoveReviewers {
		call, verb = c.PullRequests().RemoveReviewers, "Withdrew the review request of"
	}
	if e := call(ctx, owner, repo, nums[0], users, nil); e != nil {
		return forge.Error(fmt.Sprintf("Unable to change the reviewers of pull request #%d", nums[0]), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("%s %s on pull request #%d", verb, strings.Join(users, ", "), nums[0])))
	}
	return nil
}
