// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/list.go
// Original timestamp : 2026.10.30 14:41:08

package history

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// List shows the commits of owner/repo reachable from ListRef, newest first, like git log does; only pushed commits
// are seen. At most ListLimit commits are fetched, so a long history does not mean paging through all of it
func List(repo string, displayoutput bool) ([]gol.Commit, *cerr.CustomError) {
	ctx := context.Background()

	opts := gol.ListCommitsOptions{Ref: ListRef, Path: ListPath}
	var ce *cerr.CustomError
	if ListLimit < 0 {
		return nil, &cerr.CustomError{Title: "Invalid limit", Message: fmt.Sprintf("%d is not a commit count", ListLimit)}
	}
	if opts.Since, ce = parseDate(ListSince, false); ce != nil {
		return nil, ce
	}
	if opts.Until, ce = parseDate(ListUntil, true); ce != nil {
		return nil, ce
	}

	c, owner, h, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	commits := []gol.Commit{}
	opts.PerPage = shared.ListPageSize
	if ListLimit > 0 {
		opts.PerPage = min(ListLimit, shared.ListPageSize)
	}
	for commit, e := range gol.All(ctx, opts.ListOptions, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Commit], error) {
		opts.ListOptions = o
		return h.Commits().List(ctx, owner, repo, opts)
	}) {
		if e != nil {
			if gol.IsNotFound(e) {
				return nil, &cerr.CustomError{Title: "Unknown reference", Code: http.StatusNotFound,
					Message: fmt.Sprintf("%s/%s has no branch, tag or commit %s", owner, repo, ListRef)}
			}
			return nil, forge.Error(fmt.Sprintf("Unable to list the commits of %s/%s", owner, repo), e)
		}
		commits = append(commits, commit)
		if ListLimit > 0 && len(commits) == ListLimit {
			break
		}
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return commits, nil
	}
	if JSONoutput {
		return commits, shared.PrintJSON(commits, "commit list")
	}
	if len(commits) == 0 {
		if !shared.QuietOutput {
			fmt.Printf("No commit of %s/%s matches\n", owner, repo)
		}
		return commits, nil
	}
	renderCommitTable(commits)
	return commits, nil
}
