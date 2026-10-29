// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/owner.go
// Original timestamp : 2026.10.29 10:30:00

package forge

import (
	"context"
	"fmt"
	"gitclt/connections"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Decides who owns the repositories a command works on, in this order:
// -o (the flag, when given), the connection's owner, the connection's username, then whom the server says the credentials
// belong to; an e-mail is never an owner, so such a username (ex: Bitbucket's) is skipped
// Bitbucket stops before asking the server: the account's nickname is not necessarily a workspace
func (c *Client) Owner(ctx context.Context, flag string) (string, *cerr.CustomError) {
	ac := c.Connection
	switch {
	case flag != "":
		return flag, nil
	case ac.Owner != "":
		return ac.Owner, nil
	case ac.Username != "" && !strings.Contains(ac.Username, "@"):
		return ac.Username, nil
	case ac.ServerType == connections.ServerTypeBitBucket:
		return "", &cerr.CustomError{Title: "Missing owner",
			Message: fmt.Sprintf("Connection %s: Bitbucket needs a workspace; pass -o WORKSPACE, or set the connection's owner with: gitclt conn edit %s", ac.Name, ac.Name)}
	}

	me, e := c.Directory().Me(ctx)
	if e != nil {
		return "", Error("Unable to find out who owns the repositories (pass -o OWNER)", e)
	}
	return me.Login, nil
}
