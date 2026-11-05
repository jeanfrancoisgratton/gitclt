// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/get.go
// Original timestamp : 2026.10.30 15:10:00

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

// Get describes one commit of owner/repo, with its changed files and line counts, like git show --stat does
// ref is a commit SHA (full or abbreviated), a branch or a tag; the last two stand for their head commit
func Get(repo, ref string, displayoutput bool) (*gol.Commit, *cerr.CustomError) {
	ctx := context.Background()

	c, owner, repo, h, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	commit, e := h.Commits().Get(ctx, owner, repo, ref)
	if gol.IsNotFound(e) {
		return nil, &cerr.CustomError{Title: "No such commit", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no commit, branch or tag %s", owner, repo, ref)}
	}
	if e != nil {
		return nil, forge.Error("Unable to get the commit "+ref, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return commit, nil
	}
	if JSONoutput {
		return commit, shared.PrintJSON(commit, "commit info")
	}
	renderCommit(*commit)
	return commit, nil
}
