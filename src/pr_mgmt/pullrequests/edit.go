// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/edit.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditPR only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditPR(repo, number string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}

	var epo gol.EditPROptions
	if changed("title") {
		epo.Title = gol.Ptr(PRTitle)
	}
	text, set, ce := body(changed)
	if ce != nil {
		return ce
	}
	if set {
		epo.Body = gol.Ptr(text)
	}
	if changed("base") {
		epo.Base = gol.Ptr(PRBase)
	}
	if changed("draft") {
		epo.Draft = gol.Ptr(PRDraft)
	}
	if epo == (gol.EditPROptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt pr edit --help"}
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

	pr, e := c.PullRequests().Edit(ctx, owner, repo, nums[0], epo)
	if e != nil {
		// The draft change was not honoured (Forgejo with custom prefixes): the other fields were edited all the same
		if pr == nil || !errors.Is(e, gol.ErrNotSupported) {
			return forge.Error(fmt.Sprintf("Unable to edit pull request #%d", nums[0]), e)
		}
		fmt.Println(hftx.WarningSign(" The draft status is unchanged: " + e.Error()))
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated pull request #%d (%s)", pr.Number, status(*pr))))
	}
	return nil
}

// ClosePRs closes (or, with reopen, reopens) pull requests of owner/repo
// Every number is checked first; those already in the wanted state are skipped, and a merged pull request cannot
// be reopened. The run stops at the first failure; the pull requests changed before it stay changed
// Bitbucket cannot reopen: a declined pull request is final there
func ClosePRs(repo string, numbers []string, reopen bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers(numbers)
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

	want, verb := gol.PRStateClosed, "Closed"
	if reopen {
		want, verb = gol.PRStateOpen, "Reopened"
	}
	var todo []int
	for _, n := range nums {
		pr, ce := getPR(ctx, c, owner, repo, n)
		if ce != nil {
			return ce
		}
		switch {
		case pr.Merged:
			if reopen {
				return &cerr.CustomError{Title: "Pull request merged", Message: fmt.Sprintf("#%d is merged: it cannot be reopened", n)}
			}
			if !shared.QuietOutput {
				fmt.Printf("Pull request #%d is already merged\n", n)
			}
		case pr.State == want:
			if !shared.QuietOutput {
				fmt.Printf("Pull request #%d is already %s\n", n, want)
			}
		default:
			todo = append(todo, n)
		}
	}

	for _, n := range todo {
		if _, e := c.PullRequests().Edit(ctx, owner, repo, n, gol.EditPROptions{State: gol.Ptr(want)}); e != nil {
			return forge.Error(fmt.Sprintf("Unable to change pull request #%d", n), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("%s pull request #%d", verb, n)))
		}
	}
	return nil
}
