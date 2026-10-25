// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/delete.go
// Original timestamp : 2026.10.24 23:36:58

package organizations

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func DeleteOrg(orgname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Delete organization " + hftx.Brown(orgname) + "? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, organization " + hftx.Brown(orgname) + " was not deleted")
			return nil
		}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if e := adm.Orgs().Delete(ctx, orgname); e != nil {
		return forge.Error("Unable to delete the organization", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted org " + hftx.Brown(orgname)))
	}
	return nil
}
