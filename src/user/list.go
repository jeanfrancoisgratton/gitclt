// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/list.go
// Original timestamp : 2026.10.24 02:28:46

package user

import (
	"context"
	"gitclt/forge"
	"gitclt/shared"
	"os"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// UserList lists every account on the forge, as a site admin sees it (Forgejo only)
func UserList(displayOutput bool) ([]gol.User, *cerr.CustomError) {
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
		return users, printJSON(users, "user list")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Login", "Full name", "E-mail", "Visibility", "Admin", "Active", "Last login",
		"Website", "Location"})
	for _, u := range users {
		t.AppendRow(table.Row{u.ID, u.Login, u.FullName, u.Email, string(u.Visibility),
			u.IsAdmin, u.Active, formatTime(u.LastLogin), u.Website, u.Location})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return users, nil
}
