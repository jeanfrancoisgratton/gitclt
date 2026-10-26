// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/helpers.go
// Original timestamp : 2026.10.26 08:02:11

package teams

import (
	"context"
	"fmt"
	"gitclt/forge"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// teamByName resolves <ORG> <TEAM> into the team itself: the library addresses teams by ID, users by name
func teamByName(ctx context.Context, adm gol.Admin, orgname, teamname string) (*gol.Team, *cerr.CustomError) {
	team, e := adm.Teams().GetByName(ctx, orgname, teamname)
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to fetch team %s of %s", teamname, orgname), e)
	}
	return team, nil
}

// confirm asks a yes/no question; anything but y or yes (any case) is a no
func confirm(question string) bool {
	answer := hf.GetStringValFromPrompt(hftx.WarningSign(question + " [y/N]: "))
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes"
}

// parseUnits turns "code=write,issues=read" into a per-unit permission map
// Units can be given with or without their "repo." prefix (code or repo.code), and only take none, read or write
func parseUnits(s string) (map[gol.TeamUnit]gol.TeamPermission, *cerr.CustomError) {
	units := map[gol.TeamUnit]gol.TeamPermission{}
	for _, pair := range strings.Split(s, ",") {
		if pair = strings.TrimSpace(pair); pair == "" {
			continue
		}
		name, perm, found := strings.Cut(pair, "=")
		if !found {
			return nil, &cerr.CustomError{Title: "Invalid unit", Message: fmt.Sprintf("%q: expected UNIT=PERMISSION, e.g. code=write", pair)}
		}
		name = strings.ToLower(strings.TrimSpace(name))
		unit := gol.TeamUnit(name)
		if !strings.HasPrefix(name, "repo.") {
			unit = gol.TeamUnit("repo." + name)
		}
		if !validUnit(unit) {
			return nil, &cerr.CustomError{Title: "Invalid unit", Message: fmt.Sprintf("%q: must be one of %s", name, unitNames())}
		}
		tp := gol.TeamPermission(strings.ToLower(strings.TrimSpace(perm)))
		switch tp {
		case gol.TeamPermissionNone, gol.TeamPermissionRead, gol.TeamPermissionWrite:
		default:
			return nil, &cerr.CustomError{Title: "Invalid unit", Message: fmt.Sprintf("%q: the permission must be one of none, read or write", pair)}
		}
		units[unit] = tp
	}
	return units, nil
}

func validUnit(u gol.TeamUnit) bool {
	for _, known := range gol.AllTeamUnits {
		if u == known {
			return true
		}
	}
	return false
}

// unitNames lists the units without their "repo." prefix, as they are typed on the command line
func unitNames() string {
	names := make([]string, 0, len(gol.AllTeamUnits))
	for _, u := range gol.AllTeamUnits {
		names = append(names, strings.TrimPrefix(string(u), "repo."))
	}
	return strings.Join(names, ", ")
}

// teamAccess builds the access from --permission and --units: either one or the other, never both
// Neither of them gives an empty access, which the callers handle (create defaults to read, edit leaves it alone)
func teamAccess(perm PermissionFlag, unitlist string) (gol.TeamAccess, *cerr.CustomError) {
	var ta gol.TeamAccess
	if perm != "" && strings.TrimSpace(unitlist) != "" {
		return ta, &cerr.CustomError{Title: "Invalid team access", Message: "use either --permission or --units, not both"}
	}
	if perm != "" {
		ta.Permission = gol.TeamPermission(perm)
		return ta, nil
	}
	units, ce := parseUnits(unitlist)
	if ce != nil {
		return ta, ce
	}
	if len(units) > 0 {
		ta.Units = units
	}
	return ta, nil
}
