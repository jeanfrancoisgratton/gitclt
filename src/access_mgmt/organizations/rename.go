// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/rename.go
// Original timestamp : 2026.10.25 11:10:53

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

func RenameOrg(oldname, newname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if e := adm.Orgs().Rename(ctx, oldname, newname); e != nil {
		return forge.Error("Unable to rename the organization", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Renamed organization " + hftx.Brown(oldname) + " to " + hftx.Brown(newname)))
	}
	return nil
}
