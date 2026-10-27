// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/users/wipe.go
// Original timestamp : 2026.10.26 14:02:11

package users

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// What a wipe does to one of the user's organizations
type orgAction int

const (
	orgLeave   orgAction = iota // the user is removed from the org, and so from all of its teams
	orgDelete                   // the user is its only member, and it owns no repos: the org goes too
	orgBlocked                  // the user is its last owner, and the org must stay: the wipe cannot proceed
)

type orgPlan struct {
	name   string
	teams  []string // the user's teams in this org
	action orgAction
	reason string // why the org is deleted, or why it blocks the wipe
}

// WipeUser removes a user from all of their organizations (and thus teams), then deletes the user
// An org is deleted along with the user only if they are its sole member and it owns no repos
// Everything is checked first: if anything blocks the wipe, nothing is changed
func WipeUser(username string) *cerr.CustomError {
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

	// The forge would refuse most of it anyway; better to fail before anything is changed
	me, e := c.Directory().Me(ctx)
	if e != nil {
		return forge.Error("Unable to fetch the current user", e)
	}
	if !me.IsAdmin {
		return &cerr.CustomError{Title: "Admin rights required",
			Message: fmt.Sprintf("%s is not a site administrator; only an administrator can wipe a user", me.Login)}
	}
	if strings.EqualFold(me.Login, username) {
		return &cerr.CustomError{Title: "Cannot wipe yourself",
			Message: fmt.Sprintf("%s is the user tied to this connection's credentials", me.Login)}
	}
	if _, e = adm.Users().Get(ctx, username); e != nil {
		return forge.Error("Unable to fetch user "+username, e)
	}

	plans, blockers, ce := planWipe(ctx, c, adm, username)
	if ce != nil {
		return ce
	}
	if len(blockers) > 0 {
		return &cerr.CustomError{Title: "Cannot wipe user " + username,
			Message: "nothing was changed:\n  - " + strings.Join(blockers, "\n  - ")}
	}

	// The plan is what the confirmation (or the dry run) is about: only --quiet --yes skips it
	if !shared.QuietOutput || !AssumeYes || WipeDryRun {
		printWipePlan(username, plans)
	}
	if WipeDryRun {
		return nil
	}

	// None of this can be undone; ask unless --yes was passed
	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Wipe user " + hftx.Brown(username) + "? [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, user " + hftx.Brown(username) + " was not wiped")
			return nil
		}
	}

	// No rollback: on failure, report what was already done
	var done []string
	fail := func(title string, e error) *cerr.CustomError {
		ce := forge.Error(title, e)
		if len(done) > 0 {
			ce.Message += "\nalready done: " + strings.Join(done, ", ")
		}
		return ce
	}
	for _, p := range plans {
		var step string
		switch p.action {
		case orgLeave:
			if e = adm.Orgs().RemoveMember(ctx, p.name, username); e != nil {
				return fail(fmt.Sprintf("Unable to remove user %s from %s", username, p.name), e)
			}
			step = "removed from " + p.name
			if !shared.QuietOutput {
				fmt.Println(hftx.EnabledSign("Removed " + hftx.Brown(username) + " from " + hftx.Brown(p.name)))
			}
		case orgDelete:
			if e = adm.Orgs().Delete(ctx, p.name); e != nil {
				return fail("Unable to delete organization "+p.name, e)
			}
			step = "deleted org " + p.name
			if !shared.QuietOutput {
				fmt.Println(hftx.EnabledSign("Deleted org " + hftx.Brown(p.name)))
			}
		}
		done = append(done, step)
	}

	if e = adm.Users().Delete(ctx, username, gol.DeleteUserOptions{Purge: PurgeUser}); e != nil {
		return fail("Unable to delete the user", e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted user " + hftx.Brown(username)))
	}
	return nil
}

