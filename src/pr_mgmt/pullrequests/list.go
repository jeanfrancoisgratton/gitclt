// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/list.go
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
)

// ListPRs lists the pull requests of owner/repo, in the server's order (newest first): open, closed (merged
// included) or all
func ListPRs(repo string, displayoutput bool) ([]gol.PullRequest, *cerr.CustomError) {
	ctx := context.Background()

	state := gol.PRState(strings.ToLower(strings.TrimSpace(ListState)))
	if state != gol.PRStateOpen && state != gol.PRStateClosed && state != gol.PRStateAll {
		return nil, &cerr.CustomError{Title: "Invalid state", Message: ListState + " is not one of open, closed, all"}
	}

	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if _, ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	prs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.PullRequest], error) {
		return c.PullRequests().List(ctx, owner, repo, gol.ListPROptions{ListOptions: o, State: state})
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the pull requests of %s/%s", owner, repo), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return prs, nil
	}
	if JSONoutput {
		return prs, shared.PrintJSON(prs, "pull request list")
	}
	if len(prs) == 0 {
		if !shared.QuietOutput {
			fmt.Printf("%s/%s has no %s pull request\n", owner, repo, strings.TrimSuffix(string(state), "all"))
		}
		return prs, nil
	}
	renderPRTable(prs)
	return prs, nil
}

// GetPR describes one pull request of owner/repo, with its reviews
func GetPR(repo, number string, displayoutput bool) (*PullRequest, *cerr.CustomError) {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return nil, ce
	}
	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if _, ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}
	pr, ce := getPR(ctx, c, owner, repo, nums[0])
	if ce != nil {
		return nil, ce
	}

	reviews, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Review], error) {
		return c.PullRequests().ListReviews(ctx, owner, repo, nums[0], o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the reviews of pull request #%d", nums[0]), e)
	}
	out := &PullRequest{PullRequest: *pr, Reviews: reviews}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return out, nil
	}
	if JSONoutput {
		return out, shared.PrintJSON(out, "pull request info")
	}
	renderPR(*out)
	return out, nil
}
