// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/edit.go
// Original timestamp : 2026.10.25 11:30:55

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

// EditOrg only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditOrg(orgname string, changed func(string) bool) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	eoo := editOptionsFromFlags(changed)
	if eoo == (gol.EditOrgOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt org edit --help"}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if _, e := adm.Orgs().Edit(ctx, orgname, eoo); e != nil {
		return forge.Error("Unable to edit the organization", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated organization " + hftx.Brown(orgname)))
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) gol.EditOrgOptions {
	var eoo gol.EditOrgOptions

	strFlags := []struct {
		name string
		val  string
		dst  **string
	}{
		{"fullname", EditOrgFullName, &eoo.FullName},
		{"description", EditOrgDescription, &eoo.Description},
		{"website", EditOrgWebsite, &eoo.Website},
		{"email", EditOrgEmail, &eoo.Email},
		{"location", EditOrgLocation, &eoo.Location},
	}
	for _, f := range strFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	if changed("visibility") {
		eoo.Visibility = gol.Ptr(gol.OwnerVisibility(EditOrgVisibility))
	}
	if changed("repo-admin-team-access") {
		eoo.RepoAdminChangeTeamAccess = gol.Ptr(EditOrgRepoAdminChangeTeamAccess)
	}

	return eoo
}