// planWipe decides the fate of each of the user's organizations, and lists everything that blocks the wipe
func planWipe(ctx context.Context, c *forge.Client, adm gol.Admin, username string) ([]orgPlan, []string, *cerr.CustomError) {
	var plans []orgPlan
	var blockers []string

	orgs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Org], error) {
		return adm.Orgs().List(ctx, username, o)
	})
	if e != nil {
		return nil, nil, forge.Error("Unable to fetch the organizations of user "+username, e)
	}
	for _, org := range orgs {
		p, ce := planOrg(ctx, c, adm, org.Name, username)
		if ce != nil {
			return nil, nil, ce
		}
		if p.action == orgBlocked {
			blockers = append(blockers, p.name+": "+p.reason)
		}
		plans = append(plans, p)
	}

	// Deleting the user's own repos is what --purge is for
	if !PurgeUser {
		repos, ce := listRepos(ctx, c, username)
		if ce != nil {
			return nil, nil, ce
		}
		if len(repos) > 0 {
			blockers = append(blockers, fmt.Sprintf("%s owns %d repositories (%s); pass --purge to delete them along with the user",
				username, len(repos), strings.Join(repos, ", ")))
		}
	}
	return plans, blockers, nil
}

// planOrg decides the fate of one organization
// On Forgejo, the org's members are its teams' members: going through the teams also tells us who the owners are
func planOrg(ctx context.Context, c *forge.Client, adm gol.Admin, orgname, username string) (orgPlan, *cerr.CustomError) {
	p := orgPlan{name: orgname}
	others := map[string]bool{}
	otherOwners := map[string]bool{}
	isOwner := false

	teams, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Team], error) {
		return adm.Teams().List(ctx, orgname, o)
	})
	if e != nil {
		return p, forge.Error("Unable to fetch the teams of organization "+orgname, e)
	}
	for _, team := range teams {
		members, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.User], error) {
			return adm.Teams().Members(ctx, team.ID, o)
		})
		if e != nil {
			return p, forge.Error(fmt.Sprintf("Unable to fetch the member list of team %s of %s", team.Name, orgname), e)
		}
		owners := team.Permission == gol.TeamPermissionOwner
		for _, m := range members {
			login := strings.ToLower(m.Login)
			if strings.EqualFold(m.Login, username) {
				p.teams = append(p.teams, team.Name)
				isOwner = isOwner || owners
				continue
			}
			others[login] = true
			if owners {
				otherOwners[login] = true
			}
		}
	}

	switch {
	case len(others) == 0:
		// "Last man on earth": the org goes only if no repos live in it either
		repos, ce := listRepos(ctx, c, orgname)
		if ce != nil {
			return p, ce
		}
		if len(repos) > 0 {
			p.action = orgBlocked
			p.reason = fmt.Sprintf("%s is its only member, but it still owns %d repositories (%s)",
				username, len(repos), strings.Join(repos, ", "))
		} else {
			p.action = orgDelete
			p.reason = username + " is its only member, and it owns no repositories"
		}
	case isOwner && len(otherOwners) == 0:
		p.action = orgBlocked
		p.reason = fmt.Sprintf("%s is its last owner, and it still has %d other member(s); make one of them an owner first",
			username, len(others))
	default:
		p.action = orgLeave
	}
	return p, nil
}

// listRepos returns the names of the repos owned by a user or an organization
func listRepos(ctx context.Context, c *forge.Client, owner string) ([]string, *cerr.CustomError) {
	repos, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return c.Repos().List(ctx, owner, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to fetch the repositories of "+owner, e)
	}
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}
	return names, nil
}

func printWipePlan(username string, plans []orgPlan) {
	fmt.Println("Wiping user " + hftx.Brown(username) + " will:")
	for _, p := range plans {
		switch p.action {
		case orgLeave:
			line := "  - remove them from " + hftx.Brown(p.name)
			if len(p.teams) > 0 {
				line += " (teams: " + strings.Join(p.teams, ", ") + ")"
			}
			fmt.Println(line)
		case orgDelete:
			fmt.Println("  - delete organization " + hftx.Brown(p.name) + ": " + p.reason)
		}
	}
	line := "  - delete user " + hftx.Brown(username)
	if PurgeUser {
		line += ", and purge all of their content (repositories, packages)"
	}
	fmt.Println(line)
}
