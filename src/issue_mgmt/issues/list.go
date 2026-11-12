// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/list.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListIssues lists the issues of owner/repo; pull requests are never included
func ListIssues(repo string, displayoutput bool) ([]gol.Issue, *cerr.CustomError) {
	ctx := context.Background()
	state := gol.IssueState(ListState)
	if state != gol.IssueStateOpen && state != gol.IssueStateClosed && state != gol.IssueStateAll {
		return nil, &cerr.CustomError{Title: "Invalid state", Message: ListState + " is not one of open, closed, all"}
	}

	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Issue], error) {
		return c.Issues().List(ctx, owner, repo, gol.ListIssuesOptions{ListOptions: o, State: state, Labels: ListLabels})
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the issues of %s/%s", owner, repo), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "issue list")
	}
	if shared.QuietOutput {
		for _, i := range list {
			fmt.Println(i.Number)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no %sissue\n", owner, repo, strings.TrimSuffix(string(state)+" ", "all "))
		return list, nil
	}
	shared.Page(renderIssueTable(list), shared.TableHeader)
	return list, nil
}

// GetIssue describes one issue of owner/repo; --comments adds what was said on it
func GetIssue(repo, number string, displayoutput bool) (*Issue, *cerr.CustomError) {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return nil, ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	issue, ce := getRealIssue(ctx, c, owner, repo, nums[0])
	if ce != nil {
		return nil, ce
	}
	out := &Issue{Issue: *issue}
	if WithComments {
		if out.Discussion, ce = fetchComments(ctx, c, owner, repo, nums[0]); ce != nil {
			return nil, ce
		}
	}

	if !displayoutput {
		return out, nil
	}
	if JSONoutput {
		return out, shared.PrintJSON(out, "issue")
	}
	// The issue's line stays on top
	shared.Page(renderIssue(out), 1)
	return out, nil
}
