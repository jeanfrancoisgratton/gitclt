// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/create.go
// Original timestamp : 2026.10.24 10:03:43

package user

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func CreateUser(username string, mustchangepassword *bool) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()
	cuo := gol.CreateUserOptions{
		Login:              username,
		Email:              CreateUserEmail,
		FullName:           CreateUserFullname,
		MustChangePassword: mustchangepassword,
		Visibility:         gol.OwnerVisibility(CreateUserVisibility),
		Restricted:         CreateUserRestrictedUser,
		SendNotify:         CreateUserSendNotification,
	}
	if CreateUserPasswdFile == "" {
		if cuo.MustChangePassword != nil && *cuo.MustChangePassword {
			fmt.Println(hftx.NoteSign("Note: The following password is temporary as the user will need to change it at first login"))
		}
		cuo.Password = hf.GetPassword("Enter the username's password: ", shared.DebugMode)
	} else {
		if pw, e := readPasswordFile(CreateUserPasswdFile); e != nil {
			return &cerr.CustomError{Title: "Unable to read password file", Message: e.Error()}
		} else {
			cuo.Password = pw
		}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	if _, e := adm.Users().Create(ctx, cuo); e != nil {
		return forge.Error("Unable to create the user", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Created user " + hftx.Brown(username)))
	}

	// The create endpoint has no profile fields nor admin flag; set them with an Edit
	var euo gol.EditUserOptions
	for _, f := range []struct {
		val string
		dst **string
	}{{CreateUserDescription, &euo.Description}, {CreateUserWebsite, &euo.Website}, {CreateUserLocation, &euo.Location}} {
		if f.val != "" {
			*f.dst = gol.Ptr(f.val)
		}
	}
	if CreateUserAdmin {
		euo.Admin = gol.Ptr(true)
	}
	if euo != (gol.EditUserOptions{}) {
		return applyUserEdit(ctx, adm, username, euo,
			"User "+username+" was created, but setting its profile/admin fields failed (use gitclt user edit; do not re-run create)")
	}
	return nil
}
