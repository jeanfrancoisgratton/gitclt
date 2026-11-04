// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/list.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListTokens lists the access tokens of the connection's user (Forgejo only)
func ListTokens(displayOutput bool) ([]gol.AccessToken, *cerr.CustomError) {
	ctx := context.Background()

	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, ce
	}
	tk, ce := tokensOf(c)
	if ce != nil {
		return nil, ce
	}
	list, ce := fetchTokens(ctx, tk)
	if ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return list, nil
	}

	if JSONoutput {
		return list, shared.PrintJSON(list, "token list")
	}
	if shared.QuietOutput {
		for _, tok := range list {
			fmt.Println(tok.Name)
		}
		return list, nil
	}

	renderTokenTable(list)
	return list, nil
}
