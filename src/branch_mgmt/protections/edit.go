// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/edit.go
// Original timestamp : 2026.11.04 22:00:00

package protections

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditProtection only changes the settings whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditProtection(repo, pattern string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()

	edits, ce := ruleEdits(changed)
	if ce != nil {
		return ce
	}
	if len(edits) == 0 {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt branch protection edit --help"}
	}

	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	rule, ce := getRule(ctx, c, owner, repo, pattern)
	if ce != nil {
		return ce
	}

	for _, edit := range edits {
		edit(rule)
	}
	if _, e := c.BranchProtections().Update(ctx, owner, repo, pattern, *rule); e != nil {
		return forge.Error("Unable to edit the protection rule "+pattern, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated protection rule %s on %s/%s", hftx.Brown(pattern), owner, repo)))
	}
	return nil
}
