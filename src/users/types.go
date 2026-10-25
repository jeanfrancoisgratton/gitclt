// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/users/types.go
// Original timestamp : 2026.10.22 01:43:24

package users

import "gitclt/shared"

var ExtendedInfo = false
var JSONoutput = false
var PurgeUser = false
var AssumeYes = false

// CreateUserOptions flags
var CreateUserEmail string
var CreateUserPasswdFile string
var CreateUserFullname string
var CreateUserMustChangePasswd bool
var CreateUserRestrictedUser bool
var CreateUserSendNotification bool
var CreateUserVisibility shared.VisibilityFlag

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
var EditUserVisibility shared.VisibilityFlag
