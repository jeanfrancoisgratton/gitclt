// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/list.go
// Original timestamp : 2026.10.24 19:06:48

package organizations

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ListOrgs lists all organization that `username` belongs to (provided the ACLs/tokens allow for it)
func ListOrgs(username string, displayoutput bool) ([]gol.Org, *cerr.CustomError) {
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

	orgs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Org], error) {
		return adm.Orgs().List(ctx, username, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to fetch the organization list", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return orgs, nil
	}

	if JSONoutput {
		return orgs, shared.PrintJSON(orgs, "organization list")
	}

	fmt.Println("\n" + hftx.ScrollSign(hftx.Gold("Managed ?")+" means that repository admins can change team access on the repo\n"))
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Org Name", "Org Full name", "Description", "Visibility", "Managed ?", "URL"})
	for _, o := range orgs {
		t.AppendRow(table.Row{o.ID, o.Name, o.FullName, o.Description,
			string(o.Visibility), o.RepoAdminChangeTeamAccess, o.Website})
	}
	t.SetStyle(table.StyleColoredBlueWhiteOnBlack)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return orgs, nil
}

// ListAllOrgs lists all organization on the forge
func ListAllOrgs(displayoutput bool) ([]gol.Org, *cerr.CustomError) {
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

	orgs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Org], error) {
		return adm.Orgs().ListAll(ctx, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to fetch the organization list", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return orgs, nil
	}

	if JSONoutput {
		return orgs, shared.PrintJSON(orgs, "organization list")
	}

	fmt.Println("\n" + hftx.ScrollSign(hftx.Gold("Managed ?")+" means that repository admins can change team access on the repo\n"))
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Org Name", "Org Full name", "Description", "Visibility", "Managed ?", "URL"})
	for _, o := range orgs {
		t.AppendRow(table.Row{o.ID, o.Name, o.FullName, o.Description,
			string(o.Visibility), o.RepoAdminChangeTeamAccess, o.Website})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return orgs, nil
}
