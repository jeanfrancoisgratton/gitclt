// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/get.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import (
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// GetToken shows one access token of the connection's user, found by name, or by ID if --id was passed (Forgejo only)
func GetToken(name string, displayOutput bool) (*gol.AccessToken, *cerr.CustomError) {
	list, ce := ListTokens(false)
	if ce != nil {
		return nil, ce
	}
	tok, ce := findToken(list, name)
	if ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return tok, nil
	}

	if JSONoutput {
		return tok, shared.PrintJSON(tok, "token")
	}

	printToken(tok, "")
	return tok, nil
}
