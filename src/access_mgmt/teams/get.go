// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/get.go
// Original timestamp : 2026.10.26 08:09:48

package teams

import (
	"context"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// TeamGet fetches a team by organization and name (case-insensitive)
func TeamGet(orgname, teamname string, displayoutput bool) (*gol.Team, *cerr.CustomError) {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	if ce = c.RequireAdmin(); ce != nil {
		return nil, ce
	}

	team, ce := teamByName(ctx, c, orgname, teamname)
	if ce != nil {
		return nil, ce
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
