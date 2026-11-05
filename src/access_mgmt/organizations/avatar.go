// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/organizations/avatar.go
// Original timestamp : 2026.11.05 00:00:00

package organizations

import (
	"bytes"
	"context"
	"fmt"
	"gitclt/access_mgmt/users"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// SetOrgAvatar uploads an image as the avatar of an organization (Forgejo only)
func SetOrgAvatar(orgname, imageFile string) *cerr.CustomError {
	ctx := context.Background()

	img, ce := users.ReadAvatar(imageFile)
	if ce != nil {
		return ce
	}
	avatars, ce := connectOrgAvatars(ctx)
	if ce != nil {
		return ce
	}

	if e := avatars.SetOrgAvatar(ctx, orgname, bytes.NewReader(img)); e != nil {
		return users.AvatarError(e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Avatar of organization " + hftx.Brown(orgname) + " set from " + hftx.Brown(imageFile)))
	}
	return nil
}

// DeleteOrgAvatar restores the default avatar of an organization (Forgejo only)
func DeleteOrgAvatar(orgname string) *cerr.CustomError {
	ctx := context.Background()

	avatars, ce := connectOrgAvatars(ctx)
	if ce != nil {
		return ce
	}

	// The image cannot be read back from the server; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Delete the avatar of organization " + hftx.Brown(orgname) + ", and restore the default one ? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, the avatar of organization " + hftx.Brown(orgname) + " was not deleted")
			return nil
		}
	}

	if e := avatars.DeleteOrgAvatar(ctx, orgname); e != nil {
		return forge.Error("Unable to delete the avatar", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Avatar of organization " + hftx.Brown(orgname) + " deleted"))
	}
	return nil
}

func connectOrgAvatars(ctx context.Context) (gol.OrgAvatars, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, ce
	}
	return c.OrgAvatars()
}
