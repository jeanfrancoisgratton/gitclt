// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/compare.go
// Original timestamp : 2026.10.28 09:40:05

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// CompareBranches counts, on the server, the commits between two refs of OwnerOrg/repo (branches, tags or commits):
// how many commits head has that base lacks (ahead), and the reverse (behind)
func CompareBranches(repo, base, head string, displayoutput bool) (*gol.Comparison, *cerr.CustomError) {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if OwnerOrg == "" {
		return nil, &cerr.CustomError{Title: "Missing owner", Message: "comparing branches needs the owner (-o OWNER)"}
	}
	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}

	cmp, e := c.Branches().Compare(ctx, OwnerOrg, repo, base, head)
	if gol.IsNotFound(e) {
		return nil, &cerr.CustomError{Title: "Unknown branch", Code: 404,
			Message: fmt.Sprintf("%s or %s is not in %s/%s", base, head, OwnerOrg, repo)}
	}
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to compare %s with %s", head, base), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return cmp, nil
	}
	if JSONoutput {
		return cmp, shared.PrintJSON(cmp, "branch comparison")
	}
	renderComparison(cmp)
	return cmp, nil
}
