// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/create.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// CreateToken creates an access token for the connection's user, and shows its secret: the only time it can be seen (Forgejo only)
func CreateToken(name string) *cerr.CustomError {
	ctx := context.Background()

	c, ce := connectWithPassword(ctx)
	if ce != nil {
		return ce
	}
	tk, ce := tokensOf(c)
	if ce != nil {
		return ce
	}

	tok, e := tk.Create(ctx, gol.CreateTokenOptions{Name: name, Scopes: CreateTokenScopes, Repositories: CreateTokenRepos})
	if e != nil {
		return forge.Error("Unable to create the token", e)
	}

	if JSONoutput {
		return shared.PrintJSON(createdToken{AccessToken: *tok, Secret: tok.Secret}, "token")
	}
	// Only the secret, so that it can be captured by a script
	if shared.QuietOutput {
		fmt.Println(tok.Secret)
		return nil
	}

	fmt.Println(hftx.EnabledSign("Created token " + hftx.Brown(name)))
	printToken(tok, tok.Secret)
	fmt.Println(hftx.NoteSign("Copy the secret now: it cannot be shown again"))
	return nil
}
