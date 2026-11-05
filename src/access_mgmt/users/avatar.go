// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/users/avatar.go
// Original timestamp : 2026.11.04 21:00:00

package users

import (
	"bytes"
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// Forgejo refuses wider or taller images
const maxAvatarDimension = 4096

// SetAvatar uploads an image as the avatar of the connection's user (Forgejo only)
func SetAvatar(imageFile string) *cerr.CustomError {
	ctx := context.Background()

	img, ce := ReadAvatar(imageFile)
	if ce != nil {
		return ce
	}
	acct, ce := connectAvatar(ctx)
	if ce != nil {
		return ce
	}

	if e := acct.Profile().SetAvatar(ctx, bytes.NewReader(img)); e != nil {
		return AvatarError(e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Avatar set from " + hftx.Brown(imageFile)))
	}
	return nil
}

// DeleteAvatar restores the default avatar of the connection's user (Forgejo only)
func DeleteAvatar() *cerr.CustomError {
	ctx := context.Background()

	acct, ce := connectAvatar(ctx)
	if ce != nil {
		return ce
	}

	// The image cannot be read back from the server
	if !confirmed("Delete your avatar, and restore the default one ? [y/N]: ") {
		fmt.Println("Aborted, the avatar was not deleted")
		return nil
	}

	if e := acct.Profile().DeleteAvatar(ctx); e != nil {
		return forge.Error("Unable to delete the avatar", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Avatar deleted"))
	}
	return nil
}

func connectAvatar(ctx context.Context) (gol.Account, *cerr.CustomError) {
	c, acct, ce := connectAccount(ctx)
	if ce != nil {
		return nil, ce
	}
	if c.Backend() != gol.BackendForgejo {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: avatars can only be managed on Forgejo", c.Connection.Name)}
	}
	return acct, nil
}

// AvatarError converts a failed upload; exported so that any avatar (user, org) reports it the same way
func AvatarError(err error) *cerr.CustomError {
	ce := forge.Error("Unable to set the avatar", err)
	if ce.Code == http.StatusInternalServerError {
		ce.Message += "; Forgejo answers this way to an image it cannot read"
	}
	return ce
}

// ReadAvatar reads and checks an image; exported so that any avatar (user, org) gets the same checks
// They are done here, as Forgejo answers a bare HTTP 500 to an unknown format or an oversized image
// The dimensions can only be read from the formats the standard library knows: the server judges the others (ex: WebP)
func ReadAvatar(path string) ([]byte, *cerr.CustomError) {
	invalid := func(msg string) *cerr.CustomError {
		return &cerr.CustomError{Title: "Invalid avatar", Message: path + ": " + msg}
	}

	info, e := os.Stat(path)
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the avatar", Message: e.Error()}
	}
	if info.Size() > gol.MaxAvatarSize {
		return nil, invalid(fmt.Sprintf("the image is larger than %d MiB", gol.MaxAvatarSize>>20))
	}
	img, e := os.ReadFile(path)
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the avatar", Message: e.Error()}
	}
	if len(img) == 0 {
		return nil, invalid("the file is empty")
	}
	if kind := http.DetectContentType(img); !strings.HasPrefix(kind, "image/") {
		return nil, invalid("not an image (" + kind + ")")
	}
	if cfg, _, e := image.DecodeConfig(bytes.NewReader(img)); e == nil && (cfg.Width > maxAvatarDimension || cfg.Height > maxAvatarDimension) {
		return nil, invalid(fmt.Sprintf("the image is %dx%d, and can be at most %dx%d", cfg.Width, cfg.Height, maxAvatarDimension, maxAvatarDimension))
	}
	return img, nil
}
