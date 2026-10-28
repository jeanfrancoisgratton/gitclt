// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/get.go
// Original timestamp : 2026.10.27 21:50:20

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Get describes one branch of OwnerOrg/repo, as seen on the server and, when the current directory is a working copy
// of the repo, locally as well; a branch that exists on only one side is still shown
func GetBranch(repo, branch string, displayoutput bool) (*Branch, *cerr.CustomError) {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if OwnerOrg == "" {
		return nil, &cerr.CustomError{Title: "Missing owner", Message: "the branch info needs the owner (-o OWNER)"}
	}
	b := &Branch{Name: branch}

	// Not being in a working copy of the repo only drops the local side
	lr, lce := localBranches(repo)
	withLocal := lce == nil
	if withLocal {
		b.LocalSHA, b.Current = lr.branches[branch], branch == lr.current
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	sb, e := c.Branches().Get(ctx, OwnerOrg, repo, branch)
	switch {
	case gol.IsNotFound(e):
		if b.LocalSHA == "" {
			return nil, &cerr.CustomError{Title: "No such branch",
				Message: fmt.Sprintf("%s is neither on the server nor in the local working copy", branch)}
		}
	case e != nil:
		return nil, forge.Error("Unable to get the branch "+branch, e)
	default:
		b.ServerSHA, b.IsDefault, b.Protected, b.LastCommit = sb.SHA, sb.IsDefault, sb.Protected, sb.LastCommit
	}

	if ce = compareSides(ctx, c, repo, b); ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return b, nil
	}
	if JSONoutput {
		return b, shared.PrintJSON(b, "branch info")
	}
	var note string
	if !withLocal {
		note = lce.Message
	}
	renderBranch(*b, note)
	return b, nil
}
