// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/helpers.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import (
	"context"
	"fmt"
	"gitclt/connections"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	"golang.org/x/term"
)

func notSupported(connName string) *cerr.CustomError {
	return &cerr.CustomError{Title: "Unsupported operation",
		Message: fmt.Sprintf("Connection %s: access tokens can only be managed on Forgejo", connName)}
}

// Forgejo refuses token authentication when creating or deleting a token: this opens a client with the connection's
// password, which is asked for if the connection does not store one. The connection itself is left untouched
func connectWithPassword(ctx context.Context) (*forge.Client, *cerr.CustomError) {
	ac, ce := connections.ResolveConnection(shared.ConnectionName)
	if ce != nil {
		return nil, ce
	}
	if ac.ServerType != connections.ServerTypeForgejo {
		return nil, notSupported(ac.Name)
	}
	stored, ce := connections.GetConnection(ac.Name)
	if ce != nil {
		return nil, ce
	}

	username, password := stored.Username, stored.Password
	if username == "" {
		username = strings.TrimSpace(hf.GetStringValFromPrompt("Managing tokens needs a password; enter your username: "))
		if username == "" {
			return nil, &cerr.CustomError{Title: "Unable to authenticate", Message: "A username is needed to manage tokens"}
		}
	}
	if password == "" {
		// GetPassword panics without a terminal, unless in debug mode
		if !shared.DebugMode && !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, &cerr.CustomError{Title: "Unable to authenticate",
				Message: fmt.Sprintf("Managing tokens needs a password, which cannot be asked for without a terminal; store it with: gitclt conn edit %s", ac.Name)}
		}
		password = hf.GetPassword("Enter the password of "+username+" (managing tokens needs it): ", shared.DebugMode)
		if password == "" {
			return nil, &cerr.CustomError{Title: "Unable to authenticate", Message: "A password is needed to manage tokens"}
		}
	}
	return forge.ConnectWithPassword(ctx, ac, username, password)
}

// Returns the access token interface; GitHub and Bitbucket have none
func tokensOf(c *forge.Client) (gol.Tokens, *cerr.CustomError) {
	if !c.Capabilities().Has(gol.CapTokens) {
		return nil, notSupported(c.Connection.Name)
	}
	return c.Account().Tokens(), nil
}

func fetchTokens(ctx context.Context, tk gol.Tokens) ([]gol.AccessToken, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, tk.List)
	if e != nil {
		return nil, forge.Error("Unable to fetch the token list", e)
	}
	return list, nil
}

// Finds the token with this name, or the one with TokenID if --id was passed
// Forgejo has no call to fetch a single token, so the lookup is done on the list
func findToken(list []gol.AccessToken, name string) (*gol.AccessToken, *cerr.CustomError) {
	what := fmt.Sprintf("name %s", name)
	if TokenID != 0 {
		what = fmt.Sprintf("ID %d", TokenID)
	}
	for i := range list {
		if (TokenID != 0 && list[i].ID == TokenID) || (TokenID == 0 && list[i].Name == name) {
			return &list[i], nil
		}
	}
	return nil, &cerr.CustomError{Title: "Unable to find the token", Message: "There is no token with " + what, Code: http.StatusNotFound}
}

// Forgejo only reports the last eight characters of a token: this is as far as we can compare it to the connection's own
func usedByConnection(c *forge.Client, tok *gol.AccessToken) bool {
	return tok.LastEight != "" && strings.HasSuffix(c.Connection.Token, tok.LastEight)
}
