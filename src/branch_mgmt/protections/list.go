// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/list.go
// Original timestamp : 2026.11.04 22:00:00

package protections

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListProtections lists the branch protection rules of owner/repo
func ListProtections(repo string, displayoutput bool) ([]gol.BranchProtection, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	list, e := c.BranchProtections().List(ctx, owner, repo)
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the protection rules of %s/%s", owner, repo), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "protection rule list")
	}
	if shared.QuietOutput {
		for i := range list {
			fmt.Println(list[i].Pattern)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no protection rule\n", owner, repo)
		return list, nil
	}
	renderRuleTable(list)
	return list, nil
}

// GetProtection shows the branch protection rule of owner/repo addressed by pattern
func GetProtection(repo, pattern string, displayoutput bool) (*gol.BranchProtection, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}
	rule, ce := getRule(ctx, c, owner, repo, pattern)
	if ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return rule, nil
	}
	if JSONoutput {
		return rule, shared.PrintJSON(rule, "protection rule")
	}
	renderRule(rule, c.Backend())
	return rule, nil
}
