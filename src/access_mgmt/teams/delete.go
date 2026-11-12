// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/delete.go
// Original timestamp : 2026.10.26 08:24:45

package teams

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func DeleteTeam(orgname, teamname string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes && !confirm("Delete team "+hftx.Brown(teamname)+" of "+hftx.Brown(orgname)+"?") {
		fmt.Println("Aborted, team " + hftx.Brown(teamname) + " was not deleted")
		return nil
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if ce = c.RequireAdmin(); ce != nil {
		return ce
	}
	team, ce := teamByName(ctx, c, orgname, teamname)
	if ce != nil {
		return ce
	}

	// The owner team cannot be deleted: the library refuses it (gol.ErrOwnerTeam)
	if e := c.Teams().Delete(ctx, orgname, team.Name); e != nil {
		return forge.Error("Unable to delete the team", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted team " + hftx.Brown(team.Name) + " of " + hftx.Brown(orgname)))
	}
	return nil
}
