// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/types.go
// Original timestamp : 2026.10.22 01:43:24

package user

import (
	"fmt"
	"strings"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

var ExtendedInfo = false
var JSONoutput = false
var PurgeUser = false
var AssumeYes = false

// VisibilityFlag is a pflag.Value restricted to the owner visibilities gol accepts
type VisibilityFlag gol.OwnerVisibility

func (v *VisibilityFlag) String() string { return string(*v) }
func (v *VisibilityFlag) Type() string   { return "public|limited|private" }

func (v *VisibilityFlag) Set(s string) error {
	ov := gol.OwnerVisibility(strings.ToLower(s))
	if !ov.Valid() {
		return fmt.Errorf("must be one of public, limited or private")
	}
	*v = VisibilityFlag(ov)
	return nil
}

// CreateUserOptions flags
var CreateUserEmail string
var CreateUserPasswdFile string
var CreateUserFullname string
var CreateUserMustChangePasswd bool
var CreateUserRestrictedUser bool
var CreateUserSendNotification bool
var CreateUserVisibility VisibilityFlag

// Not part of the create endpoint: applied with a follow-up Edit
var CreateUserDescription string
var CreateUserWebsite string
var CreateUserLocation string
var CreateUserAdmin bool

// EditUserOptions flags
var EditUserEmail string
var EditUserFullname string
var EditUserDescription string
var EditUserWebsite string
var EditUserLocation string
var EditUserPasswdFile string
var EditUserPromptPasswd bool
var EditUserMustChangePasswd bool
var EditUserAdmin bool
var EditUserActive bool
var EditUserProhibitLogin bool
var EditUserRestricted bool
var EditUserAllowCreateOrg bool
var EditUserMaxRepos int
var EditUserVisibility VisibilityFlag

/*
type CreateUserOptions struct {
	Login    string `json:"login"` // required
	Email    string `json:"email"` // required
	Password string `json:"-"`
	FullName string `json:"fullName,omitzero"`
	// MustChangePassword forces a password change at first sign-in.
	// Nil means the Forgejo default, which is true.
	MustChangePassword *bool           `json:"mustChangePassword,omitzero"`
	Visibility         OwnerVisibility `json:"visibility,omitzero"` // zero value: backend default
	Restricted         bool            `json:"restricted,omitzero"`
	// SendNotify emails the user about the new account.
	SendNotify bool `json:"sendNotify,omitzero"`
}
*/
