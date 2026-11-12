// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/local.go
// Original timestamp : 2026.11.11 16:00:00

package history

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// localClone is the working copy of the current directory, when it is a clone of owner/repo; nil otherwise,
// as history ls then only shows the server's side
func localClone(owner, repo string) *shared.WorkingCopy {
	cwd, e := os.Getwd()
	if e != nil {
		return nil
	}
	wc, e := shared.ReadWorkingCopy(cwd)
	if e != nil || !wc.MatchesRepo(owner, repo) {
		return nil
	}
	return wc
}

// defaultRef is what history ls lists without --ref: the checked out branch, when the server has it, else the
// default branch (an empty ref). A branch the server does not have yet is said so, as its commits are all local
func defaultRef(ctx context.Context, c *forge.Client, owner, repo string, wc *shared.WorkingCopy) string {
	if wc == nil || wc.Current == "" {
		return ""
	}
	if _, e := c.Branches().Get(ctx, owner, repo, wc.Current); e != nil {
		if gol.IsNotFound(e) && !shared.QuietOutput && !JSONoutput {
			fmt.Println(hftx.WarningSign(fmt.Sprintf("The branch %s is not on the server yet: listing the default branch instead",
				hftx.Brown(wc.Current))))
		}
		return ""
	}
	return wc.Current
}

// unpushedNotice tells when the local branch the list is about differs from the server's: the list only has the
// server's commits, so local ones are missing from it. A ref that is no local branch (a tag, a commit) has no notice
// An empty ref is the default branch
func unpushedNotice(ctx context.Context, c *forge.Client, owner, repo, ref, defaultBranch string, wc *shared.WorkingCopy) string {
	if wc == nil {
		return ""
	}
	if ref == "" {
		ref = defaultBranch
	}
	localSHA, ok := wc.Branches[ref]
	if !ok {
		return ""
	}
	sb, e := c.Branches().Get(ctx, owner, repo, ref)
	if e != nil || sb.SHA == localSHA {
		return ""
	}

	branch := hftx.Brown(ref)
	// The server does not know the local head: it has commits that were never pushed
	cmp, e := c.Branches().Compare(ctx, owner, repo, ref, localSHA)
	switch {
	case gol.IsNotFound(e):
		return fmt.Sprintf("Your local branch %s has commits that are not pushed yet: they are not in this list", branch)
	case e != nil:
		return ""
	case cmp.Ahead > 0:
		return fmt.Sprintf("Your local branch %s has %d commit(s) that the server's does not: they are not in this list", branch, cmp.Ahead)
	case cmp.Behind > 0:
		return fmt.Sprintf("Your local branch %s is %d commit(s) behind the server's", branch, cmp.Behind)
	}
	return ""
}
