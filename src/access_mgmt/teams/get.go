// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/get.go
// Original timestamp : 2026.10.26 08:09:48

package teams

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// TeamGet fetches a team by organization and name (case-insensitive)
func TeamGet(orgname, teamname string, displayoutput bool) (*gol.Team, *cerr.CustomError) {
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

	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return nil, ce
	}
	return team, displayTeam(team, displayoutput)
}

// TeamGetByID fetches a team by its ID, which is unique across the forge
func TeamGetByID(id int64, displayoutput bool) (*gol.Team, *cerr.CustomError) {
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

	team, e := adm.Teams().Get(ctx, id)
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to fetch team #%d", id), e)
	}
	return team, displayTeam(team, displayoutput)
}

func displayTeam(team *gol.Team, displayoutput bool) *cerr.CustomError {
	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return nil
	}

	if JSONoutput {
		return shared.PrintJSON(team, "team information")
	}

	printTeam(team)
	return nil
}
