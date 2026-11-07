// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/helpers.go
// Original timestamp : 2026.11.06 14:00:00

package issues

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
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// connect opens the client on the selected connection, refuses Bitbucket (its issue tracker was retired), decides the
// repository and its owner (see forge.Client.Target), and makes sure that repository exists, so that a 404 further down
// is about the issue, not the repository
func connect(ctx context.Context, repo string) (*forge.Client, string, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", "", ce
	}
	if !c.Capabilities().Has(gol.CapIssues) {
		return nil, "", "", &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: Bitbucket Cloud retired its issue tracker", c.Connection.Name)}
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return nil, "", "", ce
	}
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return nil, "", "", &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return nil, "", "", forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return c, owner, repo, nil
}

// parseNumbers reads issue numbers, with or without a leading # (#12 or 12), dropping repeated ones
func parseNumbers(args []string) ([]int, *cerr.CustomError) {
	var out []int
	for _, a := range args {
		n, e := strconv.Atoi(strings.TrimPrefix(a, "#"))
		if e != nil || n <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid issue number", Message: a + " is not an issue number"}
		}
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// parseIDs reads comment IDs, dropping repeated ones
func parseIDs(args []string) ([]int64, *cerr.CustomError) {
	var out []int64
	for _, a := range args {
		id, e := strconv.ParseInt(a, 10, 64)
		if e != nil || id <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid comment ID", Message: a + " is not a comment ID"}
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// getRealIssue is getIssue for what must not touch a pull request: they share their numbers with issues, and the server
// takes a pull request's number for an issue's without a word
func getRealIssue(ctx context.Context, c *forge.Client, owner, repo string, number int) (*gol.Issue, *cerr.CustomError) {
	issue, ce := getIssue(ctx, c, owner, repo, number)
	if ce != nil {
		return nil, ce
	}
	if issue.IsPullRequest {
		return nil, &cerr.CustomError{Title: "Not an issue", Message: fmt.Sprintf("#%d of %s/%s is a pull request: see gitclt pr", number, owner, repo)}
	}
	return issue, nil
}

// getIssue fetches one issue, or pull request as they share their numbers; a 404 is reported as such
func getIssue(ctx context.Context, c *forge.Client, owner, repo string, number int) (*gol.Issue, *cerr.CustomError) {
	issue, e := c.Issues().Get(ctx, owner, repo, number)
	if e != nil {
		// The library takes GitHub's 410, for a deleted issue, for a 404
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such issue", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s has no issue #%d", owner, repo, number)}
		}
		return nil, forge.Error(fmt.Sprintf("Unable to get issue #%d", number), e)
	}
	return issue, nil
}

// body is the text of --body, or of the --body-file file; set tells whether either flag was passed
func body(changed func(string) bool) (text string, set bool, ce *cerr.CustomError) {
	switch {
	case changed("body-file"):
		b, e := os.ReadFile(IssueBodyFile)
		if e != nil {
			return "", false, &cerr.CustomError{Title: "Unable to read " + IssueBodyFile, Message: e.Error()}
		}
		return string(b), true, nil
	case changed("body"):
		return IssueBody, true, nil
	}
	return "", false, nil
}

func allMilestones(ctx context.Context, c *forge.Client, owner, repo string) ([]gol.Milestone, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Milestone], error) {
		return c.Milestones().List(ctx, owner, repo, gol.MilestoneStateAll, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the milestones of %s/%s", owner, repo), e)
	}
	return list, nil
}

// findMilestone finds a milestone by its title, then by its ID: a title can look like a number
// Forgejo lets several milestones share a title: the title is then refused, as it could hit the wrong one
func findMilestone(ctx context.Context, c *forge.Client, owner, repo, arg string) (*gol.Milestone, *cerr.CustomError) {
	list, ce := allMilestones(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}
	var found *gol.Milestone
	var ids []string
	for i := range list {
		if list[i].Title == arg {
			found = &list[i]
			ids = append(ids, strconv.FormatInt(list[i].ID, 10))
		}
	}
	if len(ids) > 1 {
		return nil, &cerr.CustomError{Title: "Ambiguous milestone",
			Message: fmt.Sprintf("%d milestones of %s/%s have the title %s; designate one by its ID (%s)", len(ids), owner, repo, arg, strings.Join(ids, ", "))}
	}
	if found != nil {
		return found, nil
	}
	if id, e := strconv.ParseInt(arg, 10, 64); e == nil {
		for i := range list {
			if list[i].ID == id {
				return &list[i], nil
			}
		}
	}
	return nil, &cerr.CustomError{Title: "No such milestone", Code: http.StatusNotFound,
		Message: fmt.Sprintf("%s/%s has no milestone %s", owner, repo, arg)}
}

// parseDue reads a due date: a day means the end of that day, where it is run
func parseDue(s string) (time.Time, *cerr.CustomError) {
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return t, nil
	}
	t, e := time.ParseInLocation(time.DateOnly, s, time.Local)
	if e != nil {
		return time.Time{}, &cerr.CustomError{Title: "Invalid date", Message: s + " is neither YYYY-MM-DD nor an RFC 3339 timestamp"}
	}
	return t.AddDate(0, 0, 1).Add(-time.Second), nil
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}

// unique drops repeated arguments, keeping the first occurrence's order
func unique(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}
