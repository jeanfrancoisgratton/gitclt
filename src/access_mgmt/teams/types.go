// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/types.go
// Original timestamp : 2026.10.26 07:30:53

package teams

import (
	"fmt"
	"strings"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

var JSONoutput = false
var AssumeYes = false

// CreateTeamOptions flags
var CreateTeamDescription string
var CreateTeamPermission PermissionFlag
var CreateTeamUnits string
var CreateTeamIncludesAllRepos bool
var CreateTeamCanCreateOrgRepo bool

// EditTeamOptions flags
var EditTeamName string
var EditTeamDescription string
var EditTeamPermission PermissionFlag
var EditTeamUnits string
var EditTeamIncludesAllRepos bool
var EditTeamCanCreateOrgRepo bool

// PermissionFlag is a pflag.Value restricted to the overall team permissions that can be assigned
// (owner is reported by the forge, but cannot be given to a team)
type PermissionFlag gol.TeamPermission

func (p *PermissionFlag) String() string { return string(*p) }
func (p *PermissionFlag) Type() string   { return "read|write|admin" }

func (p *PermissionFlag) Set(s string) error {
	tp := gol.TeamPermission(strings.ToLower(s))
	switch tp {
	case gol.TeamPermissionRead, gol.TeamPermissionWrite, gol.TeamPermissionAdmin:
	default:
		return fmt.Errorf("must be one of read, write or admin")
	}
	*p = PermissionFlag(tp)
	return nil
}
