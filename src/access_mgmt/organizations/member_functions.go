// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/member_functions.go
// Original timestamp : 2026.10.25 17:26:06

package organizations

import (
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

// ListMembers lists all members of a given organization, plain and simple
func ListMembers(orgname string, displayoutput bool) ([]gol.User, *cerr.CustomError) {
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

	memberlist, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.User], error) {
		return adm.Orgs().Members(ctx, orgname, o)
	})

	if e != nil {
		return nil, forge.Error("Unable to fetch the member list of organization "+orgname, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return memberlist, nil
	}

	if JSONoutput {
		return memberlist, shared.PrintJSON(memberlist, "organization's member list")
	}

	users.RenderUserTable(memberlist)
	return memberlist, nil
}

// IsMemberOfOrg: simple true/false: is user USR member of the organization ORG ?
func IsMemberOfOrg(orgname, usrname string) (bool, *cerr.CustomError) {
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
	b, e := adm.Orgs().IsMember(ctx, orgname, usrname)
	if e != nil {
		return false, forge.Error(fmt.Sprintf("Unable to see if %s is part of %s", usrname, orgname), e)
	}
	// The library answers false on any 404, including a missing organization: tell both cases apart
	if !b {
		if _, e = adm.Orgs().Get(ctx, orgname); e != nil {
			return false, forge.Error("Unable to fetch organization "+orgname, e)
		}
	}

	if !shared.QuietOutput {
		if b {
			fmt.Println(hftx.EnabledSign(hftx.Bronze(usrname) + " is part of " + hftx.Bronze(orgname)))
		} else {
			fmt.Println(hftx.ErrorSign(hftx.Bronze(usrname) + " is not part of " + hftx.Bronze(orgname)))
		}
	}
	return b, nil
}

// RemoveMemberFromOrg: remove a given user from an organization
func RemoveMemberFromOrg(orgname, usrname string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// The user loses access to the organization's repos; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Remove " + hftx.Brown(usrname) + " from " + hftx.Brown(orgname) + "? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, " + hftx.Brown(usrname) + " is still a member of " + hftx.Brown(orgname))
			return nil
		}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return ce
	}
	if e := adm.Orgs().RemoveMember(ctx, orgname, usrname); e != nil {
		return forge.Error(fmt.Sprintf("Unable to remove user %s from %s", usrname, orgname), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Removed " + hftx.Brown(usrname) + " from " + hftx.Brown(orgname)))
	}

	return nil
}
