// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/edit.go
// Original timestamp : 2026.10.26 08:20:02

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

// EditTeam only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditTeam(orgname, teamname string, changed func(string) bool) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	eto, ce := editOptionsFromFlags(changed)
	if ce != nil {
		return ce
	}
	if eto == (gol.EditTeamOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt team edit --help"}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}
	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return ce
	}

	// The owner team cannot be renamed nor have its access changed: the library refuses it (gol.ErrOwnerTeam)
	if _, e := adm.Teams().Edit(ctx, team.ID, eto); e != nil {
		return forge.Error("Unable to edit the team", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated team " + hftx.Brown(team.Name) + " of " + hftx.Brown(orgname)))
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) (gol.EditTeamOptions, *cerr.CustomError) {
	var eto gol.EditTeamOptions

	if changed("name") {
		eto.Name = gol.Ptr(EditTeamName)
	}
	if changed("description") {
		eto.Description = gol.Ptr(EditTeamDescription)
	}
	if changed("all-repos") {
		eto.IncludesAllRepos = gol.Ptr(EditTeamIncludesAllRepos)
	}
	if changed("can-create-repos") {
		eto.CanCreateOrgRepo = gol.Ptr(EditTeamCanCreateOrgRepo)
	}

	if changed("permission") || changed("units") {
		access, ce := teamAccess(EditTeamPermission, EditTeamUnits)
		if ce != nil {
			return eto, ce
		}
		if access.Permission == "" && len(access.Units) == 0 {
			return eto, &cerr.CustomError{Title: "Invalid team access", Message: "--units was given no unit"}
		}
		eto.Access = &access
	}

	return eto, nil
}
