// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/create.go
// Original timestamp : 2026.11.04 22:00:00

package protections

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

// CreateProtection adds a branch protection rule to owner/repo; changed is cmd.Flags().Changed
// Without any flag, the rule still blocks the deletion of, and the force pushes to, the branches it covers
func CreateProtection(repo, pattern string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()

	edits, ce := ruleEdits(changed)
	if ce != nil {
		return ce
	}
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}

	// Forgejo answers an existing rule with a vague refusal, GitHub could end up with two rulesets on the same branches
	if _, e := c.BranchProtections().Get(ctx, owner, repo, pattern); e == nil {
		return &cerr.CustomError{Title: "Protection rule already exists", Code: http.StatusConflict,
			Message: fmt.Sprintf("%s/%s already has a protection rule %s; change it with: gitclt branch protection edit", owner, repo, pattern)}
	} else if !gol.IsNotFound(e) {
		return forge.Error("Unable to get the protection rule "+pattern, e)
	}

	rule := gol.BranchProtection{Pattern: pattern}
	for _, edit := range edits {
		edit(&rule)
	}
	if _, e := c.BranchProtections().Create(ctx, owner, repo, rule); e != nil {
		return forge.Error("Unable to create the protection rule "+pattern, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created protection rule %s on %s/%s", hftx.Brown(pattern), owner, repo)))
	}
	return nil
}
