// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/create.go
// Original timestamp : 2026.10.26 08:14:20

package teams

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func CreateTeam(orgname, teamname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// Checked before connecting: a bad --units does not need a round-trip to be refused
	access, ce := teamAccess(CreateTeamPermission, CreateTeamUnits)
	if ce != nil {
		return ce
	}
	// The library requires an access level; read on every unit is the least surprising default
	if access.Permission == "" && len(access.Units) == 0 {
		access.Permission = gol.TeamPermissionRead
	}
	cto := gol.CreateTeamOptions{
		Name:             teamname,
		Description:      CreateTeamDescription,
		Access:           access,
		IncludesAllRepos: CreateTeamIncludesAllRepos,
		CanCreateOrgRepo: CreateTeamCanCreateOrgRepo,
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if _, e := adm.Teams().Create(ctx, orgname, cto); e != nil {
		return forge.Error("Unable to create the team", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Created team " + hftx.Brown(teamname) + " in " + hftx.Brown(orgname)))
	}
	return nil
}
