// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/get.go
// Original timestamp : 2026.10.24 01:50:46

package user

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"text/tabwriter"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// UserGet fetches another user's account, as a site admin sees it (Forgejo only)
// The base profile is always shown; ExtendedInfo adds the forge-specific fields.
func UserGet(username string, displayOutput bool) (*gol.User, *cerr.CustomError) {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return nil, ce
	}

	u, e := adm.Users().Get(ctx, username)
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to fetch user %s", username), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return u, nil
	}

	if JSONoutput {
		return u, shared.PrintJSON(u, "user information")
	}

	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	printUser(w, u)
	_ = w.Flush()

	return u, nil
}
