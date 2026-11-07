// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/milestones.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func milestoneState(s string) (gol.MilestoneState, *cerr.CustomError) {
	state := gol.MilestoneState(s)
	if state != gol.MilestoneStateOpen && state != gol.MilestoneStateClosed && state != gol.MilestoneStateAll {
		return "", &cerr.CustomError{Title: "Invalid state", Message: s + " is not one of open, closed, all"}
	}
	return state, nil
}

// ListMilestones lists the milestones of owner/repo
func ListMilestones(repo string, displayoutput bool) ([]gol.Milestone, *cerr.CustomError) {
	ctx := context.Background()
	state, ce := milestoneState(MilestoneListState)
	if ce != nil {
		return nil, ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Milestone], error) {
		return c.Milestones().List(ctx, owner, repo, state, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the milestones of %s/%s", owner, repo), e)
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "milestone list")
	}
	if shared.QuietOutput {
		for _, m := range list {
			fmt.Println(m.Title)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no %smilestone\n", owner, repo, strings.TrimSuffix(string(state)+" ", "all "))
		return list, nil
	}
	renderMilestoneTable(list)
	return list, nil
}

// GetMilestone shows a milestone of owner/repo, found by title or ID
func GetMilestone(repo, arg string, displayoutput bool) (*gol.Milestone, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	m, ce := findMilestone(ctx, c, owner, repo, arg)
	if ce != nil {
		return nil, ce
	}
	if !displayoutput {
		return m, nil
	}
	if JSONoutput {
		return m, shared.PrintJSON(m, "milestone")
	}
	renderMilestone(m)
	return m, nil
}

// CreateMilestone creates a milestone in owner/repo
func CreateMilestone(repo, title string) *cerr.CustomError {
	ctx := context.Background()
	opts := gol.CreateMilestoneOptions{Title: title, Description: MilestoneDescription}
	if MilestoneDue != "" {
		due, ce := parseDue(MilestoneDue)
		if ce != nil {
			return ce
		}
		opts.DueOn = due
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	m, e := c.Milestones().Create(ctx, owner, repo, opts)
	if e != nil {
		if errors.Is(e, gol.ErrNameTaken) {
			return &cerr.CustomError{Title: "Milestone already exists", Message: fmt.Sprintf("%s/%s already has a milestone %s", owner, repo, title)}
		}
		return forge.Error(fmt.Sprintf("Unable to create the milestone %s in %s/%s", title, owner, repo), e)
	}
	if JSONoutput {
		return shared.PrintJSON(m, "milestone")
	}
	if shared.QuietOutput {
		fmt.Println(m.ID)
		return nil
	}
	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created milestone %s in %s/%s", hftx.Brown(m.Title), owner, repo)))
	return nil
}

var milestoneEditFlags = []string{"title", "description", "due", "state"}

// EditMilestone only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditMilestone(repo, arg string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	if !slices.ContainsFunc(milestoneEditFlags, changed) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt issue milestone edit --help"}
	}
	var opts gol.EditMilestoneOptions
	if changed("title") {
		opts.Title = gol.Ptr(MilestoneTitle)
	}
	if changed("description") {
		opts.Description = gol.Ptr(MilestoneDescription)
	}
	if changed("due") {
		// "" removes it; Forgejo's API cannot, and the library says so
		if MilestoneDue == "" {
			opts.ClearDueOn = true
		} else {
			due, ce := parseDue(MilestoneDue)
			if ce != nil {
				return ce
			}
			opts.DueOn = gol.Ptr(due)
		}
	}
	if changed("state") {
		state, ce := milestoneState(MilestoneState)
		if ce != nil || state == gol.MilestoneStateAll {
			return &cerr.CustomError{Title: "Invalid state", Message: MilestoneState + " is not one of open, closed"}
		}
		opts.State = gol.Ptr(state)
	}

	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	m, ce := findMilestone(ctx, c, owner, repo, arg)
	if ce != nil {
		return ce
	}
	if _, e := c.Milestones().Edit(ctx, owner, repo, m.ID, opts); e != nil {
		switch {
		case errors.Is(e, gol.ErrNameTaken):
			return &cerr.CustomError{Title: "Milestone already exists", Message: fmt.Sprintf("%s/%s already has a milestone %s", owner, repo, MilestoneTitle)}
		case errors.Is(e, gol.ErrNotSupported) && opts.ClearDueOn:
			return &cerr.CustomError{Title: "Unsupported operation", Message: "Forgejo's API cannot remove a due date: the web interface can"}
		}
		return forge.Error("Unable to edit the milestone "+m.Title, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated milestone " + hftx.Brown(m.Title)))
	}
	return nil
}

// DeleteMilestones deletes milestones, after confirmation; every one is checked before asking
// The run stops at the first failure; the milestones deleted before it stay deleted
func DeleteMilestones(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	var targets []*gol.Milestone
	var titles []string
	for _, arg := range unique(args) {
		m, ce := findMilestone(ctx, c, owner, repo, arg)
		if ce != nil {
			return ce
		}
		targets = append(targets, m)
		titles = append(titles, m.Title)
	}
	if !confirm(fmt.Sprintf("Delete milestone(s) %s of %s/%s ? Their issues stay, without a milestone [y/N]: ", strings.Join(titles, ", "), owner, repo)) {
		fmt.Println("Aborted, no milestone is deleted")
		return nil
	}
	for _, m := range targets {
		if e := c.Milestones().Delete(ctx, owner, repo, m.ID); e != nil {
			return forge.Error("Unable to delete the milestone "+m.Title, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted milestone " + hftx.Brown(m.Title)))
		}
	}
	return nil
}
