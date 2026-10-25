// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/users/edit.go
// Original timestamp : 2026.10.24 16:05:00

package users

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditUser only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditUser(username string, changed func(string) bool) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	euo, ce := editOptionsFromFlags(changed)
	if ce != nil {
		return ce
	}
	if euo == (gol.EditUserOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt user edit --help"}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}

	return applyUserEdit(ctx, adm, username, euo, "Unable to edit the user")
}

// applyUserEdit is shared with CreateUser, which needs a follow-up Edit for the fields the create endpoint lacks
func applyUserEdit(ctx context.Context, adm gol.Admin, username string, euo gol.EditUserOptions, errTitle string) *cerr.CustomError {
	u, e := adm.Users().Edit(ctx, username, euo)
	if e != nil {
		// Forgejo may ignore a switch back to public visibility, while applying everything else
		if u != nil && errors.Is(e, gol.ErrNotSupported) {
			fmt.Println(hftx.WarningSign("User " + hftx.Brown(username) + " was updated, but: " + e.Error()))
			return nil
		}
		return forge.Error(errTitle, e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated user " + hftx.Brown(username)))
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) (gol.EditUserOptions, *cerr.CustomError) {
	var euo gol.EditUserOptions

	strFlags := []struct {
		name string
		val  string
		dst  **string
	}{
		{"email", EditUserEmail, &euo.Email},
		{"fullname", EditUserFullname, &euo.FullName},
		{"description", EditUserDescription, &euo.Description},
		{"website", EditUserWebsite, &euo.Website},
		{"location", EditUserLocation, &euo.Location},
	}
	for _, f := range strFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	boolFlags := []struct {
		name string
		val  bool
		dst  **bool
	}{
		{"must-change-passwd", EditUserMustChangePasswd, &euo.MustChangePassword},
		{"admin", EditUserAdmin, &euo.Admin},
		{"active", EditUserActive, &euo.Active},
		{"prohibit-login", EditUserProhibitLogin, &euo.ProhibitLogin},
		{"restricted", EditUserRestricted, &euo.Restricted},
		{"allow-create-org", EditUserAllowCreateOrg, &euo.AllowCreateOrganization},
	}
	for _, f := range boolFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	if changed("visibility") {
		euo.Visibility = gol.Ptr(gol.OwnerVisibility(EditUserVisibility))
	}
	if changed("max-repos") {
		if EditUserMaxRepos < -1 {
			return euo, &cerr.CustomError{Title: "Invalid value", Message: "--max-repos must be -1 (instance default) or greater"}
		}
		euo.MaxRepoCreation = gol.Ptr(EditUserMaxRepos)
	}

	switch {
	case changed("password-file"):
		pw, e := readPasswordFile(EditUserPasswdFile)
		if e != nil {
			return euo, &cerr.CustomError{Title: "Unable to read password file", Message: e.Error()}
		}
		euo.Password = &pw
	case changed("passwd"):
		pw := hf.GetPassword("Enter the user's new password: ", shared.DebugMode)
		euo.Password = &pw
	}

	return euo, nil
}
