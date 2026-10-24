// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/delete.go
// Original timestamp : 2026.10.24 03:16:58

package user

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

func DeleteUser(username string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		prompt := "Delete user " + hftx.Brown(username)
		if PurgeUser {
			prompt += " and purge all of their content (repositories, packages, org memberships)"
		}
		answer := hf.GetStringValFromPrompt(hftx.WarningSign(prompt + " ? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, user " + hftx.Brown(username) + " was not deleted")
			return nil
		}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if e := adm.Users().Delete(ctx, username, gol.DeleteUserOptions{Purge: PurgeUser}); e != nil {
		return forge.Error("Unable to delete the user", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted user " + hftx.Brown(username)))
	}
	return nil
}
