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
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// List shows the commits of owner/repo reachable from ListRef, newest first, like git log does; only pushed commits
// are seen, and a notice says when the local branch has others. Without ListRef, from a clone of the repository, the
// checked out branch is listed (when the server has it), else the default branch.
// At most ListLimit commits are fetched, so a long history does not mean paging through all of it
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

	c, owner, repo, h, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	r, ce := checkRepo(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}
	// From a clone of this repository, the checked out branch is listed by default, and local commits missing from the
	// list are pointed out; elsewhere, only the server is looked at
	wc := localClone(owner, repo)
	if opts.Ref == "" {
		opts.Ref = defaultRef(ctx, c, owner, repo, wc)
	}

	commits := []gol.Commit{}
	opts.PerPage = shared.ListPageSize
	if ListLimit > 0 {
		opts.PerPage = min(ListLimit, shared.ListPageSize)
	}
	for commit, e := range gol.All(ctx, opts.ListOptions, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Commit], error) {
		opts.ListOptions = o
		return h.List(ctx, owner, repo, opts)
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
	} else {
		shared.Page(renderCommitTable(commits), shared.TableHeader)
	}
	if notice := unpushedNotice(ctx, c, owner, repo, opts.Ref, r.DefaultBranch, wc); notice != "" && !shared.QuietOutput {
		fmt.Println(hftx.WarningSign(notice))
	}
	return commits, nil
}
