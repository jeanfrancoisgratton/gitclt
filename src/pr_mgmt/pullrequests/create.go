// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/create.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// CreatePR opens a pull request of owner/repo, from head into base (the default branch when empty)
// The title defaults to the head branch's name; -r requests reviews once the pull request is open
// changed is cmd.Flags().Changed, for --body / --body-file
func CreatePR(repo, head, base string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	text, _, ce := body(changed)
	if ce != nil {
		return ce
	}

	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	r, ce := checkRepo(ctx, c, owner, repo)
	if ce != nil {
		return ce
	}
	if base == "" {
		base = r.DefaultBranch
	}
	if head == base {
		return &cerr.CustomError{Title: "Same branches", Message: fmt.Sprintf("%s cannot be merged into itself", head)}
	}
	// Both branches are checked first: the forges' own errors for a missing one are vague (422, 404 on the repo...)
	// A fork's head (user:branch) is left to the server
	for _, b := range []string{head, base} {
		if strings.Contains(b, ":") {
			continue
		}
		if _, e := c.Branches().Get(ctx, owner, repo, b); e != nil {
			if gol.IsNotFound(e) {
				return &cerr.CustomError{Title: "No such branch", Code: http.StatusNotFound,
					Message: fmt.Sprintf("%s/%s has no branch %s; push it first", owner, repo, b)}
			}
			return forge.Error("Unable to get the branch "+b, e)
		}
	}

	title := PRTitle
	if title == "" {
		title = head
	}
	pr, e := c.PullRequests().Create(ctx, owner, repo, gol.CreatePROptions{Title: title, Body: text, Head: head, Base: base, Draft: PRDraft})
	if e != nil {
		// Forgejo may not honour the draft (custom work-in-progress prefixes): the pull request is open all the same
		if pr == nil || !errors.Is(e, gol.ErrNotSupported) {
			return forge.Error("Unable to create the pull request", e)
		}
		fmt.Println(hftx.WarningSign(" The pull request is not a draft: " + e.Error()))
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created pull request #%d (%s): %s", pr.Number, branches(*pr), pr.HTMLURL)))
	}

	if len(PRReviewers) > 0 {
		if e = c.PullRequests().RequestReviewers(ctx, owner, repo, pr.Number, PRReviewers, nil); e != nil {
			return forge.Error(fmt.Sprintf("Pull request #%d is open, but its reviews could not be requested", pr.Number), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Requested the review of " + strings.Join(PRReviewers, ", ")))
		}
	}
	return nil
}
