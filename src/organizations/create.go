// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/create.go
// Original timestamp : 2026.10.25 12:20:43

package organizations

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func CreateOrg(orgname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()
	coo := gol.CreateOrgOptions{
		Owner:                     CreateOrgOwner,
		Name:                      orgname,
		FullName:                  CreateOrgFullName,
		Description:               CreateOrgDescription,
		Website:                   CreateOrgWebsite,
		Location:                  CreateOrgLocation,
		Email:                     CreateOrgEmail,
		Visibility:                gol.OwnerVisibility(CreateOrgVisibility),
		RepoAdminChangeTeamAccess: CreateOrgRepoAdminChangeTeamAccess,
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if _, e := adm.Orgs().Create(ctx, coo); e != nil {
		return forge.Error("Unable to create the organization", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Created organization " + hftx.Brown(orgname)))
	}
	return nil
}
