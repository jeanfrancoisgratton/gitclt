// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/delete.go
// Original timestamp : 2026.11.04 22:00:00

package protections

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteProtections removes branch protection rules from owner/repo
// Every rule is checked before asking; the first failure stops the run, and the rules deleted before it stay deleted
func DeleteProtections(repo string, patterns []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	patterns = unique(patterns)

	var missing []string
	for _, pattern := range patterns {
		if _, e := c.BranchProtections().Get(ctx, owner, repo, pattern); e != nil {
			if !gol.IsNotFound(e) {
				return forge.Error("Unable to get the protection rule "+pattern, e)
			}
			missing = append(missing, pattern)
		}
	}
	if len(missing) > 0 {
		return noSuchRule(owner, repo, strings.Join(missing, ", "))
	}

	if !confirm(hftx.WarningSign(fmt.Sprintf("Delete protection rule(s) %s of %s/%s ? [y/N]: ", strings.Join(patterns, ", "), owner, repo))) {
		fmt.Println("Aborted, no protection rule is deleted")
		return nil
	}

	for _, pattern := range patterns {
		if e := c.BranchProtections().Delete(ctx, owner, repo, pattern); e != nil {
			return forge.Error("Unable to delete the protection rule "+pattern, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted protection rule " + hftx.Brown(pattern)))
		}
	}
	return nil
}
