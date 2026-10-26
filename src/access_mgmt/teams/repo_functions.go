// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/repo_functions.go
// Original timestamp : 2026.10.26 08:40:26

package teams

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListRepos lists the repositories a team has access to
func ListRepos(orgname, teamname string, displayoutput bool) ([]gol.Repo, *cerr.CustomError) {
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

	repolist, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return adm.Teams().Repos(ctx, team.ID, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to fetch the repository list of team %s of %s", teamname, orgname), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return repolist, nil
	}

	if JSONoutput {
		return repolist, shared.PrintJSON(repolist, "team's repository list")
	}

	renderRepoTable(repolist)
	return repolist, nil
}

// AddRepoToTeam gives a team access to a repository of its organization
// (the forge refuses repositories owned by anyone else, so the owner is always the team's organization)
func AddRepoToTeam(orgname, teamname, reponame string) *cerr.CustomError {
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

	if e := adm.Teams().AddRepo(ctx, team.ID, orgname, reponame); e != nil {
		return forge.Error(fmt.Sprintf("Unable to add repository %s/%s to team %s", orgname, reponame, team.Name), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Team " + hftx.Brown(team.Name) + " now has access to " + hftx.Brown(orgname+"/"+reponame)))
	}
	return nil
}

// RemoveRepoFromTeam revokes a team's access to a repository of its organization; the repository itself is kept
func RemoveRepoFromTeam(orgname, teamname, reponame string) *cerr.CustomError {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	// The team's members lose their access to the repo; ask unless --yes was passed
	if !AssumeYes && !confirm("Revoke the access of team "+hftx.Brown(teamname)+" to "+hftx.Brown(orgname+"/"+reponame)+"?") {
		fmt.Println("Aborted, team " + hftx.Brown(teamname) + " still has access to " + hftx.Brown(orgname+"/"+reponame))
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

	if e := adm.Teams().RemoveRepo(ctx, team.ID, orgname, reponame); e != nil {
		return forge.Error(fmt.Sprintf("Unable to remove repository %s/%s from team %s", orgname, reponame, team.Name), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Team " + hftx.Brown(team.Name) + " no longer has access to " + hftx.Brown(orgname+"/"+reponame)))
	}
	return nil
}
