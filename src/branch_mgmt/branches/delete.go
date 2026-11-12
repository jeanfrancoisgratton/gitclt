// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/delete.go
// Original timestamp : 2026.10.27 19:22:41

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// errServerOnly refuses rm and rename without -r: they used to act on the local working copy by default, and a bare
// command must not now delete or rename on the server by mistake
func errServerOnly(action, gitCommand string) *cerr.CustomError {
	return &cerr.CustomError{Title: "Server branches only",
		Message: "gitclt " + action + "s branches on the server only, and needs -r (--remote) to say so; " +
			"local branches are git's job (" + gitCommand + ")"}
}

// DeleteBranch deletes branches of owner/repo on the server; the local working copy is never touched
// The first failure stops the run; the branches before it stay deleted
func DeleteBranch(repo string, branches []string) *cerr.CustomError {
	if !RemoteOnly {
		return errServerOnly("delete", "git branch -d")
	}
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign(fmt.Sprintf("Delete branch(es) %s on the server, %s/%s ? [y/N]: ",
			strings.Join(branches, ", "), owner, repo)))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, no branch is deleted")
			return nil
		}
	}

	for _, branch := range branches {
		if e := c.Branches().Delete(ctx, owner, repo, branch); e != nil {
			return forge.Error("Unable to delete the branch "+branch+" on the server", e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted server branch " + hftx.Brown(branch)))
		}
	}
	return nil
}
