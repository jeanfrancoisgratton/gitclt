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

func DeleteBranch(repo string, branches []string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Delete branch(es) ? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, no branch is deleted")
			return nil
		}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}

	for _, branch := range branches {
		if e := c.Branches().Delete(ctx, OwnerOrg, repo, branch); e != nil {
			return forge.Error("Unable to delete the branch "+branch, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted branch " + hftx.Brown(branch)))
		}
	}
	return nil
}
