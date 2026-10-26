// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/list.go
// Original timestamp : 2026.10.26 07:31:03

package teams

import (
	"context"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListTeams lists the teams of an organization; Forgejo only shows them to the organization's members (and admins)
func ListTeams(orgname string, displayoutput bool) ([]gol.Team, *cerr.CustomError) {
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

	teamlist, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Team], error) {
		return adm.Teams().List(ctx, orgname, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to fetch the team list of organization "+orgname, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return teamlist, nil
	}

	if JSONoutput {
		return teamlist, shared.PrintJSON(teamlist, "team list")
	}

	renderTeamTable(teamlist)
	return teamlist, nil
}
