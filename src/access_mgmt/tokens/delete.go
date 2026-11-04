// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/delete.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

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

// DeleteToken deletes an access token of the connection's user, found by name, or by ID if --id was passed (Forgejo only)
func DeleteToken(name string) *cerr.CustomError {
	ctx := context.Background()

	c, ce := connectWithPassword(ctx)
	if ce != nil {
		return ce
	}
	tk, ce := tokensOf(c)
	if ce != nil {
		return ce
	}
	list, ce := fetchTokens(ctx, tk)
	if ce != nil {
		return ce
	}
	tok, ce := findToken(list, name)
	if ce != nil {
		return ce
	}
	inUse := usedByConnection(c, tok)

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		prompt := "Delete token " + hftx.Brown(tok.Name)
		if inUse {
			prompt += ", which connection " + c.Connection.Name + " authenticates with"
		}
		answer := hf.GetStringValFromPrompt(hftx.WarningSign(prompt + " ? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, token " + hftx.Brown(tok.Name) + " was not deleted")
			return nil
		}
	}

	if e := tk.Delete(ctx, tok.ID); e != nil {
		return forge.Error("Unable to delete the token", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted token " + hftx.Brown(tok.Name)))
		if inUse {
			fmt.Println(hftx.WarningSign(fmt.Sprintf("Connection %s no longer works; fix it with: gitclt conn edit %s", c.Connection.Name, c.Connection.Name)))
		}
	}
	return nil
}
