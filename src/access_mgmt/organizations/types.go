// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/types.go
// Original timestamp : 2026.10.24 19:06:09

package organizations

import "gitclt/shared"

var JSONoutput = false
var AssumeYes = false

// CreateOrgOptions flags
var CreateOrgOwner string
var CreateOrgFullName string
var CreateOrgDescription string
var CreateOrgWebsite string
var CreateOrgEmail string
var CreateOrgLocation string
var CreateOrgVisibility shared.VisibilityFlag
var CreateOrgRepoAdminChangeTeamAccess bool

// EditOrgOptions flags
var EditOrgFullName string
var EditOrgDescription string
var EditOrgWebsite string
var EditOrgEmail string
var EditOrgLocation string
var EditOrgVisibility shared.VisibilityFlag
var EditOrgRepoAdminChangeTeamAccess bool
