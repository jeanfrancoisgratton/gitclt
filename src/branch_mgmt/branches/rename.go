// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/rename.go
// Original timestamp : 2026.10.28 09:12:30

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// RenameBranch renames a branch of owner/repo on the server; the local working copy is never touched
// The open pull requests and the protection rules follow the branch
func RenameBranch(repo, branch, newName string) *cerr.CustomError {
	if !RemoteOnly {
		return errServerOnly("rename", "git branch -m")
	}
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	if _, e := c.Branches().Rename(ctx, owner, repo, branch, newName); e != nil {
		return forge.Error("Unable to rename the branch "+branch+" on the server", e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Renamed server branch " + hftx.Brown(branch) + " to " + hftx.Brown(newName)))
	}
	return nil
}
