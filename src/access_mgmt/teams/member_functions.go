// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/member_functions.go
// Original timestamp : 2026.10.26 08:31:10

package teams

import (
	"context"
	"fmt"
	"gitclt/access_mgmt/users"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListMembers lists all members of a given team
func ListMembers(orgname, teamname string, displayoutput bool) ([]gol.User, *cerr.CustomError) {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return nil, ce
	}
	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return nil, ce
	}

	memberlist, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.User], error) {
		return adm.Teams().Members(ctx, team.ID, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to fetch the member list of team %s of %s", teamname, orgname), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return memberlist, nil
	}

	if JSONoutput {
		return memberlist, shared.PrintJSON(memberlist, "team's member list")
	}

	users.RenderUserTable(memberlist)
	return memberlist, nil
}

// IsMemberOfTeam: simple true/false: is user USR member of the team TEAM of ORG ?
// A missing org or team is an error (the team is resolved first); a missing user is reported as "not part of"
func IsMemberOfTeam(orgname, teamname, usrname string) (bool, *cerr.CustomError) {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return false, ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return false, ce
	}
	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return false, ce
	}

	b, e := adm.Teams().IsMember(ctx, team.ID, usrname)
	if e != nil {
		return false, forge.Error(fmt.Sprintf("Unable to see if %s is part of %s/%s", usrname, orgname, team.Name), e)
	}

	if !shared.QuietOutput {
		if b {
			fmt.Println(hftx.EnabledSign(hftx.Bronze(usrname) + " is part of " + hftx.Bronze(orgname+"/"+team.Name)))
		} else {
			fmt.Println(hftx.ErrorSign(hftx.Bronze(usrname) + " is not part of " + hftx.Bronze(orgname+"/"+team.Name)))
		}
	}
	return b, nil
}

// AddMemberToTeam: add a given user to a team, which also makes them a member of the organization
func AddMemberToTeam(orgname, teamname, usrname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}
	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return ce
	}

	if e := adm.Teams().AddMember(ctx, team.ID, usrname); e != nil {
		return forge.Error(fmt.Sprintf("Unable to add user %s to %s/%s", usrname, orgname, team.Name), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Added " + hftx.Brown(usrname) + " to " + hftx.Brown(orgname+"/"+team.Name)))
	}
	return nil
}

// RemoveMemberFromTeam: remove a given user from a team; they stay a member of the organization
// if they belong to another of its teams
func RemoveMemberFromTeam(orgname, teamname, usrname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// The user loses the team's access to the repos; ask unless --yes was passed
	if !AssumeYes && !confirm("Remove "+hftx.Brown(usrname)+" from "+hftx.Brown(orgname+"/"+teamname)+"?") {
		fmt.Println("Aborted, " + hftx.Brown(usrname) + " is still a member of " + hftx.Brown(orgname+"/"+teamname))
		return nil
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}
	team, ce := teamByName(ctx, adm, orgname, teamname)
	if ce != nil {
		return ce
	}

	if e := adm.Teams().RemoveMember(ctx, team.ID, usrname); e != nil {
		return forge.Error(fmt.Sprintf("Unable to remove user %s from %s/%s", usrname, orgname, team.Name), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Removed " + hftx.Brown(usrname) + " from " + hftx.Brown(orgname+"/"+team.Name)))
	}
	return nil
}
