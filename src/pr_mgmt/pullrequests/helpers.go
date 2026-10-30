// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/helpers.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// connect opens the client on the selected connection, and decides the repositories' owner (-o, or the connection's default)
func connect(ctx context.Context) (*forge.Client, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", ce
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return nil, "", ce
	}
	return c, owner, nil
}

// checkRepo makes sure owner/repo exists, so that a 404 further down is about the pull request, not the repository
func checkRepo(ctx context.Context, c *forge.Client, owner, repo string) (*gol.Repo, *cerr.CustomError) {
	r, e := c.Repos().Get(ctx, owner, repo)
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return nil, forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return r, nil
}

// parseNumbers reads pull request numbers, with or without a leading # (#12 or 12), dropping repeated ones
func parseNumbers(args []string) ([]int, *cerr.CustomError) {
	var out []int
	for _, a := range args {
		n, e := strconv.Atoi(strings.TrimPrefix(a, "#"))
		if e != nil || n <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid pull request number", Message: a + " is not a pull request number"}
		}
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// getPR fetches one pull request, a 404 being reported as such
func getPR(ctx context.Context, c *forge.Client, owner, repo string, number int) (*gol.PullRequest, *cerr.CustomError) {
	pr, e := c.PullRequests().Get(ctx, owner, repo, number)
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such pull request", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s has no pull request #%d", owner, repo, number)}
		}
		return nil, forge.Error(fmt.Sprintf("Unable to get pull request #%d", number), e)
	}
	return pr, nil
}

// status words where a pull request stands: merged pull requests are closed ones, for the forges
func status(pr gol.PullRequest) string {
	switch {
	case pr.Merged:
		return "merged"
	case pr.State == gol.PRStateClosed:
		return "closed"
	case pr.Draft:
		return "draft"
	}
	return "open"
}

// body is the text of --body, or of the --body-file file; set tells whether either flag was passed
func body(changed func(string) bool) (text string, set bool, ce *cerr.CustomError) {
	switch {
	case changed("body-file"):
		b, e := os.ReadFile(PRBodyFile)
		if e != nil {
			return "", false, &cerr.CustomError{Title: "Unable to read " + PRBodyFile, Message: e.Error()}
		}
		return string(b), true, nil
	case changed("body"):
		return PRBody, true, nil
	}
	return "", false, nil
}
