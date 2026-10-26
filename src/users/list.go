// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/users/list.go
// Original timestamp : 2026.10.24 02:28:46

package users

import (
	"context"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListUsers lists every account on the forge, as a site admin sees it (Forgejo only)
func ListUsers(displayOutput bool) ([]gol.User, *cerr.CustomError) {
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

	users, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.User], error) {
		return adm.Users().List(ctx, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to fetch the user list", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return users, nil
	}

	if JSONoutput {
		return users, shared.PrintJSON(users, "user list")
	}

	RenderUserTable(users)
	return users, nil
}
